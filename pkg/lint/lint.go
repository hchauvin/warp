// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package lint

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/deploy/helm"
	"github.com/hchauvin/warp/pkg/deploy/kustomize"
	"github.com/hchauvin/warp/pkg/lint/kubescore"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

func Lint(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline) error {
	var name names.Name
	if pipeline.Stack.Family != "" {
		name = names.Name{Family: pipeline.Stack.Family, ShortName: "lint"}
	} else {
		name = names.Name{ShortName: pipeline.Stack.Name}
	}
	if pipeline.Stack.Variant != "" {
		name.ShortName += "-" + pipeline.Stack.Variant
	}

	var refs container.ImageRefs
	if pipeline.Deploy.Container != nil {
		var err error
		refs, err = container.Exec(ctx, cfg, pipeline, name)
		if err != nil {
			return fmt.Errorf("deploy.container: %v", err)
		}
	}

	if pipeline.Deploy.Helm != nil && !pipeline.Lint.DisableHelmKubeScore {
		k8sResourcesPath, err := helm.ExpandResources(ctx, cfg, pipeline, name, refs)
		if err != nil {
			return fmt.Errorf("deploy.helm: %v", err)
		}

		if err := kubescore.Lint(ctx, cfg, k8sResourcesPath); err != nil {
			return err
		}
	}

	if pipeline.Deploy.Kustomize != nil && !pipeline.Lint.DisableKustomizeKubeScore {
		k8sResourcesPath, err := kustomize.ExpandResources(ctx, cfg, pipeline, name, refs)
		if err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}

		if err := kubescore.Lint(ctx, cfg, k8sResourcesPath); err != nil {
			return err
		}
	}

	return nil
}
