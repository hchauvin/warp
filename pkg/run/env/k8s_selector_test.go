// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceSelector(t *testing.T) {
	name := names.Name{Family: "foo", ShortName: "0"}

	selector, err := serviceSelector(name, "service")
	assert.NoError(t, err)
	assert.Equal(t, "warp.stack=foo-0,warp.service=service", selector)

	selector, err = serviceSelector(name, "foo=bar")
	assert.NoError(t, err)
	assert.Equal(t, "warp.stack=foo-0,foo=bar", selector)

	selector, err = serviceSelector(name, "::foo=bar,qux=wobble")
	assert.NoError(t, err)
	assert.Equal(t, "foo=bar,qux=wobble", selector)
}
