// run implements the "run" step of pipelines.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package run

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"os"
	"strings"
)

// Exec runs the commands of a pipeline.
func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	specNames []string,
	ports *k8s.Ports,
) error {
	for _, specName := range specNames {
		var spec *pipelines.Command
		for _, s := range pipeline.Commands {
			if s.Name == specName {
				spec = &s
				break
			}
		}
		if spec == nil {
			return fmt.Errorf("unrecognized run '%s'", specName)
		}

		if len(spec.Command) == 0 {
			return fmt.Errorf("run '%s': command must at least give the program name", specName)
		}
		cmd := proc.GracefulCommandContext(ctx, spec.Command[0], spec.Command[1:]...)
		if spec.WorkingDir != "" {
			cmd.Dir = cfg.Path(spec.WorkingDir)
		}
		extraEnv, err := env.Transform(ctx, cfg, name, spec.Env, ports)
		if err != nil {
			return fmt.Errorf("cannot transform env vars: %v", err)
		}
		cfg.Logger().Info("run:"+specName+":env", strings.Join(extraEnv, "\n"))
		cmd.Env = append(os.Environ(), extraEnv...)
		cfg.Logger().Pipe("run:"+specName, cmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("could not run '%s': %v", spec.Name, err)
		}
	}

	return nil
}
