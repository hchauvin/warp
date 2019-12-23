// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package names deals with stack names.  A stack name can be composed of a
// family name and a short name in the case multiple instances of the
// same stack are permitted.
package names

import "strings"

// Name is the name of a stack.
type Name struct {
	Family    string
	ShortName string
}

func (name Name) String() string {
	if name.Family != "" {
		return name.Family + "_" + name.ShortName
	}
	return name.ShortName
}

// DNSName returns the name adapted for DNS use.
func (name Name) DNSName() string {
	return strings.ReplaceAll(name.String(), "_", "-")
}
