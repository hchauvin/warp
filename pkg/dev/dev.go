// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package dev implements the "dev" steps of pipelines.
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

// PatchPipeline patches a pipeline depending on the "dev" steps.
func PatchPipeline(cfg *config.Config, setup *pipelines.Setup, pipeline *pipelines.Pipeline) error {
	return ksync.PatchSetup(cfg, setup, pipeline)
}

// Exec executes the "dev" steps.
func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	setupName string,
	k8sClient *k8s.K8s,
) error {
	setup, err := pipeline.Setups.Get(setupName)
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)

	if len(setup.Dev.Ksync) > 0 {
		g.Go(func() error {
			return ksync.Exec(gctx, cfg, setup.Dev.Ksync, name, k8sClient)
		})
	}

	if len(setup.Dev.BrowserSync) > 0 {
		g.Go(func() error {
			return browsersync.Exec(gctx, cfg, setup.Dev.BrowserSync, name, k8sClient)
		})
	}

	if len(setup.Dev.PortForward) > 0 {
		g.Go(func() error {
			return portforward.Exec(gctx, cfg, setup.Dev.PortForward, name, k8sClient)
		})
	}

	return g.Wait()
}
