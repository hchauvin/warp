// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/stacks/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"strings"
	"sync"
	"text/template"
)

func K8sTemplateFuncs(cfg *config.Config, name names.Name, k8sClient *k8s.K8s) *k8sTemplateFuncs {
	return &k8sTemplateFuncs{
		cfg,
		name,
		k8sClient,
		sync.RWMutex{},
		make(map[string]cacheEntry),
	}
}

type k8sTemplateFuncs struct {
	cfg       *config.Config
	name      names.Name
	k8sClient *k8s.K8s
	cacheMut  sync.RWMutex
	cache     map[string]cacheEntry
}

func (funcs *k8sTemplateFuncs) TxtFuncMap(ctx context.Context) template.FuncMap {
	return map[string]interface{}{
		"serviceAddress": func(service string, exposedTCPPort int) (string, error) {
			return funcs.memoize(
				func() (string, error) {
					return funcs.serviceAddress(ctx, service, exposedTCPPort)
				},
				"serviceAddress",
				service,
				exposedTCPPort,
			)
		},
		"k8sServiceAddress": func(namespace, service string, exposedTCPPort int) (string, error) {
			return funcs.memoize(
				func() (string, error) {
					return funcs.k8sServiceAddress(ctx, namespace, service, exposedTCPPort)
				},
				"k8sServiceAddress",
				namespace,
				service,
				exposedTCPPort,
			)
		},
		"k8sServiceName": func(namespace, service string) (string, error) {
			return funcs.memoize(
				func() (string, error) {
					return funcs.k8sServiceName(ctx, namespace, service)
				},
				"k8sServiceName",
				namespace,
				service,
			)
		},
		"k8sConfigMapKey": func(namespace, name, key string) (string, error) {
			return funcs.memoize(
				func() (string, error) {
					return funcs.k8sConfigMapKey(ctx, namespace, name, key)
				},
				"k8sConfigMapKey",
				namespace,
				name,
				key,
			)
		},
		"k8sSecretKey": func(namespace, name, key string) (string, error) {
			return funcs.memoize(
				func() (string, error) {
					return funcs.k8sSecretKey(ctx, namespace, name, key)
				},
				"k8sSecretKey",
				namespace,
				name,
				key,
			)
		},
	}
}

func (funcs *k8sTemplateFuncs) serviceAddress(
	ctx context.Context,
	service string,
	exposedTCPPort int,
) (string, error) {
	return funcs.k8sServiceAddress(ctx, "default", service, exposedTCPPort)
}

func (funcs *k8sTemplateFuncs) k8sServiceAddress(
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

func (funcs *k8sTemplateFuncs) k8sServiceName(
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

func (funcs *k8sTemplateFuncs) k8sConfigMapKey(
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
		re := regexp.MustCompile("^" + funcs.name.DNSName() + "-" + name + "(-[a-z0-9]+)?$")
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

func (funcs *k8sTemplateFuncs) k8sSecretKey(
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

func (funcs *k8sTemplateFuncs) memoize(f func() (string, error), fname string, args ...interface{}) (string, error) {
	hash := fmt.Sprintf("%s %v", fname, args)
	funcs.cacheMut.RLock()
	cached, ok := funcs.cache[hash]
	funcs.cacheMut.RUnlock()
	if ok {
		return cached.string, cached.error
	}
	ans, err := f()
	funcs.cacheMut.Lock()
	funcs.cache[hash] = cacheEntry{ans, err}
	funcs.cacheMut.Unlock()
	return ans, err
}
