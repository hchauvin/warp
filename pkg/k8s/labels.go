// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import "strings"

const (
	StackLabel   = "warp.stack"
	ServiceLabel = "warp.service"
	NameLabel    = "warp.name"
)

type Labels map[string]string

func (lbls Labels) String() string {
	var b strings.Builder
	first := true
	for k, v := range lbls {
		if !first {
			b.WriteRune(',')
		}
		first = false
		b.WriteString(k)
		b.WriteRune('=')
		b.WriteString(v)
	}
	return b.String()
}
