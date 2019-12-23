// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package env implements environment variable templating.
package env

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/templates"
	"sync"
	"text/template"
)

// Transformer is used to to expand gotemplates in environment
// variables.
type Transformer struct {
	funcs templateFuncs
}

// NewTransformer creates a new Transformer.
func NewTransformer(
	cfg *config.Config,
	name names.Name,
	k8sClient *k8s.K8s,
) *Transformer {
	funcs := templateFuncs{
		cfg,
		name,
		k8sClient,
		sync.RWMutex{},
		make(map[string]cacheEntry),
	}
	return &Transformer{funcs}
}

// Get gets the expansion of a template.
func (trans *Transformer) Get(
	ctx context.Context,
	tplStr string,
) (string, error) {
	tpl, err := template.New("config").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templates.TxtFuncMap()).
		Funcs(trans.funcs.txtFuncMap(ctx)).
		Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("cannot parse template <<< %s >>>: %v", tplStr, err)
	}
	data := map[string]interface{}{}
	w := &bytes.Buffer{}
	if err := tpl.Execute(w, data); err != nil {
		return "", fmt.Errorf("cannot expand template <<< %s >>>: %v", tplStr, err)
	}
	return w.String(), nil
}

type cacheEntry struct {
	string
	error
}

type templateFuncs struct {
	cfg       *config.Config
	name      names.Name
	k8sClient *k8s.K8s
	cacheMut  sync.RWMutex
	cache     map[string]cacheEntry
}

func (funcs *templateFuncs) txtFuncMap(ctx context.Context) template.FuncMap {
	return template.FuncMap(map[string]interface{}{
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
	})
}

func (funcs *templateFuncs) serviceAddress(
	ctx context.Context,
	service string,
	exposedTCPPort int,
) (string, error) {
	return funcs.k8sServiceAddress(ctx, "default", service, exposedTCPPort)
}

func (funcs *templateFuncs) memoize(f func() (string, error), fname string, args ...interface{}) (string, error) {
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
