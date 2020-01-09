// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package env implements environment variable templating.
package env

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/hchauvin/warp/pkg/templates"
	"text/template"
)

// TemplateFuncs gives a func map.
type TemplateFuncs interface {
	TxtFuncMap(ctx context.Context) template.FuncMap
}

// Transformer is used to to expand gotemplates in environment
// variables.
type Transformer struct {
	funcs []TemplateFuncs
}

// NewTransformer creates a new Transformer.
func NewTransformer(funcs ...TemplateFuncs) *Transformer {
	return &Transformer{funcs}
}

// Get gets the expansion of a template.
func (trans *Transformer) Get(
	ctx context.Context,
	tplStr string,
) (string, error) {
	tpl := template.New("config").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templates.TxtFuncMap())
	for _, funcs := range trans.funcs {
		tpl = tpl.Funcs(funcs.TxtFuncMap(ctx))
	}
	tpl, err := tpl.Parse(tplStr)
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
