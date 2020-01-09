// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCacheMemoize(t *testing.T) {
	funcs := newTemplateFuncsCache()

	called := 0
	f := func() (string, error) {
		called++
		return "bar", nil
	}

	ans, err := funcs.memoize(
		f,
		"func",
		"a",
		0)
	assert.NoError(t, err)
	assert.Equal(t, "bar", ans)
	assert.Equal(t, 1, called)

	// Call with different arguments
	ans, err = funcs.memoize(
		f,
		"func",
		"a",
		1000)
	assert.NoError(t, err)
	assert.Equal(t, "bar", ans)
	assert.Equal(t, 2, called)

	// Call with same argument: function is not called again
	ans, err = funcs.memoize(
		f,
		"func",
		"a",
		1000)
	assert.NoError(t, err)
	assert.Equal(t, "bar", ans)
	assert.Equal(t, 2, called)
}

func TestCacheMemoizeWithError(t *testing.T) {
	funcs := newTemplateFuncsCache()

	called := 0
	f := func() (string, error) {
		called++
		return "", errors.New("__error__")
	}

	_, err := funcs.memoize(
		f,
		"func",
		"a",
		0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "__error__")
	assert.Equal(t, 1, called)

	// Call with same argument: function is not called again, even if
	// it errored
	_, err = funcs.memoize(
		f,
		"func",
		"a",
		0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "__error__")
	assert.Equal(t, 1, called)
}
