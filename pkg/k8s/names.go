// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package k8s

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceName returns the name of a service matching a spec.
func (k8s *K8s) ServiceName(ctx context.Context, service ServiceSpec) (string, error) {
	list, err := k8s.Clientset.CoreV1().Services(service.Namespace).List(metav1.ListOptions{
		LabelSelector: service.Labels,
	})
	if err != nil {
		return "", err
	}
	if len(list.Items) != 1 {
		return "", fmt.Errorf("expected one and only one service to match spec %v", service)
	}
	return list.Items[0].Name, nil
}
