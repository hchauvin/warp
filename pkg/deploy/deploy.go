// deploy implements the "deploy" steps of pipelines.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package deploy

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

const (
	logDomain = "deploy.helm"
)

// Exec executes the "deploy" steps.
func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name, k8sClient *k8s.K8s) error {
	var imageRefs container.ImageRefs
	if pipeline.Deploy.Container != nil {
		var err error
		imageRefs, err = container.Exec(ctx, cfg, pipeline, name)
		if err != nil {
			return fmt.Errorf("deploy.container: %v", err)
		}
	}

	k8sResourcesPath, err := ExpandResources(ctx, cfg, pipeline, name, imageRefs)
	if err != nil {
		return err
	}

	if err := k8sClient.Apply(ctx, k8sResourcesPath, k8s.StackLabel+"="+name.DNSName()); err != nil {
		return err
	}
	return nil
}

// CleanUp cleans up/removes the resources that are created by the deployment steps.
func CleanUp(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name, k8sClient *k8s.K8s) error {
	if err := k8sClient.DeleteAll(ctx, k8s.StackLabel+"="+name.DNSName()); err != nil {
		return err
	}

	if pipeline.Deploy.Container != nil {
		if err := container.CleanUp(ctx, cfg, pipeline, name); err != nil {
			return fmt.Errorf("deploy.kustomize: %v", err)
		}
	}

	return nil
}
