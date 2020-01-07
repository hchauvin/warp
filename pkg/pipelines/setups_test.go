// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExpandSetups(t *testing.T) {
	p := &Pipeline{
		Setups: Setups{
			{Name: "foo", Env: []string{"env1"}},
			{Name: "bar", Bases: []string{"foo"}, Env: []string{"env2"}},
		},
	}

	err := expandSetups(p)
	assert.NoError(t, err)

	assert.ElementsMatch(t, []string{"env1"}, p.Setups.MustGet("foo").Env)
	assert.ElementsMatch(t, []string{"env1", "env2"}, p.Setups.MustGet("bar").Env)
}

func TestSetupsCycle(t *testing.T) {
	p := &Pipeline{
		Setups: Setups{
			{Name: "foo", Bases: []string{"bar"}},
			{Name: "bar", Bases: []string{"foo"}},
		},
	}

	err := expandSetups(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}
