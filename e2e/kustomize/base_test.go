// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package kustomize

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Printf("Entered\n")
	exitVal := m.Run()
	fmt.Printf("Exited\n")
	os.Exit(exitVal)
}

func TestBase(t *testing.T) {
	godotenv.Load("../../.env")

	err := os.MkdirAll("../../examples/.warp", 0777)
	assert.NoError(t, err)

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "kustomize/base",
		Tail:         true,
		Run:          []string{"test"},
	})
	assert.NoError(t, err)
}
