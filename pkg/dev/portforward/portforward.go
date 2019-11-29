// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package portforward

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
)

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	ports *k8s.Ports,
) error {
	var g errgroup.Group
	for _, spec := range pipeline.Dev.PortForward {
		g.Go(func() error {
			return ports.PodPortForward(
				k8s.ServiceSpec{
					Labels: k8s.Labels{
						k8s.StackLabel: name.DNSName(),
					}.String() + "," + spec.Selector,
				},
				spec.LocalPort,
				spec.RemotePort,
			)
		})
	}
	return g.Wait()
}
