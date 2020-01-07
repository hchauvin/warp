// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"github.com/hchauvin/warp/pkg/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestReadOneSpec(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/folder/pipeline.yml", onePipelineBytes, 0666)
	assert.NoError(t, err)

	p, err := ReadFs(cfg, "folder/pipeline.yml", fs)
	assert.NoError(t, err)

	assert.EqualValues(t, &onePipeline, p)
}

func TestMergeSpecs(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/folder/pipeline.yml", onePipelineBytes, 0666)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, "/workspace/overlay/pipeline.yml", overlayPipelineBytes, 0666)
	assert.NoError(t, err)

	p, err := ReadFs(cfg, "overlay/pipeline.yml", fs)
	assert.NoError(t, err)

	assert.EqualValues(t, &overlayPipeline, p)
}

func TestParseContainerManifest(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/folder/pipeline.yml", pipelineWithContainerManifestBytes, 0666)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, "/workspace/container/manifest.json", containerManifestBytes, 0666)
	assert.NoError(t, err)

	p, err := ReadFs(cfg, "folder/pipeline.yml", fs)
	assert.NoError(t, err)

	assert.EqualValues(t, &pipelineWithContainerManifest, p)
}

func TestParseInvalidContainerManifest(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/folder/pipeline.yml", pipelineWithContainerManifestBytes, 0666)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, "/workspace/container/manifest.json", invalidContainerManifestBytes, 0666)
	assert.NoError(t, err)

	_, err = ReadFs(cfg, "folder/pipeline.yml", fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse container manifest")
}

func TestCycle(t *testing.T) {
	var base = []byte(`
bases: ['overlay']
stack:
  family: foo`)
	var overlay = []byte(`
bases: ['base']
`)

	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/base/pipeline.yml", base, 0666)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, "/workspace/overlay/pipeline.yml", overlay, 0666)
	assert.NoError(t, err)

	_, err = ReadFs(cfg, "overlay", fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestPipelineFileNotExist(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}
	fs := afero.NewMemMapFs()

	_, err := ReadFs(cfg, "folder/pipeline.yml", fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read pipeline file")
}

func TestInvalidPipeline(t *testing.T) {
	t.Skip() // TODO

	var pipeline = []byte(`
stack:
  family: foo
dev:
  portForward:
  - {}
`)

	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/base/pipeline.yml", pipeline, 0666)
	assert.NoError(t, err)

	_, err = ReadFs(cfg, "base/pipeline.yml", fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pipeline config")
}

func TestFamilyOrName(t *testing.T) {
	var pipeline = []byte(`
stack: {}
`)

	cfg := &config.Config{WorkspaceDir: "/workspace"}

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/base/pipeline.yml", pipeline, 0666)
	assert.NoError(t, err)

	_, err = ReadFs(cfg, "base/pipeline.yml", fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either stack.name or stack.family must be given")
}

var onePipelineBytes = []byte(`
stack:
  family: foo
deploy:
  kustomize:
    path: folder/kustomization.yml
`)

var onePipeline = Pipeline{
	Path: filepath.FromSlash("/workspace/folder/pipeline.yml"),
	Stack: Stack{
		Family: "foo",
	},
	Deploy: Deploy{
		Kustomize: &Kustomize{
			Path: "folder/kustomization.yml",
		},
	},
}

var overlayPipelineBytes = []byte(`
bases: ['folder/pipeline.yml']
stack:
  family: bar
`)

var overlayPipeline = Pipeline{
	Path:  filepath.FromSlash("/workspace/overlay/pipeline.yml"),
	Bases: []string{"folder/pipeline.yml"},
	Stack: Stack{
		Family: "bar",
	},
	Deploy: Deploy{
		Kustomize: &Kustomize{
			Path: "folder/kustomization.yml",
		},
	},
}

var pipelineWithContainerManifestBytes = []byte(`
stack:
  family: foo
deploy:
  container:
    manifest: container/manifest.json
`)

var containerManifestBytes = []byte(`
{
  "image": {
    "ref": "replacement"
  }
}`)

var invalidContainerManifestBytes = []byte(`
__invalid__
`)

var pipelineWithContainerManifest = Pipeline{
	Path: filepath.FromSlash("/workspace/folder/pipeline.yml"),
	Stack: Stack{
		Family: "foo",
	},
	Deploy: Deploy{
		Container: &Container{
			Manifest: "container/manifest.json",
			ParsedManifest: ContainerManifest{
				"image": ContainerManifestEntry{
					Ref: "replacement",
				},
			},
		},
	},
}
