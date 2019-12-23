// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package kustomize

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestManifest(t *testing.T) {
	godotenv.Load("../../.env")

	err := os.MkdirAll("../../examples/.warp", 0777)
	assert.NoError(t, err)

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "kustomize/manifest",
		Tail:         true,
		Run:          []string{"test"},
	})
	assert.NoError(t, err)
}
