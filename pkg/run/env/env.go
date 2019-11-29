// env implements environment variable templating.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package env

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"text/template"
)

func Transform(
	ctx context.Context,
	cfg *config.Config,
	name names.Name,
	env []string,
	ports *k8s.Ports,
) ([]string, error) {
	funcs := templateFuncs{cfg, name, ports, make(map[string]struct {
		string
		error
	})}

	ans := make([]string, len(env))
	for i, tplStr := range env {
		tpl, err := template.New("config").
			Funcs(sprig.TxtFuncMap()).
			Funcs(funcs.txtFuncMap(ctx)).
			Parse(tplStr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse template '%s': %v", tplStr, err)
		}
		data := map[string]interface{}{}
		w := &bytes.Buffer{}
		if err := tpl.Execute(w, data); err != nil {
			return nil, fmt.Errorf("cannot expand template: %v", err)
		}
		ans[i] = w.String()
	}
	return ans, nil
}

type templateFuncs struct {
	cfg   *config.Config
	name  names.Name
	ports *k8s.Ports
	cache map[string]struct {
		string
		error
	}
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
	if ans, ok := funcs.cache[hash]; ok {
		return ans.string, ans.error
	}
	ans, err := f()
	funcs.cache[hash] = struct {
		string
		error
	}{ans, err}
	return ans, err
}
