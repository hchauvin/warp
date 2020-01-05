// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/run/batch/fsreporter"
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

func TestBatchReport(t *testing.T) {
	setUpBatch(t)
	defer tearDownBatch()

	reportDir := filepath.Join(os.Getenv("OUTPUT_DIR"), "_batch_report")

	err := warp.Batch(context.Background(), &warp.BatchCfg{
		WorkingDir:           "../examples",
		ConfigPath:           ".warprc.toml",
		BatchPath:            "batch/batch.yml",
		Parallelism:          1,
		MaxStacksPerPipeline: 1,
		Tags:                 "",
		Focus:                "report",
		Bail:                 true,
		Advisory:             false,
		Report:               reportDir,
		Stream:               true,
	})
	assert.NoError(t, err)

	assertRun(t, "report")

	fmt.Printf("Report dir: %s\n", reportDir)

	reportJson, err := ioutil.ReadFile(filepath.Join(reportDir, "report.json"))
	assert.NoError(t, err, "cannot open report.json")

	var report fsreporter.Report
	err = json.Unmarshal(reportJson, &report)
	assert.NoError(t, err, "cannot unmarshal report")

	assert.Len(t, report.EnvironmentSetupResults, 0)
	assert.Len(t, report.Results, 1)
	assert.Equal(t, "report", report.Results[0].Name)

	logFiles, err := ioutil.ReadDir(filepath.Join(reportDir, "log"))
	assert.NoError(t, err, "cannot read log directory")
	assert.Len(t, logFiles, 1)

	log, err := ioutil.ReadFile(filepath.Join(reportDir, "log/report.1.txt"))
	assert.Contains(t, string(log), "__stdout__")
	assert.Contains(t, string(log), "__stderr__")
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
