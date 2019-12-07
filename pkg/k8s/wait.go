package k8s

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// WaitForEndpoints waits for all the services to have at least one ready endpoint.
func (k8s *K8s) WaitForEndpoints(ctx context.Context, k8sNamespace string, name names.Name) error {
	const subLogDomain = logDomain + ":waitFor:endpoints"

	// Wait for all the services to have at least one endpoint ready
	services, err := k8s.Clientset.CoreV1().Services(k8sNamespace).
		List(metav1.ListOptions{
			LabelSelector: Labels{
				StackLabel: name.DNSName(),
			}.String(),
		})
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, service := range services.Items {
		service := service
		serviceName, ok := service.Labels[ServiceLabel]
		if !ok {
			k8s.cfg.Logger().Warning(
				subLogDomain,
				"service %s|%s does not have the %s label, we cannot wait for its endpoints to be ready",
				service.Namespace,
				service.Name,
			)
			continue
		}
		g.Go(func() error {
			for {
				labelSelector := Labels{
					StackLabel:   name.DNSName(),
					ServiceLabel: serviceName,
				}.String()
				endpoints, err := k8s.Clientset.CoreV1().
					Endpoints(k8sNamespace).
					List(metav1.ListOptions{
						LabelSelector: labelSelector,
					})
				if err != nil {
					return fmt.Errorf(
						"could not get endpoints with label selector '%s': %v",
						labelSelector, err)
				}
				k8s.cfg.Logger().Info(
					subLogDomain,
					"service %s|%s has %d endpoints ready",
					service.Namespace,
					service.Name,
					len(endpoints.Items),
				)
				if len(endpoints.Items) > 0 {
					return nil
				}
				select {
				case <-time.After(3 * time.Second):
				case <-gctx.Done():
					return gctx.Err()
				}
			}
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// WaitForPods waits for all the pods to be ready
func (k8s *K8s) WaitForPods(ctx context.Context, k8sNamespace string, name names.Name) error {
	const subLogDomain = logDomain + ":waitFor:pods"

	for {
		pods, err := k8s.Clientset.CoreV1().Pods(k8sNamespace).
			List(metav1.ListOptions{
				LabelSelector: Labels{
					StackLabel: name.DNSName(),
				}.String(),
			})
		if err != nil {
			return err
		}
		ready := true
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodFailed {
				k8s.cfg.Logger().Info(
					subLogDomain,
					"pod %s: phase=%s", pod.Name, pod.Status.Phase)
				ready = false
			}
		}
		if ready {
			return nil
		}
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
