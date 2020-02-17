// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package stacks implements the acquisition and release of stacks, using name_manager.
package stacks

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/terraform"
	"go.uber.org/atomic"

	// Registers the local backend
	_ "github.com/hchauvin/name_manager/pkg/local_backend"
	// Registers the mongo backend
	_ "github.com/hchauvin/name_manager/pkg/firestore_backend"
	_ "github.com/hchauvin/name_manager/pkg/mongo_backend"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy"
	"github.com/hchauvin/warp/pkg/dev"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/run"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"os"
	"os/signal"
)

// Hold uses name_manager to hold a stack for the given pipeline.
func Hold(cfg *config.Config, pipeline *pipelines.Pipeline) (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
	if pipeline.Stack.Name != "" {
		errc := make(chan error)
		release := func() error {
			close(errc)
			return nil
		}
		return &names.Name{ShortName: pipeline.Stack.Name}, errc, release, nil
	} else if pipeline.Stack.Family == "" {
		// This should be dealt with when parsing the pipeline
		panic("either stack.name or stack.family must be given")
	} else {
		nameManager, err := name_manager.CreateFromURL(cfg.NameManagerURL)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot create name manager: %v", err)
		}
		shortName, errc, release, err := nameManager.Hold(pipeline.Stack.Family)
		if err != nil {
			return nil, nil, nil, err
		}
		return &names.Name{Family: pipeline.Stack.Family, ShortName: shortName}, errc, release, nil
	}
}

// ExecConfig is the configuration for the Exec function.
type ExecConfig struct {
	Name             names.Name
	Dev              bool
	Tail             bool
	Run              []string
	Setup            string
	DumpEnv          string
	PersistEnv       bool
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
	detachedErrcClosed := atomic.NewBool(false)
	defer func() {
		if !detachedErrcClosed.Swap(true) {
			close(detachedErrc)
		}
	}()

	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	var setup *pipelines.Setup
	if execCfg.Setup != "" {
		s, err := pipeline.Setups.Get(execCfg.Setup)
		if err != nil {
			return err
		}
		setup = s

		if err := dev.PatchPipeline(cfg, setup, pipeline); err != nil {
			return fmt.Errorf("cannot patch pipeline according to dev step: %v", err)
		}
	}

	deploymentInfo, err := deploy.Exec(ctx, cfg, pipeline, execCfg.Name, k8sClient)
	if err != nil {
		return fmt.Errorf("deploy step failed: %v", err)
	}

	if execCfg.Dev || execCfg.Tail {
		detachedCtx, cancelDetached := context.WithCancel(ctx)
		defer cancelDetached()
		detachedg, detachedCtx := errgroup.WithContext(detachedCtx)

		if execCfg.Dev && execCfg.Setup != "" {
			detachedg.Go(func() error {
				if err := dev.Exec(detachedCtx, cfg, pipeline, execCfg.Name, execCfg.Setup, k8sClient); err != nil {
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
			if err := detachedg.Wait(); err != nil && err != context.Canceled {
				detachedErrc <- err
			}
			if !detachedErrcClosed.Swap(true) {
				close(detachedErrc)
			}
		}()
	} else {
		if !detachedErrcClosed.Swap(true) {
			close(detachedErrc)
		}
	}

	if setup != nil {
		err = run.ExecHooks(
			ctx,
			cfg,
			execCfg.Name,
			"before",
			setup.Before,
			nil,
			k8sClient)
		if err != nil {
			return err
		}

		envVars := make([]string, len(setup.Env))
		funcs := []env.TemplateFuncs{env.StackTemplateFuncs(cfg, execCfg.Name)}
		funcs = append(funcs, env.K8sTemplateFuncs(cfg, execCfg.Name, k8sClient))
		if deploymentInfo.TerraformRootModulePath != "" {
			tf, err := terraform.New(cfg, deploymentInfo.TerraformRootModulePath)
			if err != nil {
				return err
			}
			funcs = append(funcs, env.TerraformTemplateFuncs(cfg, tf))
		}
		trans := env.NewTransformer(funcs...)
		g, gctx := errgroup.WithContext(ctx)
		for i, envTpl := range setup.Env {
			i, envTpl := i, envTpl
			g.Go(func() error {
				s, err := trans.Get(gctx, envTpl)
				if err != nil {
					return err
				}
				envVars[i] = s
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}

		if execCfg.DumpEnv != "" {
			var content bytes.Buffer
			content.WriteString("# Auto-generated by warp - DO NOT EDIT\n")
			for _, e := range envVars {
				content.WriteString(e)
				content.WriteRune('\n')
			}
			if err := ioutil.WriteFile(cfg.Path(execCfg.DumpEnv), content.Bytes(), 0777); err != nil {
				return err
			}
			if !execCfg.PersistEnv {
				defer func() {
					if err := os.Remove(cfg.Path(execCfg.DumpEnv)); err != nil {
						cfg.Logger().Error("could not remove dumped env file '%s': %v", cfg.Path(execCfg.DumpEnv), err)
					}
				}()
			}
		}
	}

	if err := run.Exec(ctx, cfg, pipeline, execCfg.Name, execCfg.Run, k8sClient); err != nil {
		return err
	}

	if execCfg.WaitForInterrupt {
		fmt.Printf("[Press Ctl-C to exit]\n")
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
