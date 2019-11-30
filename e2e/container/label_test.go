// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package container

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLabel(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "1" {
		t.Skip()
	}

	godotenv.Load("../../.env")

	err := os.MkdirAll("../../examples/.warp", 0777)
	assert.NoError(t, err)

	if err := os.Setenv("BAR", "__bar__"); err != nil {
		t.Fatal(err)
	}

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "container/label",
		Tail:         true,
		Run:          []string{"test"},
		Rm:           false,
	})
	assert.NoError(t, err)
}
