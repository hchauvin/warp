// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package e2e

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHelm(t *testing.T) {
	godotenv.Load("../.env")

	err := warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "helm",
		Tail:         true,
		Run:          []string{"test"},
	})
	assert.NoError(t, err)
}
