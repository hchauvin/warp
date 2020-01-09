// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"strings"
)

func serviceSelector(name names.Name, service string) (string, error) {
	var selector string
	if strings.Contains(service, "=") {
		if strings.HasPrefix(service, "::") {
			selector = service[2:]
		} else {
			selector = k8s.Labels{
				k8s.StackLabel: name.DNSName(),
			}.String() + "," + service
		}
	} else {
		selector = k8s.Labels{
			k8s.StackLabel:   name.DNSName(),
			k8s.ServiceLabel: service,
		}.String()
	}
	return selector, nil
}
