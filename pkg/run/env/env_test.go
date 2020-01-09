// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestTransformerGet(t *testing.T) {
	tr := NewTransformer(&testTemplateFuncs{
		txtFuncMap: template.FuncMap{
			"ok": func(arg string) string {
				return arg
			},
			"fail": func(arg string) string {
				panic("__fail__")
			},
		},
	})

	s, err := tr.Get(context.Background(), "Hello, {{ ok \"world\" }}")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world", s)

	// Failure to parse
	s, err = tr.Get(context.Background(), "Hello, {{ fail ")
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"cannot parse template <<< Hello, {{ fail  >>>: template: config:1: unclosed action")

	// Failure to expand
	s, err = tr.Get(context.Background(), "Hello, {{ fail \"world\" }}")
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		`cannot expand template <<< Hello, {{ fail "world" }} >>>: template: config:1:10: executing "config" at <fail "world">: error calling fail: __fail__`)
}

type testTemplateFuncs struct {
	txtFuncMap template.FuncMap
}

func (funcs *testTemplateFuncs) TxtFuncMap(ctx context.Context) template.FuncMap {
	return funcs.txtFuncMap
}
