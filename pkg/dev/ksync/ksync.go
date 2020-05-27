// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package ksync implements file synchronization between the local file system
// and Kubernetes pods.
package ksync

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/templates"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"strings"
	"text/template"
)

const logDomain = "dev.ksync"

const deploymentPatchTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: __NAME__
spec:
  template:
    metadata:
      labels:
        __LABEL__: __VALUE__
`

// PatchSetup patches the pipeline definition given a ksync dev setup.
func PatchSetup(cfg *config.Config, setup *pipelines.Setup, pipeline *pipelines.Pipeline) error {
	for _, e := range setup.Dev.Ksync {
		patch := deploymentPatchTemplate
		patch = strings.ReplaceAll(patch, "__NAME__", e.DeploymentName)
		patch = strings.ReplaceAll(patch, "__LABEL__", k8s.RunIDLabel)
		patch = strings.ReplaceAll(patch, "__VALUE__", cfg.RunID)

		pipeline.Deploy.Kustomize.PatchesStrategicMerge = append(
			pipeline.Deploy.Kustomize.PatchesStrategicMerge,
			patch,
		)
	}

	return nil
}

// Exec sets up file synchronization with ksync.
func Exec(ctx context.Context, cfg *config.Config, ksync []pipelines.Ksync, name names.Name, k8sClient *k8s.K8s) error {
	ksyncPath, err := cfg.ToolPath(config.Ksync)
	if err != nil {
		return err
	}

	var createg errgroup.Group
	for _, e := range ksync {
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
					k8s.RunIDLabel: cfg.RunID,
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
			local, err := expandTemplate(e.Local)
			if err != nil {
				return fmt.Errorf("could not expand template 'local': %v", err)
			}
			args = append(args, cfg.Path(local), e.Remote)
			cfg.Logger().Info(logDomain, strings.Join(args, " "))
			cmd, err := k8sClient.KubectlLikeCommandContext(ctx, ksyncPath, args...)
			if err != nil {
				return err
			}
			cfg.Logger().Pipe(config.Ksync.LogDomain(), cmd)
			if err := cmd.Run(); err != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					return err
				}
			}
			return nil
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
			cfg.Logger().Error(logDomain, "cannot kill syncthing port: %v", err)
		}

		cmd := proc.GracefulCommandContext(ctx, ksyncPath, "watch")
		cfg.Logger().Pipe(logDomain, cmd)
		if err := cmd.Run(); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return fmt.Errorf("'ksync watch' failed: %v", err)
			}
		}
	}
	return nil
}

func expandTemplate(tplStr string) (string, error) {
	tpl, err := template.New("config").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templates.TxtFuncMap()).
		Parse(tplStr)
	if err != nil {
		return "", err
	}
	data := map[string]interface{}{}
	w := &bytes.Buffer{}
	if err := tpl.Execute(w, data); err != nil {
		return "", fmt.Errorf("cannot expand template <<< %s >>>: %v", tplStr, err)
	}
	return w.String(), nil
}
