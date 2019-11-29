// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package deploy

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/deploy/kustomize"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) error {
	var refs container.ImageRefs
	if pipeline.Deploy.Container != nil {
		var err error
		refs, err = container.Exec(ctx, cfg, pipeline, name)
		if err != nil {
			return fmt.Errorf("deploy.container: %v", err)
		}
	}

	if pipeline.Deploy.Kustomize != nil {
		if err := kustomize.Exec(ctx, cfg, pipeline, name, refs); err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}
	}

	return nil
}

func CleanUp(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) error {
	if pipeline.Deploy.Kustomize != nil {
		if err := kustomize.CleanUp(ctx, cfg, pipeline, name); err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}
	}

	if pipeline.Deploy.Container != nil {
		if err := container.CleanUp(ctx, cfg, pipeline, name); err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}
	}

	return nil
}
