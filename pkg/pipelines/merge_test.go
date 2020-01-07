// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeCommands(t *testing.T) {
	p := &Pipeline{
		Commands: []Command{
			{Name: "foo", Tags: []string{"foo1"}},
			{Name: "foo", Tags: []string{"foo2"}},
			{Name: "bar", Tags: []string{"bar"}},
		},
	}

	err := p.mergeCommands()
	assert.NoError(t, err)

	expectedCommands := []Command{
		{Name: "foo", Tags: []string{"foo1", "foo2"}},
		{Name: "bar", Tags: []string{"bar"}},
	}

	assert.ElementsMatch(t, expectedCommands, p.Commands)
}

func TestMergeSetups(t *testing.T) {
	p := &Pipeline{
		Setups: Setups{
			{Name: "foo", Env: []string{"foo1"}},
			{Name: "foo", Env: []string{"foo2"}},
			{Name: "bar", Env: []string{"bar"}},
		},
	}

	err := p.mergeSetups()
	assert.NoError(t, err)

	expectedSetups := Setups{
		{Name: "foo", Env: []string{"foo1", "foo2"}},
		{Name: "bar", Env: []string{"bar"}},
	}

	assert.ElementsMatch(t, expectedSetups, p.Setups)
}
