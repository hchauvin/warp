// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetups(t *testing.T) {
	setups := Setups{
		{Name: "foo", Env: []string{"foo_dev"}},
		{Name: "bar", Env: []string{"bar_dev"}},
	}

	assert.ElementsMatch(t, []string{"foo", "bar"}, setups.Names())

	setup, err := setups.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo_dev", setup.Env[0])

	_, err = setups.Get("unknown")
	assert.Error(t, err)
}
