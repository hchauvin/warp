// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var waitFor = &WaitForHook{Resources: []WaitForResourceKind{Endpoints}}

func TestHooksAreDeduped(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", TimeoutSeconds: 1, WaitFor: waitFor},
		{Name: "foo", TimeoutSeconds: 1, WaitFor: waitFor}, // deep equal
		{Name: "bar", WaitFor: waitFor},
	}

	ans, err := dedupeAndValidateCommandHooks(hooks)
	assert.NoError(t, err)

	expected := []CommandHook{
		{Name: "foo", TimeoutSeconds: 1, WaitFor: waitFor},
		{Name: "bar", WaitFor: waitFor},
	}

	assert.ElementsMatch(t, expected, ans)
}

func TestHooksNameClash(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", WaitFor: waitFor},
		{Name: "foo", TimeoutSeconds: 1, WaitFor: waitFor}, // not deep equal
	}

	_, err := dedupeAndValidateCommandHooks(hooks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple hooks are named")
}

func TestHooksCycle(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", DependsOn: []string{"bar"}, WaitFor: waitFor},
		{Name: "bar", DependsOn: []string{"foo"}, WaitFor: waitFor},
	}

	_, err := dedupeAndValidateCommandHooks(hooks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestHooksAnonymous(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", WaitFor: waitFor},
		{DependsOn: []string{"foo"}, WaitFor: waitFor},
	}

	_, err := dedupeAndValidateCommandHooks(hooks)
	assert.NoError(t, err)
}

func TestHooksDepNotExists(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", DependsOn: []string{"unknown"}, WaitFor: waitFor},
	}

	_, err := dedupeAndValidateCommandHooks(hooks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "this hook does not exist")
}

func TestHooksMultipleActions(t *testing.T) {
	hooks := []CommandHook{
		{Name: "foo", WaitFor: waitFor, HTTPGet: &HTTPGet{}},
	}

	_, err := dedupeAndValidateCommandHooks(hooks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "there must be one and only one action per command hook")
}

func TestValidateWaitForHook(t *testing.T) {
	h := &WaitForHook{Resources: []WaitForResourceKind{Endpoints}}
	err := h.validate()
	assert.NoError(t, err)

	h = &WaitForHook{Resources: nil}
	err = h.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected at least one resource kind to wait for")

	h = &WaitForHook{Resources: []WaitForResourceKind{Endpoints, "foo"}}
	err = h.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource kind 'foo'")
}
