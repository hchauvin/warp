// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import (
	"context"
	"errors"
	"fmt"
	"github.com/phayes/freeport"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

const logDomain = "k8s"

// Ports is a "singleton" to pass around port forwarding information.  This
// singleton keeps track of all the port forwarding to 1) avoid forwarding
// ports in duplicate, 2) provide a single function to cancel all the port
// forwarding.
type Ports struct {
	k8sClient        *K8s
	forwardg         *errgroup.Group
	gctx             context.Context
	cancelForwarding context.CancelFunc
	mut              sync.Mutex
	cache            map[string]interface{}
}

// NewPorts creates the Ports singleton.
func newPorts(k8sClient *K8s) *Ports {
	ctx, cancelForwarding := context.WithCancel(context.Background())
	forwardg, gctx := errgroup.WithContext(ctx)
	forwarded := make(map[string]interface{})
	return &Ports{
		k8sClient,
		forwardg,
		gctx,
		cancelForwarding,
		sync.Mutex{},
		forwarded,
	}
}

// CancelForwarding cancels all the port forwarding.
func (ports *Ports) CancelForwarding() {
	ports.cancelForwarding()
}

// ServiceSpec specifies a Kubernetes service to forward from.
type ServiceSpec struct {
	// Namespace is the Kubernetes namespace to which the service belongs.
	Namespace string

	// Labels is a Kubernetes selector.
	Labels string
}

func (s ServiceSpec) String() string {
	return s.Namespace + "|" + s.Labels
}

// Port gives a random local port to which a remote port is forwarded.
// The remote port comes from the first pod that matches a service spec.
//
// PodPortForward can be used instead to fix the local port.
func (ports *Ports) Port(service ServiceSpec, exposedTcpPort int) (int, error) {
	ans, err := ports.memoize(
		func() (interface{}, error) {
			return ports.doPort(service, exposedTcpPort)
		},
		"Port",
		service,
		exposedTcpPort)
	if err != nil {
		return 0, err
	}
	return ans.(int), nil
}

// PodPortForward forward a remote port to a fixed local port.
// The remote port comes from the first pod that matches a service spec.
func (ports *Ports) PodPortForward(service ServiceSpec, localPort, exposedTCPPort int) error {
	_, err := ports.memoize(
		func() (interface{}, error) {
			err := ports.doPodPortForward(service, localPort, exposedTCPPort)
			return nil, err
		},
		"PodPortForward",
		service,
		localPort,
		exposedTCPPort)
	return err
}

func (ports *Ports) doPort(service ServiceSpec, exposedTcpPort int) (int, error) {
	localPort, err := freeport.GetFreePort()
	if err != nil {
		return 0, err
	}

	if err := ports.PodPortForward(service, localPort, exposedTcpPort); err != nil {
		return 0, err
	}

	return localPort, nil
}

func (ports *Ports) doPodPortForward(service ServiceSpec, localPort, exposedTCPPort int) error {
	ports.forwardg.Go(func() error {
		// This is very ugly, but will do until we use k8s endpoints to properly determine
		// which pods are alive.  Without this sleep, you listen to the old port in case
		// of a rolling update.
		time.Sleep(3 * time.Second)
		for {
			// Let's get the endpoints for the service
			endpoints, err := ports.getEndpoints(service)
			if err != nil {
				ports.k8sClient.cfg.Logger().Info(logDomain, "port-forward: cannot get endpoints for service %s: %v", service, err)
			} else {
				var targetRef *corev1.ObjectReference
				for _, endpoint := range endpoints {
					portFound := false
					for _, port := range endpoint.Ports {
						if port.Port == int32(exposedTCPPort) || port.Protocol == corev1.ProtocolTCP {
							portFound = true
							break
						}
					}
					if !portFound {
						continue
					}

					if len(endpoint.Addresses) == 0 {
						continue
					}

					targetRef = endpoint.Addresses[0].TargetRef
					if targetRef.Kind != "Pod" {
						return fmt.Errorf("expected 'Pod' kind for target ref, got '%s'", targetRef.Kind)
					}
				}

				if targetRef == nil {
					ports.k8sClient.cfg.Logger().Info(logDomain, "port-forward: no ready endpoint for service %s and TCP port %d", service, exposedTCPPort)
				} else {
					// TODO: Programmatic port forward
					cmd, err := ports.k8sClient.KubectlCommandContext(ports.gctx,
						"port-forward",
						"--namespace", targetRef.Namespace,
						"pod/"+targetRef.Name,
						fmt.Sprintf("%d:%d", localPort, exposedTCPPort),
					)
					if err != nil {
						return err
					}
					ports.k8sClient.cfg.Logger().Pipe(logDomain+":port-forward:ns="+service.Namespace+";l="+service.Labels, cmd)
					if err := cmd.Run(); err != nil {
						if err == ports.gctx.Err() {
							return err
						}
						ports.k8sClient.cfg.Logger().Info(logDomain, "port-forward: %v", err)
					}
				}
			}
			time.Sleep(2 * time.Second)
		}
	})

	return nil
}

func (ports *Ports) memoize(f func() (interface{}, error), fname string, args ...interface{}) (interface{}, error) {
	ports.mut.Lock()
	hash := fmt.Sprintf("%s %v", fname, args)
	if ans, ok := ports.cache[hash]; ok {
		ports.mut.Unlock()
		return ans, nil
	}
	ports.mut.Unlock()
	ans, err := f()
	if err != nil {
		return nil, err
	}
	ports.mut.Lock()
	ports.cache[hash] = ans
	ports.mut.Unlock()
	return ans, nil
}

func (ports *Ports) getEndpoints(service ServiceSpec) ([]corev1.EndpointSubset, error) {
	lst, err := ports.k8sClient.Clientset.CoreV1().
		Endpoints(service.Namespace).
		List(metav1.ListOptions{LabelSelector: service.Labels})
	if err != nil {
		return nil, err
	}

	if len(lst.Items) == 0 {
		return nil, errors.New("found no endpoint match")
	} else if len(lst.Items) > 1 {
		return nil, errors.New("found multiple endpoint matches")
	}

	return lst.Items[0].Subsets, nil
}
