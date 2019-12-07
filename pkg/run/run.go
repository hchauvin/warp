// run implements the "run" step of pipelines.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
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

		if len(spec.Before) > 0 {
			if err := ExecHooks(ctx, cfg, name, specName, spec.Before, k8sClient); err != nil {
				return err
			}
		}

		err := ExecBaseCommand(
			ctx,
			cfg,
			name,
			specName,
			&spec.BaseCommand,
			k8sClient,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func ExecHooks(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	specName string,
	hooks []pipelines.CommandHook,
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

			if err := execHook(hookCtx, cfg, name, specName, i, &hook, k8sClient); err != nil {
				return fmt.Errorf("hook #%d: %s", i, err)
			}

			if hook.Name != "" {
				close(done[hook.Name])
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
	k8sClient *k8s.K8s,
) error {
	if hook.WaitFor != nil {
		k, err := k8s.New(cfg)
		if err != nil {
			return err
		}
		for _, resource := range hook.WaitFor.Resources {
			if resource == pipelines.Endpoints {
				if err := k.WaitForEndpoints(ctx, "default", name); err != nil {
					return err
				}
			}
			if resource == pipelines.Pods {
				if err := k.WaitForPods(ctx, "default", name); err != nil {
					return err
				}
			}
		}
	} else if hook.Run != nil {
		err := ExecBaseCommand(
			ctx,
			cfg,
			name,
			fmt.Sprintf("%s:before(%d)", specName, i),
			hook.Run,
			k8sClient,
		)
		if err != nil {
			return err
		}
	} else if hook.HTTPGet != nil {
		if err := httpGet(ctx, cfg, name, hook.HTTPGet, k8sClient); err != nil {
			return err
		}
	}

	return nil
}

func ExecBaseCommand(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	specName string,
	spec *pipelines.BaseCommand,
	k8sClient *k8s.K8s,
) error {
	if len(spec.Command) == 0 {
		return fmt.Errorf("run '%s': command must at least give the program name", specName)
	}
	cmd := proc.GracefulCommandContext(ctx, spec.Command[0], spec.Command[1:]...)
	if spec.WorkingDir != "" {
		cmd.Dir = cfg.Path(spec.WorkingDir)
	}
	trans := env.NewTranformer(cfg, name, k8sClient)
	extraEnv := make([]string, len(spec.Env))
	for i, e := range spec.Env {
		ans, err := trans.Get(ctx, e)
		if err != nil {
			return fmt.Errorf("cannot transform env var '%s': %v", e, err)
		}
		extraEnv[i] = ans
	}
	cfg.Logger().Info("run:"+specName+":env", "%s", strings.Join(extraEnv, "\n"))
	cmd.Env = append(os.Environ(), extraEnv...)
	cfg.Logger().Pipe("run:"+specName, cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run '%s': %v", specName, err)
	}
	return nil
}
