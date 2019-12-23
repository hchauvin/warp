// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package k8s

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// gcResources lists the resources that are garbage-collected by default.
var gcResources = []config.Resource{
	{
		Version:    "v1",
		Resource:   "services",
		Namespaced: true,
	},
	{
		Version:    "v1",
		Resource:   "configmaps",
		Namespaced: true,
	},
	{
		Version:    "v1",
		Resource:   "secrets",
		Namespaced: true,
	},
	{
		Group:      "apps",
		Version:    "v1",
		Resource:   "deployments",
		Namespaced: true,
	},
	{
		Group:      "apps",
		Version:    "v1",
		Resource:   "statefulsets",
		Namespaced: true,
	},
	{
		Group:      "apps",
		Version:    "v1",
		Resource:   "daemonsets",
		Namespaced: true,
	},
	{
		Group:      "autoscaling",
		Version:    "v2beta2",
		Resource:   "horizontalpodautoscalers",
		Namespaced: true,
	},
	{
		Group:      "policy",
		Version:    "v1beta1",
		Resource:   "poddisruptionbudgets",
		Namespaced: true,
	},
	{
		Group:      "networking.k8s.io",
		Version:    "v1",
		Resource:   "networkpolicies",
		Namespaced: true,
	},
}

// gcResourcesVolumes lists the resources that are garbage-collected when
// the volumes are not preserved.
var gcResourcesVolumes = []config.Resource{
	{
		Version:    "v1",
		Resource:   "persistentvolumeclaims",
		Namespaced: true,
	},
}

// GcOptions are options for the Gc function.
type GcOptions struct {
	// PreservePersistentVolumeClaims indicates whether to preserve
	// the persistent volume claims instead of garbage-collecting them.
	PreservePersistentVolumeClaims bool
}

// Gc garbage-collects resources that pertain to a given stack.
func (k8s *K8s) Gc(ctx context.Context, cfg *config.Config, name names.Name, options *GcOptions) error {
	namespace := "default"
	labelSelector := Labels{
		StackLabel: name.DNSName(),
	}.String()

	var g errgroup.Group
	var resources []config.Resource
	resources = append(resources, gcResources...)
	if !options.PreservePersistentVolumeClaims {
		resources = append(resources, gcResourcesVolumes...)
	}
	if k8s.cfg.Kubernetes != nil {
		resources = append(resources, k8s.cfg.Kubernetes.Resources...)
	}
	for _, res := range resources {
		res := res
		g.Go(func() error {
			var api dynamic.ResourceInterface
			if res.Namespaced {
				api = k8s.DynClient.
					Resource(schema.GroupVersionResource{
						Group:    res.Group,
						Version:  res.Version,
						Resource: res.Resource,
					}).
					Namespace(namespace)
			} else {
				api = k8s.DynClient.
					Resource(schema.GroupVersionResource{
						Group:    res.Group,
						Version:  res.Version,
						Resource: res.Resource,
					})
			}
			list, err := api.List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return fmt.Errorf("cannot list resources %v: %v", res, err)
			}
			var gd errgroup.Group
			for _, resource := range list.Items {
				resource := resource
				gd.Go(func() error {
					if err := api.Delete(resource.GetName(), nil); err != nil {
						return fmt.Errorf(
							"cannot delete resource %v %v: %v",
							res, resource.GetName(), err)
					}
					return nil
				})
			}
			return gd.Wait()
		})
	}
	return g.Wait()
}
