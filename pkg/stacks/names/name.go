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
