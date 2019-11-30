// ksync implements file synchronization between the local file system
// and Kubernetes pods.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package ksync

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"strings"
)

const logDomain = "dev.ksync"

func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) error {
	ksyncPath, err := cfg.Tools[config.Ksync].Resolve()
	if err != nil {
		return err
	}

	var createg errgroup.Group
	for _, e := range pipeline.Dev.Ksync {
		e := e
		createg.Go(func() error {
			args := []string{
				"create",
				"--name",
				e.Name,
				"--force",
				"--selector",
				k8s.Labels{
					k8s.StackLabel: name.DNSName(),
				}.String() + "," + e.Selector,
			}
			if e.LocalReadOnly {
				args = append(args, "--local-read-only")
			}
			if e.RemoteReadOnly {
				args = append(args, "--remote-read-only")
			}
			if e.DisableReloading {
				args = append(args, "--reload=false")
			}
			args = append(args, cfg.Path(e.Local), e.Remote)
			cfg.Logger().Info(logDomain, strings.Join(args, " "))
			cmd := proc.GracefulCommandContext(ctx, ksyncPath, args...)
			cfg.Logger().Pipe(config.Ksync.LogDomain(), cmd)
			return cmd.Run()
		})
	}

	if err := createg.Wait(); err != nil {
		return fmt.Errorf("cannot create ksync entries: %v", err)
	}

	startWatcher := false
	cmd := proc.GracefulCommandContext(ctx, ksyncPath, "get")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			startWatcher = true
		} else {
			return fmt.Errorf("cannot invoke 'ksync get': %v", err)
		}
	}

	if startWatcher {
		// syncthing, daemonized by ksync watch, sometimes is not properly
		// killed.
		if err := proc.KillPort(8384); err != nil {
			return err
		}

		cmd := proc.GracefulCommandContext(ctx, ksyncPath, "watch")
		cfg.Logger().Pipe(logDomain, cmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("'ksync watch' failed: %v", err)
		}
	}
	return nil
}
