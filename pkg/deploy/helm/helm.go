// helm implements kustomize-based Kubernetes deployment.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package helm

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"os"
	"path/filepath"
)

const (
	logDomain = "deploy.helm"
)

// Exec deploys a stack on Kubernetes using a Helm chart.
func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	imageRefs container.ImageRefs,
	k8sClient *k8s.K8s,
) error {
	k8sResourcesPath, err := ExpandResources(ctx, cfg, pipeline, name, imageRefs)
	if err != nil {
		return err
	}

	var labelSelector string
	h := pipeline.Deploy.Helm
	if h.LabelSelector != "" {
		labelSelector = h.LabelSelector
	} else {
		labelSelector = k8s.StackLabel + "=" + name.DNSName()
	}

	if err := k8sClient.Apply(ctx, k8sResourcesPath, labelSelector); err != nil {
		return err
	}
	return nil
}

func ExpandResources(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	imageRefs container.ImageRefs,
) (k8sResourcesPath string, err error) {
	// TODO: Use imageRefs
	h := pipeline.Deploy.Helm

	outputFolderPath := filepath.Join(cfg.Path(cfg.OutputRoot), "helm", name.String())
	if err := os.MkdirAll(outputFolderPath, 0777); err != nil {
		return "", err
	}

	args := []string{"template"}
	for _, arg := range h.Args {
		args = append(args, arg)
	}
	args = append(args, h.Path)

	helmPath, err := cfg.ToolPath(config.Helm)
	if err != nil {
		return "", err
	}

	k8sResourcesPath = filepath.Join(outputFolderPath, "expanded_resources.yml")
	f, err := os.OpenFile(k8sResourcesPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return "", err
	}
	defer f.Close()

	cmd := proc.GracefulCommandContext(ctx, helmPath, args...)
	cmd.Stdout = f
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Errorf("could not pipe command stderr: %v", err))
	}
	cfg.Logger().PipeReader(config.Kustomize.LogDomain(), stderr)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not run helm on '%s': %v", h.Path, err)
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	cfg.Logger().Info(logDomain, "helm chart expanded to '%s'", k8sResourcesPath)

	return k8sResourcesPath, nil
}

// CleanUp cleans up/removes all the Kubernetes resources created during a Kustomization
// deployment.
func CleanUp(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	k8sClient *k8s.K8s,
) error {
	return k8sClient.DeleteAll(ctx, k8s.StackLabel+"="+name.DNSName())
}
