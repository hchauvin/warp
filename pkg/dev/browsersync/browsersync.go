// browsersync implements browser-sync live reload of web pages.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package browsersync

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const logDomain = "dev.browsersync"

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	ports *k8s.Ports,
) error {
	if len(pipeline.Dev.BrowserSync) == 0 {
		return nil
	}

	browserSyncPath, err := cfg.Tools[config.BrowserSync].Resolve()
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, spec := range pipeline.Dev.BrowserSync {
		spec := spec
		g.Go(func() error {
			localPort, err := ports.Port(
				k8s.ServiceSpec{
					Labels: k8s.Labels{
						k8s.StackLabel: name.DNSName(),
					}.String() + "," + spec.K8sProxy.Selector,
				},
				spec.K8sProxy.RemotePort)
			if err != nil {
				return fmt.Errorf("browsersync: cannot proxy k8s port: %v", err)
			}

			args := []string{
				"start",
				"--proxy",
				fmt.Sprintf("127.0.0.1:%d", localPort),
				"--port",
				strconv.Itoa(spec.LocalPort),
			}
			for _, path := range spec.Paths {
				args = append(args, "--files", cfg.Path(path))
			}
			if spec.Config != nil {
				b, err := json.MarshalIndent(spec.Config, "", "  ")
				if err != nil {
					return fmt.Errorf("cannot marshal browser-sync config: %v", err)
				}
				configPath := filepath.Join(cfg.Path(cfg.OutputRoot), "browsersync", name.String(), "config.json")
				if err := os.MkdirAll(filepath.Dir(configPath), 0777); err != nil {
					return fmt.Errorf("cannot create parent directory for browser-sync config: %v", err)
				}
				if err := ioutil.WriteFile(configPath, b, 0777); err != nil {
					return fmt.Errorf("cannot write browser-sync config: %v", err)
				}
				args = append(args, "-c", configPath)
			}
			cmd := proc.GracefulCommandContext(gctx, browserSyncPath, args...)
			cfg.Logger().Pipe(logDomain, cmd)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("browser-sync: %v", err)
			}
			return nil
		})
	}

	return g.Wait()
}
