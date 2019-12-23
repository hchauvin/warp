// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package helm

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/templates"
	"text/template"
)

type templateFuncs struct {
	cfg  *config.Config
	name names.Name
}

// Gets get the expansion of a template.
func (funcs *templateFuncs) Get(
	ctx context.Context,
	tplStr string,
) (string, error) {
	tpl, err := template.New("config").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templates.TxtFuncMap()).
		Funcs(funcs.txtFuncMap(ctx)).
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

func (funcs *templateFuncs) txtFuncMap(ctx context.Context) template.FuncMap {
	return map[string]interface{}{
		"stackName": func() string {
			return funcs.name.DNSName()
		},
	}
}
