// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"io"
	"os/exec"
	"strings"
)

func Tail(ctx context.Context, cfg *config.Config, name names.Name) error {
	kubectlPath, err := cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return err
	}

	out, err := exec.CommandContext(
		ctx,
		kubectlPath,
		"get",
		"service",
		"--all-namespaces",
		"-l",
		Labels{
			StackLabel: name.DNSName(),
		}.String(),
		"-o=json",
	).Output()
	if err != nil {
		return err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(out, &info); err != nil {
		return fmt.Errorf("cannot unmarshal output of 'kubectl get': %v; full output: <<< %s >>>", err, out)
	}

	specs, err := parseTailSpec(cfg, info)
	if err != nil {
		return fmt.Errorf("cannot process output of 'kubectl get': %v; full output: <<< %s >>>", err, out)
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, spec := range specs {
		spec := spec
		g.Go(func() error {
			for {
				cmd := exec.CommandContext(
					gctx,
					kubectlPath,
					"logs",
					"-f",
					"--namespace",
					spec.namespace,
					"--tail=1",
					"--all-containers=true",
					"service/"+spec.name,
				)
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					return err
				}
				stderr, err := cmd.StderrPipe()
				if err != nil {
					return err
				}
				r := io.MultiReader(stdout, stderr)
				subLogDomain := "tail." + spec.namespace + "." + spec.name
				initializing := false
				go func() {
					scanner := bufio.NewScanner(r)
					for scanner.Scan() {
						s := string(scanner.Bytes())
						cfg.Logger().Info(subLogDomain, s)
						if strings.Contains(s, "PodInitializing") || strings.Contains(s, "ContainerCreating") {
							initializing = true
						}
					}
				}()
				if err := cmd.Run(); err != nil {
					if initializing {
						continue
					}
					return fmt.Errorf("cannot tail %s|%s: %v", spec.namespace, spec.name, err)
				}
				return nil
			}
		})
	}
	return g.Wait()
}

type tailSpec struct {
	namespace string
	name      string
}

func parseTailSpec(cfg *config.Config, info map[string]interface{}) (specs []tailSpec, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	var specStr strings.Builder
	for _, service := range info["items"].([]interface{}) {
		metadata := service.(map[string]interface{})["metadata"].(map[string]interface{})
		cur := tailSpec{
			namespace: metadata["namespace"].(string),
			name:      metadata["name"].(string),
		}
		specs = append(specs, cur)
		specStr.WriteString("\n  ")
		specStr.WriteString(cur.namespace)
		specStr.WriteRune('|')
		specStr.WriteString(cur.name)
	}

	cfg.Logger().Info(
		"k8s",
		"Tailing the following services: %s",
		specStr.String(),
	)

	return specs, nil
}
