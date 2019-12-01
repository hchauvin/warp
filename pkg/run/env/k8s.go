// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package env

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/hchauvin/warp/pkg/k8s"
	"strings"
)

func (funcs *templateFuncs) k8sServiceAddress(
	ctx context.Context,
	namespace string,
	service string,
	exposedTCPPort int,
) (string, error) {
	var selector string
	if strings.Contains(service, "=") {
		selector = k8s.Labels{
			k8s.StackLabel: funcs.name.DNSName(),
		}.String() + "," + service
	} else {
		selector = k8s.Labels{
			k8s.StackLabel:   funcs.name.DNSName(),
			k8s.ServiceLabel: service,
		}.String()
	}
	port, err := funcs.k8sClient.Ports.Port(
		k8s.ServiceSpec{
			Namespace: namespace,
			Labels:    selector,
		},
		exposedTCPPort)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("127.0.0.1:%d", port), nil
}

func (funcs *templateFuncs) k8sConfigMapKey(
	ctx context.Context,
	namespace string,
	name string,
	key string,
) (string, error) {
	return funcs.k8sDataKey(
		ctx,
		"configMap",
		namespace,
		name,
		key,
	)
}

func (funcs *templateFuncs) k8sSecretKey(
	ctx context.Context,
	namespace string,
	name string,
	key string,
) (string, error) {
	return funcs.k8sDataKey(
		ctx,
		"secret",
		namespace,
		name,
		key,
	)
}

func (funcs *templateFuncs) k8sDataKey(
	ctx context.Context,
	kind string,
	namespace string,
	name string,
	key string,
) (string, error) {
	var val string
	err := retry.Do(func() error {
		cmd, err := funcs.k8sClient.KubectlCommandContext(
			ctx,
			"get",
			"--namespace",
			namespace,
			"-l",
			k8s.Labels{
				k8s.StackLabel: funcs.name.DNSName(),
				k8s.NameLabel:  name,
			}.String(),
			"-o=json",
			kind)
		if err != nil {
			return err
		}
		out, err := cmd.Output()
		if err != nil {
			return err
		}

		var resource map[string]interface{}
		if err := json.Unmarshal(out, &resource); err != nil {
			return retry.Unrecoverable(fmt.Errorf("cannot unmarshal output of 'kubectl get': %v; full output: <<< %s >>>", err, out))
		}

		var recoverable bool
		val, recoverable, err = parseK8sData(resource, key)
		if err != nil {
			if recoverable {
				return err
			}
			return retry.Unrecoverable(fmt.Errorf("cannot parse output of 'kubectl get': %v; full output: <<< %s >>>", err, out))
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	return val, nil
}

func parseK8sData(resource map[string]interface{}, key string) (val string, recoverable bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	items := resource["items"].([]interface{})
	if len(items) == 0 {
		return "", true, fmt.Errorf("expected at least one resource matching the selector")
	}

	val = items[0].(map[string]interface{})["data"].(map[string]interface{})[key].(string)
	return val, true, nil
}
