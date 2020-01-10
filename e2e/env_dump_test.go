// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package e2e

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestEnvDump(t *testing.T) {
	godotenv.Load("../.env")

	err := os.MkdirAll("../examples/.warp", 0777)
	assert.NoError(t, err)

	dumpEnvFile, err := ioutil.TempFile("", "dump_env")
	assert.NoError(t, err)
	defer os.Remove(dumpEnvFile.Name())

	err = os.Setenv("ENV_DUMP_PATH", dumpEnvFile.Name())
	assert.NoError(t, err)

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "env_dump",
		Tail:         true,
		Run:          []string{"test"},
		Setup:        "setup0",
		PersistEnv:   false,
		DumpEnv:      dumpEnvFile.Name(),
	})
	assert.NoError(t, err)
}
