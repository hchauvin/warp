// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package fsreporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommandNameToPath(t *testing.T) {
	assert.Equal(t, "foo_bar", commandNameToPath("foo/bar"))
	assert.Equal(t, "__foo_bar", commandNameToPath("./foo/bar"))
	assert.Equal(t, "tag/foo_bar", commandNameToPath("[tag]foo/bar"))
	assert.Equal(t, "tag1/tag2/foo_bar", commandNameToPath("[tag1][tag2]foo/bar"))
	assert.Equal(t, "tag/Hello_world", commandNameToPath("Hello[tag] world"))
}
