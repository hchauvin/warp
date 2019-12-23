// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import (
	"github.com/hchauvin/warp/pkg/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRead(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/folder/batch.yml", batchBytes, 0666)
	assert.NoError(t, err)

	b, err := ReadFs(cfg, "folder/batch.yml", fs)
	assert.NoError(t, err)

	assert.EqualValues(t, &batch, b)
}

var batchBytes = []byte(`
pipelines:
  - name: foo
    path: bar
    setup: setup
commands:
  - name: cmd
`)

var batch = Batch{
	Pipelines: []Pipeline{
		{
			Name:  "foo",
			Path:  "bar",
			Setup: "setup",
		},
	},
	Commands: []BatchCommand{
		{
			Name: "cmd",
		},
	},
}
