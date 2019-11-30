// names deals with stack names.  A stack name can be composed of a
// family nalme and a short name in the case multiple instances of the
// same stack are permitted.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package names

import "strings"

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

func (name Name) DNSName() string {
	return strings.ReplaceAll(name.String(), "_", "-")
}
