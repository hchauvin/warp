// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package kubescore

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
)

const logDomain = "kubescore"

func Lint(ctx context.Context, cfg *config.Config, k8sResourcesPath string) error {
	kubeScorePath, err := cfg.ToolPath(config.KubeScore)
	if err != nil {
		return err
	}

	cmd := proc.GracefulCommandContext(ctx, kubeScorePath, "score", k8sResourcesPath)
	cfg.Logger().Pipe(logDomain, cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
