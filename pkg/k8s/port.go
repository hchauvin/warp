// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/phayes/freeport"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"sync"
)

const logDomain = "k8s"

type Ports struct {
	cfg              *config.Config
	forwardg         *errgroup.Group
	gctx             context.Context
	cancelForwarding context.CancelFunc
	mut              sync.Mutex
	cache            map[string]interface{}
}

type ServiceSpec struct {
	Namespace string
	Labels    string
}

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

func (ports *Ports) CancelForwarding() {
	ports.cancelForwarding()
}

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
		for {
			out, err := exec.CommandContext(
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

			cmd := exec.CommandContext(ports.gctx, kubectlPath,
				"port-forward",
				"--namespace", service.Namespace,
				"service/"+serviceName,
				fmt.Sprintf("%d:%d", localPort, exposedTCPPort),
			)
			ports.cfg.Logger().Pipe(logDomain+":port-forward:ns="+service.Namespace+";l="+service.Labels, cmd)
			if err := cmd.Run(); err != nil {
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

	/*for _, pod := range podInfo["items"].([]interface{}) {
		containerStatuses := pod.
		(map[string]interface{})["status"].
		(map[string]interface{})["containerStatuses"].
		([]interface{})
		ready := true
		for _, status := range containerStatuses {
			if !status.(map[string]interface{})["ready"].(bool) {
				ready = false
				break
			}
		}
		if ready {
			name := pod.
			(map[string]interface{})["metadata"].
			(map[string]interface{})["name"].
			(string)
			readyPods = append(readyPods, name)
		}
	}

	if len(readyPods) == 0 {
		return nil, errors.New("expected at least one ready pod")
	}
	return readyPods, nil*/
}
