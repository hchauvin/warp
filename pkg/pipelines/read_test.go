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
