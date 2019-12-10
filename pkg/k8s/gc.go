package k8s

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type GcOptions struct {
	PreservePersistentVolumeClaims bool
}

func (k8s *K8s) Gc(ctx context.Context, cfg *config.Config, name names.Name, options *GcOptions) error {
	namespace := "default"
	labelSelector := Labels{
		StackLabel: name.DNSName(),
	}.String()

	var g errgroup.Group
	g.Go(func() error {
		api := k8s.Clientset.CoreV1().Services(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.AppsV1().Deployments(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.AppsV1().DaemonSets(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.AppsV1().StatefulSets(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.CoreV1().ConfigMaps(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.CoreV1().Secrets(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	g.Go(func() error {
		api := k8s.Clientset.NetworkingV1().NetworkPolicies(namespace)
		list, err := api.List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		var gd errgroup.Group
		for _, resource := range list.Items {
			resource := resource
			gd.Go(func() error {
				return api.Delete(resource.Name, nil)
			})
		}
		return gd.Wait()
	})
	for _, res := range k8s.cfg.Kubernetes.Resources {
		g.Go(func() error {
			api := k8s.DynClient.Resource(schema.GroupVersionResource{
				Group:    res.Group,
				Version:  res.Version,
				Resource: res.Resource,
			})
			list, err := api.List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return err
			}
			var gd errgroup.Group
			for _, resource := range list.Items {
				resource := resource
				gd.Go(func() error {
					return api.Delete(resource.GetName(), nil)
				})
			}
			return gd.Wait()
		})
	}
	if !options.PreservePersistentVolumeClaims {
		g.Go(func() error {
			api := k8s.Clientset.CoreV1().PersistentVolumeClaims(namespace)
			list, err := api.List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return err
			}
			var gd errgroup.Group
			for _, resource := range list.Items {
				resource := resource
				gd.Go(func() error {
					return api.Delete(resource.Name, nil)
				})
			}
			return gd.Wait()
		})
		g.Go(func() error {
			api := k8s.Clientset.CoreV1().PersistentVolumes()
			list, err := api.List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return err
			}
			var gd errgroup.Group
			for _, resource := range list.Items {
				resource := resource
				gd.Go(func() error {
					return api.Delete(resource.Name, nil)
				})
			}
			return gd.Wait()
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
