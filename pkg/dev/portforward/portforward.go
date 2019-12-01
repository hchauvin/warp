// portforward implements port forwarding for development purposes.
//
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
	k8sClient *k8s.K8s,
) error {
	var g errgroup.Group
	for _, spec := range pipeline.Dev.PortForward {
		g.Go(func() error {
			return k8sClient.Ports.PodPortForward(
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
