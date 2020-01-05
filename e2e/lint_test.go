// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package e2e

import (
	"context"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLint(t *testing.T) {
	godotenv.Load("../.env")

	err := warp.Lint(context.Background(), &warp.LintCfg{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "lint/fail",
	})
	assert.Error(t, err)

	err = warp.Lint(context.Background(), &warp.LintCfg{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "lint/pass",
	})
	assert.NoError(t, err)
}
