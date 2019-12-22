// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package e2e

import (
	"context"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestBatchBase(t *testing.T) {
	setUpBatch(t)
	defer tearDownBatch()

	err := warp.Batch(context.Background(), &warp.BatchCfg{
		WorkingDir:           "../examples",
		ConfigPath:           ".warprc.toml",
		BatchPath:            "batch/batch.yml",
		Parallelism:          1,
		MaxStacksPerPipeline: 1,
		Tags:                 "",
		Focus:                "base",
		Bail:                 true,
		Advisory:             false,
		Report:               "",
		Stream:               true,
	})
	assert.NoError(t, err)

	assertRun(t, "base")
}

func TestBatchDependsOn(t *testing.T) {
	setUpBatch(t)
	defer tearDownBatch()

	err := warp.Batch(context.Background(), &warp.BatchCfg{
		WorkingDir:           "../examples",
		ConfigPath:           ".warprc.toml",
		BatchPath:            "batch/batch.yml",
		Parallelism:          1,
		MaxStacksPerPipeline: 1,
		Tags:                 "depends-on",
		Focus:                "",
		Bail:                 true,
		Advisory:             false,
		Report:               "",
		Stream:               true,
	})
	assert.NoError(t, err)

	assertRun(t, "depends-on", "dependency")
}

func TestBatchPipeline(t *testing.T) {
	setUpBatch(t)
	defer tearDownBatch()

	err := warp.Batch(context.Background(), &warp.BatchCfg{
		WorkingDir:           "../examples",
		ConfigPath:           ".warprc.toml",
		BatchPath:            "batch/batch.yml",
		Parallelism:          1,
		MaxStacksPerPipeline: 1,
		Tags:                 "",
		Focus:                "pipeline",
		Bail:                 true,
		Advisory:             false,
		Report:               "",
		Stream:               true,
	})
	assert.NoError(t, err)

	assertRun(t, "pipeline")
}

func TestBatchFail(t *testing.T) {
	setUpBatch(t)
	defer tearDownBatch()

	err := warp.Batch(context.Background(), &warp.BatchCfg{
		WorkingDir:           "../examples",
		ConfigPath:           ".warprc.toml",
		BatchPath:            "batch/batch.yml",
		Parallelism:          1,
		MaxStacksPerPipeline: 1,
		Tags:                 "",
		Focus:                "fail",
		Bail:                 true,
		Advisory:             false,
		Report:               "",
		Stream:               true,
	})
	assert.Error(t, err)

	assertRun(t, "fail")
}

func setUpBatch(t *testing.T) {
	godotenv.Load("../.env")
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create output directory: %v", err)
	}
	os.Setenv("OUTPUT_DIR", dir)
}

func tearDownBatch() {
	os.RemoveAll(os.Getenv("OUTPUT_DIR"))
}

func assertRun(t *testing.T, commands ...string) {
	for _, cmd := range commands {
		file := filepath.Join(os.Getenv("OUTPUT_DIR"), cmd)
		_, err := os.Stat(file)
		assert.NoError(t, err, "command '%s' has not been run: file '%s' does not exist", cmd, file)
	}
}
