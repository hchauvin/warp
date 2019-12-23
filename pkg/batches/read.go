// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
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
	fullPath := config.Path(path)

	yamlBatch, err := afero.ReadFile(fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read batch file %s: %v", path, err)
	}

	batch := &Batch{}
	if err := yaml.Unmarshal(yamlBatch, batch); err != nil {
		return nil, err
	}

	// Generic validation
	if err := validate.Struct(batch); err != nil {
		return nil, fmt.Errorf("%s: invalid batch config: %v", path, err)
	}

	return batch, nil
}
