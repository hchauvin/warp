// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/phayes/freeport"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

const logDomain = "k8s"

// Ports is a "singleton" to pass around port forwarding information.  This
// singleton keeps track of all the port forwarding to 1) avoid forwarding
// ports in duplicat, 2) provide a single function to cancel all the port
// forwarding.
type Ports struct {
	cfg              *config.Config
	forwardg         *errgroup.Group
	gctx             context.Context
	cancelForwarding context.CancelFunc
	mut              sync.Mutex
	cache            map[string]interface{}
}

// NewPorts creates the Ports singleton.
func NewPorts(cfg *config.Config) *Ports {
	ctx, cancelForwarding := context.WithCancel(context.Background())
	forwardg, gctx := errgroup.WithContext(ctx)
	forwarded := make(map[string]interface{})
	return &Ports{
		cfg,
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
	kubectlPath, err := ports.cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return err
	}

	ports.forwardg.Go(func() error {
		// This is very ugly, but will do until we use k8s endpoints to properly determine
		// which pods are alive.  Without this sleep, you listen to the old port in case
		// of a rolling update.
		time.Sleep(3 * time.Second)
		for {
			// Let's get the full name of the service from the selector
			out, err := proc.GracefulCommandContext(
				ports.gctx, kubectlPath, "get",
				"--namespace", service.Namespace, "-l", service.Labels, "-o=json", "service").Output()
			if err != nil {
				return err
			}

			var serviceInfo map[string]interface{}
			if err := json.Unmarshal(out, &serviceInfo); err != nil {
				return fmt.Errorf("cannot unmarshal output of 'kubectl get': %v; full output: <<< %s >>>", err, out)
			}
			serviceName, err := parseServiceInfo(serviceInfo)
			if err != nil {
				ports.cfg.Logger().Info(logDomain, "port-fowarding: cannot process output of 'kubectl get': %v; full output: <<< %s >>>", err, out)
				continue
			}

			cmd := proc.GracefulCommandContext(ports.gctx, kubectlPath,
				"port-forward",
				"--namespace", service.Namespace,
				"service/"+serviceName,
				fmt.Sprintf("%d:%d", localPort, exposedTCPPort),
			)
			ports.cfg.Logger().Pipe(logDomain+":port-forward:ns="+service.Namespace+";l="+service.Labels, cmd)
			if err := cmd.Run(); err != nil {
				if err == ports.gctx.Err() {
					return err
				}
				ports.cfg.Logger().Info(logDomain, "port-forward: %v", err)
			}
			return nil
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

func parseServiceInfo(serviceInfo map[string]interface{}) (serviceName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	items := serviceInfo["items"].([]interface{})
	if len(items) == 0 {
		return "", errors.New("found no service match")
	} else if len(items) > 1 {
		return "", errors.New("found multiple service matches")
	}

	serviceName = items[0].(map[string]interface{})["metadata"].(map[string]interface{})["name"].(string)
	return serviceName, nil
}
