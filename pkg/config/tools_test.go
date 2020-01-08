// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package config

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestToolPath(t *testing.T) {
	path, err := toolPath(Kustomize, "/workspace", "/absolute")
	assert.NoError(t, err)
	assert.Equal(t, "/absolute", filepath.ToSlash(path))

	path, err = toolPath(Kustomize, "/workspace", "program-in-path")
	assert.NoError(t, err)
	assert.Equal(t, "program-in-path", path)

	path, err = toolPath(Kustomize, "/workspace", "./relative")
	assert.NoError(t, err)
	assert.Equal(t, "/workspace/relative", filepath.ToSlash(path))

	path, err = toolPath(Kustomize, "/workspace", "relative/path")
	assert.NoError(t, err)
	assert.Equal(t, "/workspace/relative/path", filepath.ToSlash(path))
}
