// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package e2e

import (
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var devLocalPorts = []int{10001, 10002}

func TestDev(t *testing.T) {
	godotenv.Load("../.env")

	err := os.MkdirAll("../examples/.warp", 0777)
	if err != nil {
		t.Fatalf("cannot create .wrap output dir: %v", err)
	}

	err = os.MkdirAll("../examples/dev/www", 0777)
	if err != nil {
		t.Fatalf("cannot create www dir: %v", err)
	}

	for _, port := range devLocalPorts {
		if err := proc.KillPort(port); err != nil {
			t.Fatalf("cannot kill process listening on port %d: %v", port, err)
		}
	}

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "dev",
		Tail:         true,
		Run:          []string{"test"},
		Setup:        "setup0",
		Dev:          true,
	})
	assert.NoError(t, err)
}
