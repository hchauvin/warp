package k8s

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// WaitForEndpoints waits for all the services to have at least one ready endpoint.
func (k8s *K8s) WaitForEndpoints(ctx context.Context, k8sNamespace string, name names.Name) error {
	const subLogDomain = logDomain + ":waitFor:endpoints"

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

				addressesCount := 0
				notReadyAddressesCount := 0
				for _, endpoint := range endpoints.Items {
					addressesCount += len(endpoint.Subsets[0].Addresses)
					notReadyAddressesCount += len(endpoint.Subsets[0].NotReadyAddresses)
				}

				if notReadyAddressesCount == 0 {
					k8s.cfg.Logger().Info(
						subLogDomain,
						"service %s|%s has %d endpoints and %d addresses ready",
						service.Namespace,
						service.Name,
						len(endpoints.Items),
						addressesCount,
					)
					return nil
				}

				k8s.cfg.Logger().Info(
					subLogDomain,
					"service %s|%s: %d endpoints, %d/%d addresses ready",
					service.Namespace,
					service.Name,
					len(endpoints.Items),
					addressesCount,
					addressesCount/(addressesCount+notReadyAddressesCount),
				)

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

// WaitForEndpoints waits for all the services to have at least one ready endpoint.
func (k8s *K8s) WaitForOnePodPerService(ctx context.Context, k8sNamespace string, name names.Name) error {
	const subLogDomain = logDomain + ":waitFor:onePodPerService"

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
	var readyCount atomic.Int64
	k8s.cfg.Logger().Info(
		subLogDomain,
		"0/%d services ready",
		len(services.Items))
	for _, service := range services.Items {
		service := service
		serviceName, ok := service.Labels[ServiceLabel]
		if !ok {
			k8s.cfg.Logger().Warning(
				subLogDomain,
				"service %s|%s does not have the %s label, we cannot wait for at least one of its pods to be ready",
				service.Namespace,
				service.Name,
			)
			continue
		}
		g.Go(func() error {
			labelSelector := Labels{
				StackLabel:   name.DNSName(),
				ServiceLabel: serviceName,
			}.String()

			if err := k8s.WaitForOnePodRunning(gctx, k8sNamespace, labelSelector); err != nil {
				return err
			}
			k8s.cfg.Logger().Info(
				subLogDomain,
				"%d/%d services ready",
				readyCount.Inc(),
				len(services.Items))

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

type podStatus string

const (
	scheduled   = podStatus("scheduled")
	autoscaling = podStatus("autoscaling")
	pulling     = podStatus("pulling")
	pending     = podStatus("pending")
	running     = podStatus("running")
	failed      = podStatus("failed")
	terminating = podStatus("terminating")
)

// WaitForPodsAllRunning waits for all the pods to be running.
func (k8s *K8s) WaitForAllPodsRunning(
	ctx context.Context,
	k8sNamespace string,
	labelSelector string,
) error {
	for {
		pods, err := k8s.Clientset.CoreV1().Pods(k8sNamespace).
			List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
		if err != nil {
			return err
		}

		notRunningCount := 0
		statusList := make(map[string]podStatus)
		for _, pod := range pods.Items {
			status, err := getPodStatus(&pod)
			if err != nil {
				return err
			}
			statusList[pod.Name] = status
			if status != running {
				notRunningCount++
				k8s.cfg.Logger().Info(logDomain+":wait", "%s %s\n", pod.Name, status)
			}
		}

		if notRunningCount == 0 {
			return nil
		}

		time.Sleep(3 * time.Second)
	}
}

// WaitForPodsAllRunning waits for at least one pod to be running.
func (k8s *K8s) WaitForOnePodRunning(
	ctx context.Context,
	k8sNamespace string,
	labelSelector string,
) error {
	for {
		pods, err := k8s.Clientset.CoreV1().Pods(k8sNamespace).
			List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
		if err != nil {
			return err
		}
		statusList := make(map[string]podStatus)
		for _, pod := range pods.Items {
			status, err := getPodStatus(&pod)
			if err != nil {
				return err
			}
			statusList[pod.Name] = status
			if status == running {
				return nil
			}
		}

		time.Sleep(3 * time.Second)
	}
}

func getPodStatus(pod *corev1.Pod) (podStatus, error) {
	if pod.DeletionTimestamp != nil {
		return terminating, nil
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		switch pod.Status.Reason {
		case "Scheduled":
			return scheduled, nil

		case "Pulling":
			return pulling, nil

		default:
			return pending, nil
		}

	case corev1.PodRunning:
		fallthrough
	case corev1.PodSucceeded:
		return running, nil

	case corev1.PodFailed:
		return failed, nil

	case corev1.PodUnknown:
		fallthrough
	default:
		panic(fmt.Sprintf("unrecognized pod status %v", pod.Status.Phase))
	}
}
