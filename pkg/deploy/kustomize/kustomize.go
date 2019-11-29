// kustomize implements kustomize-based Kubernetes deployment.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package kustomize

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	logDomain = "deploy.kustomize"
)

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	imageRefs container.ImageRefs,
) error {
	dnsName := name.DNSName()
	k := pipeline.Deploy.Kustomize

	overlayFolderPath := filepath.Join(cfg.Path(cfg.OutputRoot), "kustomize", name.String())
	if err := os.MkdirAll(overlayFolderPath, 0777); err != nil {
		return err
	}

	relativeBase, err := filepath.Rel(overlayFolderPath, cfg.Path(pipeline.Deploy.Kustomize.Path))
	if err != nil {
		return err
	}

	type m map[string]interface{}
	overlay := m{
		"resources": []string{relativeBase},
		"commonLabels": m{
			k8s.StackLabel: dnsName,
		},
	}
	if !k.DisableNamePrefix {
		overlay["namePrefix"] = dnsName + "-"
	}
	if imageRefs != nil {
		images := make([]m, 0, len(imageRefs))
		for k, v := range imageRefs {
			if v == "" {
				continue
			}
			image := m{"name": k}
			parts := strings.Split(v, ":")
			if len(parts) == 1 {
				image["newName"] = parts[0]
				image["newTag"] = "latest"
			} else if len(parts) == 2 {
				image["newName"] = parts[0]
				image["newTag"] = parts[1]
			} else {
				return fmt.Errorf("invalid image ref '%s'", v)
			}
			images = append(images, image)
		}
		overlay["images"] = images
	}

	overlayYaml, err := yaml.Marshal(overlay)
	if err != nil {
		return fmt.Errorf("could not marshal overlay to Yaml: %v", err)
	}

	overlayPath := filepath.Join(overlayFolderPath, "kustomization.yml")
	err = ioutil.WriteFile(
		overlayPath,
		overlayYaml,
		0777)
	if err != nil {
		return fmt.Errorf("could not write kustomization overlay '%s': %v", overlayPath, err)
	}

	kustomizePath, err := cfg.Tools[config.Kustomize].Resolve()
	if err != nil {
		return err
	}
	k8sResourcesPath := filepath.Join(overlayFolderPath, "expanded_resources.yml")
	cmd := exec.CommandContext(ctx, kustomizePath, "build", overlayFolderPath, "-o", k8sResourcesPath)
	cfg.Logger().Pipe(config.Kustomize.LogDomain(), cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run kustomize on '%s': %v", overlayFolderPath, err)
	}

	cfg.Logger().Info(logDomain, "kustomization expanded to '%s'", k8sResourcesPath)

	kubectlPath, err := cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return err
	}
	cmd = exec.CommandContext(ctx, kubectlPath, "apply",
		"-f", k8sResourcesPath,
		"--force",
		"--prune", "-l",
		k8s.StackLabel+"="+dnsName,
	)
	cfg.Logger().Pipe(config.Kubectl.LogDomain(), cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not apply kubectl patch '%s': %v", k8sResourcesPath, err)
	}

	return nil
}

func CleanUp(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) error {
	kubectlPath, err := cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, kubectlPath, "delete",
		"all",
		"-l", k8s.StackLabel+"="+name.DNSName(),
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not delete k8s resources: %v", err)
	}
	return nil
}
