// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package templates

import (
	"io/ioutil"
	"runtime"
	ttemplate "text/template"
)

// TxtFuncMap returns a 'text/template'.FuncMap
func TxtFuncMap() ttemplate.FuncMap {
	return GenericFuncMap()
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	gfm := make(map[string]interface{}, len(genericMap))
	for k, v := range genericMap {
		gfm[k] = v
	}
	return gfm
}

var genericMap = map[string]interface{}{
	"os": func() string { return runtime.GOOS },
	"readTextFile": func(path string) (string, error) {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
}
