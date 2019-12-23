// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterFocus(t *testing.T) {
	batch := Batch{
		Commands: []BatchCommand{
			{
				Name: "focus",
			},
			{
				Name: "other",
			},
		},
	}
	filtered, err := batch.Filter("", "focus")
	assert.NoError(t, err)
	assert.EqualValues(t, []BatchCommand{{Name: "focus"}}, filtered.Commands)

	_, err = batch.Filter("", "not_found")
	assert.Error(t, err)
}

func TestFilterTags(t *testing.T) {
	batch := Batch{
		Commands: []BatchCommand{
			{
				Tags: []string{"foo"},
			},
			{
				Tags: []string{"bar"},
			},
		},
	}
	filtered, err := batch.Filter("foo", "")
	assert.NoError(t, err)
	assert.EqualValues(t, []BatchCommand{{Tags: []string{"foo"}}}, filtered.Commands)
}
