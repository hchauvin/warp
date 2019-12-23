// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/hchauvin/warp/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
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
		if strings.HasPrefix(service, "::") {
			selector = service[2:]
		} else {
			selector = k8s.Labels{
				k8s.StackLabel: funcs.name.DNSName(),
			}.String() + "," + service
		}
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

func (funcs *templateFuncs) k8sServiceName(
	ctx context.Context,
	namespace string,
	service string,
) (string, error) {
	var selector string
	if strings.Contains(service, "=") {
		if strings.HasPrefix(service, "::") {
			selector = service[2:]
		} else {
			selector = k8s.Labels{
				k8s.StackLabel: funcs.name.DNSName(),
			}.String() + "," + service
		}
	} else {
		selector = k8s.Labels{
			k8s.StackLabel:   funcs.name.DNSName(),
			k8s.ServiceLabel: service,
		}.String()
	}
	name, err := funcs.k8sClient.ServiceName(ctx,
		k8s.ServiceSpec{
			Namespace: namespace,
			Labels:    selector,
		})
	if err != nil {
		return "", err
	}
	return name, nil
}

func (funcs *templateFuncs) k8sConfigMapKey(
	ctx context.Context,
	namespace string,
	name string,
	key string,
) (string, error) {
	var configMap corev1.ConfigMap
	err := retry.Do(func() error {
		list, err := funcs.k8sClient.Clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{
			LabelSelector: k8s.Labels{
				k8s.StackLabel: funcs.name.DNSName(),
			}.String(),
		})
		if err != nil {
			return err
		}
		re := regexp.MustCompile("^" + funcs.name.DNSName() + "-" + name + "-[a-z0-9]+$")
		var secretNames []string
		for _, cfgmap := range list.Items {
			if re.MatchString(cfgmap.Name) {
				secretNames = append(secretNames, cfgmap.Name)
				configMap = cfgmap
			}
		}
		if len(secretNames) == 0 {
			return fmt.Errorf("no matching config map found")
		}
		if len(secretNames) > 1 {
			return retry.Unrecoverable(fmt.Errorf("multiple matching config maps found: %s", strings.Join(secretNames, " ")))
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	val, ok := configMap.Data[key]
	if !ok {
		keys := make([]string, 0, len(configMap.Data))
		for key := range configMap.Data {
			keys = append(keys, key)
		}
		return "", fmt.Errorf("key '%s' was not found in config map %s; keys: [%s]", key, configMap.Name, strings.Join(keys, " "))
	}

	return val, nil
}

func (funcs *templateFuncs) k8sSecretKey(
	ctx context.Context,
	namespace string,
	name string,
	key string,
) (string, error) {
	var secret corev1.Secret
	err := retry.Do(func() error {
		list, err := funcs.k8sClient.Clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{
			LabelSelector: k8s.Labels{
				k8s.StackLabel: funcs.name.DNSName(),
			}.String(),
		})
		if err != nil {
			return err
		}
		re := regexp.MustCompile("^" + funcs.name.DNSName() + "-" + name + "(-[a-z0-9]+)?$")
		var secretNames []string
		for _, sec := range list.Items {
			if re.MatchString(sec.Name) {
				secretNames = append(secretNames, sec.Name)
				secret = sec
			}
		}
		if len(secretNames) == 0 {
			return fmt.Errorf("no matching secret found")
		}
		if len(secretNames) > 1 {
			return retry.Unrecoverable(fmt.Errorf("multiple matching secrets found: %s", strings.Join(secretNames, " ")))
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	val, ok := secret.Data[key]
	if !ok {
		keys := make([]string, 0, len(secret.Data))
		for key := range secret.Data {
			keys = append(keys, key)
		}
		return "", fmt.Errorf("key '%s' was not found in secret %s; keys: [%s]", key, secret.Name, strings.Join(keys, " "))
	}

	return string(val), nil
}
