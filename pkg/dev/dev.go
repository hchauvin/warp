// dev implements the "dev" step of pipelines.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package dev

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/dev/browsersync"
	"github.com/hchauvin/warp/pkg/dev/ksync"
	"github.com/hchauvin/warp/pkg/dev/portforward"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
)

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	ports *k8s.Ports,
) error {
	g, gctx := errgroup.WithContext(ctx)

	if len(pipeline.Dev.Ksync) > 0 {
		g.Go(func() error {
			return ksync.Exec(gctx, cfg, pipeline, name)
		})
	}

	if len(pipeline.Dev.BrowserSync) > 0 {
		g.Go(func() error {
			return browsersync.Exec(gctx, cfg, pipeline, name, ports)
		})
	}

	if len(pipeline.Dev.PortForward) > 0 {
		g.Go(func() error {
			return portforward.Exec(gctx, cfg, pipeline, name, ports)
		})
	}

	return g.Wait()
}
