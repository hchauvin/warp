// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import (
	"errors"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

// Read reads a pipeline from a file on the local file system.  The path
// is given relative to the workspace root (see config.Config.WorkspaceDir).
//
// The resulting pipeline is expanded: all the base pipelines that are
// referenced in the pipeline are merged.
func Read(config *config.Config, path string) (*Batch, error) {
	return ReadFs(config, path, afero.NewOsFs())
}

// ReadFs does the same as Read but on an arbitrary afero file system.
func ReadFs(config *config.Config, path string, fs afero.Fs) (*Batch, error) {
	p, err := read(config, path, fs, make(map[string]struct{}))
	if err != nil {
		return nil, err
	}

	// Generic validation
	if err := validate.Struct(p); err != nil {
		return nil, fmt.Errorf("%s: invalid pipeline config: %v", path, err)
	}

	return p, nil
}

func read(
	config *config.Config,
	path string,
	fs afero.Fs,
	visitedPaths map[string]struct{},
) (*Batch, error) {
	if _, ok := visitedPaths[path]; ok {
		return nil, errors.New("pipeline bases: cycle detected")
	}

	fullPath := config.Path(path)
	if isDir, err := afero.IsDir(fs, fullPath); isDir && err == nil {
		fullPath = filepath.Join(fullPath, "pipeline.yml")
	}

	yamlBatch, err := afero.ReadFile(fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read pipeline file %s: %v", path, err)
	}

	batch := &Batch{}
	if err := yaml.Unmarshal(yamlBatch, batch); err != nil {
		return nil, err
	}

	return batch, nil
}