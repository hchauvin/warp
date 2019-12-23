// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package templates

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"text/template"
)

func TestOS(t *testing.T) {
	tpl, err := template.New("config").
		Funcs(TxtFuncMap()).
		Parse("Hello, {{ os }}!")
	assert.NoError(t, err)

	data := map[string]interface{}{}
	w := &bytes.Buffer{}
	err = tpl.Execute(w, data)
	assert.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("Hello, %s!", runtime.GOOS), w.String())
}
