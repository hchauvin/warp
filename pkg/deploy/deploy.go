// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package deploy implements the "deploy" steps of pipelines.
package deploy

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/deploy/helm"
	"github.com/hchauvin/warp/pkg/deploy/kustomize"
	"github.com/hchauvin/warp/pkg/deploy/terraform"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

// Exec executes the "deploy" steps.
func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name, k8sClient *k8s.K8s) error {
	if pipeline.Deploy.Terraform != nil {
		if err := terraform.Exec(ctx, cfg, pipeline, name); err != nil {
			return fmt.Errorf("deploy.terraform: %v", err)
		}
	}

	var refs container.ImageRefs
	if pipeline.Deploy.Container != nil {
		var err error
		refs, err = container.Exec(ctx, cfg, pipeline, name)
		if err != nil {
			return fmt.Errorf("deploy.container: %v", err)
		}
	}

	if pipeline.Deploy.Helm != nil {
		if err := helm.Exec(ctx, cfg, pipeline, name, refs, k8sClient); err != nil {
			return fmt.Errorf("deploy.helm: %v", err)
		}
	}

	if pipeline.Deploy.Kustomize != nil {
		if err := kustomize.Exec(ctx, cfg, pipeline, name, refs, k8sClient); err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}
	}

	return nil
}
