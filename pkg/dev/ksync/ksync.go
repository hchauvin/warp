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
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"runtime"
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
			cmd := exec.CommandContext(ctx, ksyncPath, args...)
			cfg.Logger().Pipe(config.Ksync.LogDomain(), cmd)
			return cmd.Run()
		})
	}

	if err := createg.Wait(); err != nil {
		return fmt.Errorf("cannot create ksync entries: %v", err)
	}

	startWatcher := false
	cmd := exec.CommandContext(ctx, ksyncPath, "get")
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
		if err := killPort(8384); err != nil {
			return err
		}

		cmd := exec.CommandContext(ctx, ksyncPath, "watch")
		cfg.Logger().Pipe(logDomain, cmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("'ksync watch' failed: %v", err)
		}
	}
	return nil
}

func killPort(port int) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		command := fmt.Sprintf("(Get-NetTCPConnection -LocalPort %d).OwningProcess -Force", port)
		cmd = exec.Command("Stop-Process", "-Id", command)
	} else {
		command := fmt.Sprintf("lsof -i tcp:%d | grep LISTEN | awk '{print $2}' | xargs kill -9", port)
		cmd = exec.Command("bash", "-c", command)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kill port: %v: %s", err, out)
	}
	return nil
}
