// stacks implements the acquisition and release of stacks,
// using name_manager.
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

type ExecConfig struct {
	Name             names.Name
	Dev              bool
	Tail             bool
	Run              []string
	WaitForInterrupt bool
}

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

func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, execCfg *ExecConfig, detachedErrc chan<- error) (err error) {
	if err := deploy.Exec(ctx, cfg, pipeline, execCfg.Name); err != nil {
		return fmt.Errorf("deploy step failed: %v", err)
	}

	ports := k8s.NewPorts(cfg)
	defer ports.CancelForwarding()

	if execCfg.Dev || execCfg.Tail {
		detachedCtx, cancelDetached := context.WithCancel(ctx)
		defer cancelDetached()
		detachedg, detachedCtx := errgroup.WithContext(detachedCtx)

		if execCfg.Dev {
			detachedg.Go(func() error {
				if err := dev.Exec(detachedCtx, cfg, pipeline, execCfg.Name, ports); err != nil {
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
				if err := k8s.Tail(detachedCtx, cfg, execCfg.Name); err != nil {
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

	if err := run.Exec(ctx, cfg, pipeline, execCfg.Name, execCfg.Run, ports); err != nil {
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

func Remove(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, shortName string) error {
	if shortName == "" {
		if pipeline.Stack.Name == "" {
			return errors.New("stack.name is mandatory if not specified otherwise")
		}
		shortName = pipeline.Stack.Name
	}
	name := names.Name{Family: pipeline.Stack.Family, ShortName: shortName}

	if err := deploy.CleanUp(ctx, cfg, pipeline, name); err != nil {
		return err
	}

	return nil
}

func List(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, freeOnly bool) ([]string, error) {
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
