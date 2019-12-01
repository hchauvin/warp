// stacks implements the acquisition and release of stacks, using name_manager.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package stacks

import (
	"context"
	"errors"
	"fmt"
	_ "github.com/hchauvin/name_manager/pkg/local_backend"
	_ "github.com/hchauvin/name_manager/pkg/mongo_backend"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy"
	"github.com/hchauvin/warp/pkg/dev"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/run"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
)

func Hold(cfg *config.Config, pipeline *pipelines.Pipeline) (*names.Name, name_manager.ReleaseFunc, error) {
	if pipeline.Stack.Name != "" {
		return &names.Name{ShortName: pipeline.Stack.Name}, func() error { return nil }, nil
	} else if pipeline.Stack.Family == "" {
		return nil, nil, errors.New("either stack.name or stack.family must be given")
	} else {
		nameManager, err := name_manager.CreateFromURL(cfg.NameManagerURL)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot create name manager: %v", err)
		}
		shortName, release, err := nameManager.Hold(pipeline.Stack.Family)
		if err != nil {
			return nil, nil, err
		}
		return &names.Name{Family: pipeline.Stack.Family, ShortName: shortName}, release, nil
	}
}

// ExecConfig is the configuration for the Exec function.
type ExecConfig struct {
	Name             names.Name
	Dev              bool
	Tail             bool
	Run              []string
	WaitForInterrupt bool
}

// Exec executes the stages in a pipeline.  Detached errors (of goroutines that
// execute in the background) are reported in a channel.
func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	execCfg *ExecConfig,
	detachedErrc chan<- error,
) (err error) {
	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	if err := deploy.Exec(ctx, cfg, pipeline, execCfg.Name, k8sClient); err != nil {
		return fmt.Errorf("deploy step failed: %v", err)
	}

	if execCfg.Dev || execCfg.Tail {
		detachedCtx, cancelDetached := context.WithCancel(ctx)
		defer cancelDetached()
		detachedg, detachedCtx := errgroup.WithContext(detachedCtx)

		if execCfg.Dev {
			detachedg.Go(func() error {
				if err := dev.Exec(detachedCtx, cfg, pipeline, execCfg.Name, k8sClient); err != nil {
					if err == context.Canceled {
						return err
					}
					return fmt.Errorf("dev step failed: %v", err)
				}
				return nil
			})
		}

		if execCfg.Tail {
			detachedg.Go(func() error {
				if err := k8sClient.Tail(detachedCtx, cfg, execCfg.Name); err != nil {
					if err == context.Canceled {
						return err
					}
					return fmt.Errorf("log tailing failed: %v", err)
				}
				return nil
			})
		}

		go func() {
			if err := detachedg.Wait(); err != nil {
				detachedErrc <- err
			}
		}()
	}

	if err := run.Exec(ctx, cfg, pipeline, execCfg.Name, execCfg.Run, k8sClient); err != nil {
		return err
	}

	if execCfg.WaitForInterrupt {
		fmt.Printf("[Press Ctl-C to exit]")
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
		}
	}

	return nil
}

// Remove removes the stack with the given short name.  The short name (combined with the family name
// to give the full name) is optional if the stack does not come with a family name but instead
// with one single name.
func Remove(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, shortName string) error {
	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	if shortName == "" {
		if pipeline.Stack.Name == "" {
			return errors.New("stack.name is mandatory if not specified otherwise")
		}
		shortName = pipeline.Stack.Name
	}
	name := names.Name{Family: pipeline.Stack.Family, ShortName: shortName}

	if err := deploy.CleanUp(ctx, cfg, pipeline, name, k8sClient); err != nil {
		return err
	}

	return nil
}

// List lists all the stack names for a pipeline.  If freeOnly is true, only the
// stacks that are not currently in use are returned.  Otherwise, all the stack anems
// are returned.
func List(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	freeOnly bool,
) ([]string, error) {
	if pipeline.Stack.Name != "" {
		return []string{pipeline.Stack.Name}, nil
	}

	nameManager, err := name_manager.CreateFromURL(cfg.NameManagerURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create name manager: %v", err)
	}
	allNames, err := nameManager.List()
	if err != nil {
		return nil, err
	}

	var shortNames []string
	for _, name := range allNames {
		if name.Family != pipeline.Stack.Family {
			continue
		}
		if freeOnly && !name.Free {
			continue
		}
		shortNames = append(shortNames, name.Name)
	}
	return shortNames, nil
}
