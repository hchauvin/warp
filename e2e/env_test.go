// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package e2e

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	t.Skip() // TODO

	godotenv.Load("../.env")

	err := os.MkdirAll("../examples/.warp", 0777)
	assert.NoError(t, err)

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "env",
		Tail:         true,
		Run:          []string{"test"},
		Rm:           false,
	})
	assert.NoError(t, err)
}
