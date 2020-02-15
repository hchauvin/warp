// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package kustomize implements kustomize-based Kubernetes deployment.
package kustomize

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy/container"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/terraform"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	logDomain = "deploy.kustomize"
)

// Exec deploys a stack on Kubernetes using a Kustomization configuration.
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

	return k8sClient.Apply(ctx, k8sResourcesPath, k8s.StackLabel+"="+name.DNSName())
}

// ExpandResources expands the resources defined in a kustomization
// into a YAML file, with one resource per YAML document.  The
// path to this file is returned.
func ExpandResources(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	imageRefs container.ImageRefs,
) (k8sResourcesPath string, err error) {
	dnsName := name.DNSName()
	k := pipeline.Deploy.Kustomize

	overlayFolderPath := filepath.Join(cfg.Path(cfg.OutputRoot), "kustomize", name.String())
	if err := os.MkdirAll(overlayFolderPath, 0777); err != nil {
		return "", err
	}

	relativeBase, err := filepath.Rel(overlayFolderPath, cfg.Path(pipeline.Deploy.Kustomize.Path))
	if err != nil {
		return "", err
	}

	type m map[string]interface{}
	overlay := m{
		"resources": []string{relativeBase},
		"commonLabels": m{
			k8s.StackLabel: dnsName,
		},
		"patchesStrategicMerge": pipeline.Deploy.Kustomize.PatchesStrategicMerge,
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
				return "", fmt.Errorf("invalid image ref '%s'", v)
			}
			images = append(images, image)
		}
		overlay["images"] = images
	}
	if len(k.SecretGenerator) > 0 {
		secretGenerator := make([]map[string]interface{}, len(k.SecretGenerator))
		tf := terraform.New(cfg)
		trans := env.NewTransformer(
			env.StackTemplateFuncs(cfg, name),
			env.TerraformTemplateFuncs(cfg, tf))
		for i, gen := range k.SecretGenerator {
			literals := make([]string, len(gen.Literals))
			for i, tpl := range gen.Literals {
				lit, err := trans.Get(ctx, tpl)
				if err != nil {
					return "", err
				}
				literals[i] = lit
			}
			secretGenerator[i] = map[string]interface{}{
				"name": gen.Name,
				"literals": literals,
			}
		}
		overlay["secretGenerator"] = secretGenerator
	}

	overlayYaml, err := yaml.Marshal(overlay)
	if err != nil {
		return "", fmt.Errorf("could not marshal overlay to Yaml: %v", err)
	}

	overlayPath := filepath.Join(overlayFolderPath, "kustomization.yml")
	err = ioutil.WriteFile(
		overlayPath,
		overlayYaml,
		0777)
	if err != nil {
		return "", fmt.Errorf("could not write kustomization overlay '%s': %v", overlayPath, err)
	}

	kustomizePath, err := cfg.ToolPath(config.Kustomize)
	if err != nil {
		return "", err
	}
	k8sResourcesPath = filepath.Join(overlayFolderPath, "expanded_resources.yml")
	cmd := proc.GracefulCommandContext(ctx, kustomizePath, "build", overlayFolderPath, "-o", k8sResourcesPath)
	cfg.Logger().Pipe(config.Kustomize.LogDomain(), cmd)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not run kustomize on '%s': %v", overlayFolderPath, err)
	}

	cfg.Logger().Info(logDomain, "kustomization expanded to '%s'", k8sResourcesPath)

	return k8sResourcesPath, nil
}
