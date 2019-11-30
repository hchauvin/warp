// container implements deployment operations performed on the containers themselves.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package container

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"os"
	"strings"
	"sync"
)

type ImageRefs map[string]string

// Exec excutes the deployment operations addressing the containers themselves.
func Exec(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) (ImageRefs, error) {
	manifest := pipeline.Deploy.Container.ParsedManifest
	if manifest == nil {
		return nil, nil
	}

	dockerPath, err := cfg.Tools[config.Docker].Resolve()
	if err != nil {
		return nil, err
	}
	dk := docker{
		path: dockerPath,
	}

	refs := make(map[string]string, len(manifest))
	var mut sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	for k, v := range manifest {
		k, v := k, v
		g.Go(func() error {
			ref := v.Ref
			if ref == "" {
				return nil
			}
			if len(pipeline.Deploy.Container.Label) > 0 {
				expandedLabels := make(map[string]string, len(pipeline.Deploy.Container.Label))
				for _, lbl := range pipeline.Deploy.Container.Label {
					parts := strings.SplitN(lbl, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("unexpected label spec '%s'", lbl)
					}
					expandedLabels[parts[0]] = os.ExpandEnv(parts[1])
				}
				nextRef, err := dk.build(gctx, cfg, ref, expandedLabels)
				if err != nil {
					return err
				}
				if pipeline.Deploy.Container.Push == "" {
					parts := strings.SplitN(ref, ":", 2)
					ref = parts[0] + ":" + strings.Replace(nextRef, "sha256:", "", 1)
					if err := dk.tag(gctx, cfg, nextRef, ref); err != nil {
						return err
					}
					if err := dk.push(gctx, cfg, ref); err != nil {
						return err
					}
				}
			}
			if pipeline.Deploy.Container.Push != "" {
				parts := strings.SplitN(ref, "/", 2)
				registry := os.ExpandEnv(pipeline.Deploy.Container.Push)
				var nextRef string
				if len(parts) == 2 {
					nextRef = registry + "/" + parts[1]
				} else {
					nextRef = registry + "/" + ref
				}
				if err := dk.tag(gctx, cfg, ref, nextRef); err != nil {
					return err
				}
				if err := dk.push(gctx, cfg, nextRef); err != nil {
					return err
				}
				ref = nextRef
			}
			mut.Lock()
			refs[k] = ref
			mut.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return refs, nil
}

// CleanUp cleans up/removes the resources created during the "container deploy" step.
func CleanUp(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) error {
	return nil
}
