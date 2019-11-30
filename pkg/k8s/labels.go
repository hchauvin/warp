// k8s implements Kubernetes-specific code.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package k8s

import "strings"

const (
	// StackLabel is a label put on all the Kubernetes resources a stack.  This label
	// indicates ownership, and can be used for resource pruning (kubectl --prune)
	// and deletion, when the stack is removed.
	StackLabel = "warp.stack"

	// ServiceLabel is a label put on a Service resource to identify it.  Typically,
	// the service name (coming from metadata.name) contains some stack-specific
	// prefix, which makes it difficult to refer to a service by its name.
	ServiceLabel = "warp.service"

	// NameLabel is a label put on any resource to identify it.  Typically,
	// the resource name (coming from metadata.name) contains some stack-specific
	// prefix, which makes it difficult to refer to a resource by its name.
	NameLabel = "warp.name"
)

// Labels maps labels to values.  Combining Labels with the String function gives
// a convenient way to specify a Kubernetes selector.
type Labels map[string]string

// String converts a Labels map to a Kubernetes selector.
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
