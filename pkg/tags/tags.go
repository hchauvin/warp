// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package tags implements tag-based filtering.
package tags

import (
	"fmt"
	"strings"
)

// Filter holds a compiled tag filter.
type Filter struct {
	includeTagSet map[string]struct{}
	excludeTagSet map[string]struct{}
}

// CompileFilter compiles a tag filter.
func CompileFilter(filter string) (*Filter, error) {
	compiled := &Filter{
		includeTagSet: make(map[string]struct{}),
		excludeTagSet: make(map[string]struct{}),
	}
	if filter == "" {
		return compiled, nil
	}
	for i, component := range strings.Split(filter, ",") {
		if len(component) == 0 {
			return nil, fmt.Errorf("tag filter component #%d: cannot be empty", i)
		}
		if component[0] == '!' || component[0] == '-' {
			compiled.excludeTagSet[component[1:]] = struct{}{}
		} else {
			compiled.includeTagSet[component] = struct{}{}
		}
	}
	return compiled, nil
}

// Apply applies a tag filter to a slice of tags.
func (filter *Filter) Apply(tags []string) bool {
	if len(tags) == 0 {
		if len(filter.includeTagSet) == 0 {
			return true
		}
		return false
	}

	if len(filter.excludeTagSet) > 0 {
		for _, tag := range tags {
			if _, ok := filter.excludeTagSet[tag]; ok {
				return false
			}
		}
	}

	if len(filter.includeTagSet) > 0 {
		for _, tag := range tags {
			if _, ok := filter.includeTagSet[tag]; ok {
				return true
			}
		}
		return false
	}

	return true
}
