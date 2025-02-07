// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package run implements the "run" step of pipelines.
package run

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"os"
	"strings"
	"time"
)

// Exec runs the commands of a pipeline.
func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	specNames []string,
	k8sClient *k8s.K8s,
) error {
	for _, specName := range specNames {
		var spec *pipelines.Command
		for _, s := range pipeline.Commands {
			if s.Name == specName {
				spec = &s
				break
			}
		}
		if spec == nil {
			return fmt.Errorf("unrecognized run '%s'", specName)
		}

		var extraEnv []string
		if spec.Setup != "" {
			setup, err := pipeline.Setups.Get(spec.Setup)
			if err != nil {
				return err
			}

			if len(setup.Before) > 0 {
				if err := ExecHooks(ctx, cfg, name, specName, setup.Before, nil, k8sClient); err != nil {
					return err
				}
			}

			extraEnv = setup.Env
		}

		err := ExecBaseCommand(
			ctx,
			cfg,
			name,
			specName,
			&spec.BaseCommand,
			extraEnv,
			k8sClient,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExecHooks executes a slice of hooks.  The hooks can depend on
// each other.  They cannot depend on hooks not defined in the slice.
func ExecHooks(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	specName string,
	hooks []pipelines.CommandHook,
	sharedEnv []string,
	k8sClient *k8s.K8s,
) error {
	done := make(map[string]chan struct{})
	for _, hook := range hooks {
		if hook.Name != "" {
			done[hook.Name] = make(chan struct{})
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	for i, hook := range hooks {
		i, hook := i, hook
		g.Go(func() error {
			for _, dep := range hook.DependsOn {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case <-done[dep]:
				}
			}

			var hookCtx context.Context
			if hook.TimeoutSeconds == 0 {
				hookCtx = gctx
			} else {
				var cancel context.CancelFunc
				hookCtx, cancel = context.WithTimeout(gctx, time.Duration(hook.TimeoutSeconds)*time.Second)
				defer cancel()
			}

			var hookID string
			if hook.Name != "" {
				hookID = fmt.Sprintf("#%d(%s)", i, hook.Name)
			} else {
				hookID = fmt.Sprintf("#%d", i)
			}

			execDone := make(chan struct{})
			defer close(execDone)
			go func() {
				for {
					select {
					case <-gctx.Done():
						return
					case <-execDone:
						return
					case <-time.After(10 * time.Second):
						cfg.Logger().Info("run:exec-hooks", "hook %s: still running...", hookID)
					}
				}
			}()
			if err := execHook(hookCtx, cfg, name, specName, i, &hook, sharedEnv, k8sClient); err != nil {
				return fmt.Errorf("hook %s: %s", hookID, err)
			}

			if hook.Name != "" {
				close(done[hook.Name])
				cfg.Logger().Info("run:exec-hooks", "hook %s: success", hookID)
			} else {
				cfg.Logger().Info("run:exec-hooks", "hook %s: success", hookID)
			}
			return nil
		})
	}
	return g.Wait()
}

func execHook(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	specName string,
	i int,
	hook *pipelines.CommandHook,
	sharedEnv []string,
	k8sClient *k8s.K8s,
) error {
	if hook.WaitFor != nil {
		k, err := k8s.New(cfg)
		if err != nil {
			return err
		}
		labelSelector := k8s.Labels{
			k8s.StackLabel: name.DNSName(),
		}.String()
		for _, resource := range hook.WaitFor.Resources {
			switch resource {
			case pipelines.OnePodPerService:
				if err := k.WaitForOnePodPerService(ctx, "default", name); err != nil {
					return err
				}
			case pipelines.Endpoints:
				if err := k.WaitForEndpoints(ctx, "default", name); err != nil {
					return err
				}
			case pipelines.Pods:
				if err := k.WaitForAllPodsRunning(ctx, "default", labelSelector); err != nil {
					return err
				}
			default:
				// invalid specifiers were caught when the pipeline configuration was parsed.
				panic(fmt.Sprintf("invalid waitFor resource specifier: '%s'", resource))
			}
		}
	} else if hook.Run != nil {
		err := ExecBaseCommand(
			ctx,
			cfg,
			name,
			fmt.Sprintf("%s:before(%d)", specName, i),
			hook.Run,
			sharedEnv,
			k8sClient,
		)
		if err != nil {
			return err
		}
	} else if hook.HTTPGet != nil {
		trans := env.NewTransformer(env.K8sTemplateFuncs(cfg, name, k8sClient))
		if err := httpGet(ctx, cfg.Logger(), hook.HTTPGet, trans, time.After); err != nil {
			return err
		}
	}

	return nil
}

// ExecBaseCommand executes a base command.  Base commands can be test commands,
// hooks, batch commands, ...
func ExecBaseCommand(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	specName string,
	spec *pipelines.BaseCommand,
	sharedEnv []string,
	k8sClient *k8s.K8s,
) error {
	if len(spec.Command) == 0 {
		return fmt.Errorf("run '%s': command must at least give the program name", specName)
	}
	cmd := proc.GracefulCommandContext(ctx, spec.Command[0], spec.Command[1:]...)
	if spec.WorkingDir != "" {
		cmd.Dir = cfg.Path(spec.WorkingDir)
	}
	trans := env.NewTransformer(env.K8sTemplateFuncs(cfg, name, k8sClient))
	extraEnv := make([]string, len(sharedEnv)+len(spec.Env))
	g, gctx := errgroup.WithContext(ctx)
	for i, e := range sharedEnv {
		i, e := i, e
		g.Go(func() error {
			ans, err := trans.Get(gctx, e)
			if err != nil {
				return fmt.Errorf("cannot transform env var '%s': %v", e, err)
			}
			extraEnv[i] = ans
			return nil
		})
	}
	for i, e := range spec.Env {
		i, e := i, e
		g.Go(func() error {
			ans, err := trans.Get(gctx, e)
			if err != nil {
				return fmt.Errorf("cannot transform env var '%s': %v", e, err)
			}
			extraEnv[len(sharedEnv)+i] = ans
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	cfg.Logger().Info("run:"+specName+":env", "%s", strings.Join(extraEnv, "\n"))
	cmd.Env = append(os.Environ(), extraEnv...)
	cfg.Logger().Pipe("run:"+specName, cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run '%s': %v", specName, err)
	}
	return nil
}
