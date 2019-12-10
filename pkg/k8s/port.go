// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import (
	"errors"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"os"
	"sync"
	"time"
)

const logDomain = "k8s"

// Ports is a "singleton" to pass around port forwarding information.  This
// singleton keeps track of all the port forwarding to 1) avoid forwarding
// ports in duplicate, 2) provide a single function to cancel all the port
// forwarding.
type Ports struct {
	k8sClient *K8s
	mut       sync.Mutex
	cache     map[string]interface{}
	stopc     []chan struct{}
}

// NewPorts creates the Ports singleton.
func newPorts(k8sClient *K8s) *Ports {
	cache := make(map[string]interface{})
	return &Ports{
		k8sClient,
		sync.Mutex{},
		cache,
		nil,
	}
}

// CancelForwarding cancels all the port forwarding.
func (ports *Ports) CancelForwarding() {
	for _, c := range ports.stopc {
		close(c)
	}
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
func (ports *Ports) ServicePortForward(service ServiceSpec, localPortSpec, exposedTCPPort int) (localPort int, err error) {
	ans, err := ports.memoize(
		func() (interface{}, error) {
			return ports.doServicePortForward(service, localPortSpec, exposedTCPPort)
		},
		"PodPortForward",
		service,
		localPortSpec,
		exposedTCPPort)
	if err != nil {
		return 0, err
	}
	return ans.(int), nil
}

func (ports *Ports) doPort(service ServiceSpec, exposedTcpPort int) (int, error) {
	localPort, err := ports.ServicePortForward(service, 0, exposedTcpPort)
	if err != nil {
		return 0, err
	}

	return localPort, nil
}

func (ports *Ports) doServicePortForward(service ServiceSpec, localPortSpec, exposedTCPPort int) (int, error) {
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
					return 0, fmt.Errorf("expected 'Pod' kind for target ref, got '%s'", targetRef.Kind)
				}
			}

			if targetRef == nil {
				ports.k8sClient.cfg.Logger().Info(logDomain, "port-forward: no ready endpoint for service %s and TCP port %d", service, exposedTCPPort)
			} else {
				return ports.PodPortForward(targetRef.Namespace, targetRef.Name, localPortSpec, exposedTCPPort)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (ports *Ports) PodPortForward(namespace, name string, localPortSpec, exposedTCPPort int) (int, error) {
	req := ports.k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(ports.k8sClient.restconfig)
	if err != nil {
		return 0, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	stopc := make(chan struct{}, 1)
	ports.mut.Lock()
	ports.stopc = append(ports.stopc, stopc)
	ports.mut.Unlock()
	readyc := make(chan struct{})

	fw, err := portforward.NewOnAddresses(
		dialer,
		[]string{"localhost"},
		[]string{fmt.Sprintf("%d:%d", localPortSpec, exposedTCPPort)},
		stopc, readyc, ioutil.Discard, os.Stderr)
	if err != nil {
		return 0, err
	}

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			ports.k8sClient.cfg.Logger().Error(logDomain, "port-forward: %v\n", err)
		}
	}()

	<-readyc

	forwardedPorts, err := fw.GetPorts()
	if err != nil {
		return 0, err
	}
	return int(forwardedPorts[0].Local), nil
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
