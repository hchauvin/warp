// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package pipelines

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/imdario/mergo"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

// Read reads a pipeline from a file on the local file system.  The path
// is given relative to the workspace root (see config.Config.WorkspaceDir).
//
// The resulting pipeline is expanded: all the base pipelines that are
// referenced in the pipeline are merged.
func Read(config *config.Config, path string) (*Pipeline, error) {
	return ReadFs(config, path, afero.NewOsFs())
}

// ReadFs does the same as Read but on an arbitrary afero file system.
func ReadFs(config *config.Config, path string, fs afero.Fs) (*Pipeline, error) {
	return read(config, path, fs, make(map[string]struct{}))
}

func read(
	config *config.Config,
	path string,
	fs afero.Fs,
	visitedPaths map[string]struct{},
) (*Pipeline, error) {
	if _, ok := visitedPaths[path]; ok {
		return nil, errors.New("pipeline bases: cycle detected")
	}

	fullPath := config.Path(path)
	if isDir, err := afero.IsDir(fs, fullPath); isDir && err == nil {
		fullPath = filepath.Join(fullPath, "pipeline.yml")
	}

	yamlPipeline, err := afero.ReadFile(fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read pipeline: %v", err)
	}

	pipeline := &Pipeline{}
	if err := yaml.Unmarshal(yamlPipeline, pipeline); err != nil {
		return nil, err
	}

	if err := validator.New().Struct(pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline config: %v", err)
	}

	if len(pipeline.Bases) > 0 {
		nextVisitedPaths := make(map[string]struct{})
		for p := range visitedPaths {
			nextVisitedPaths[p] = struct{}{}
		}
		nextVisitedPaths[path] = struct{}{}

		mergedPipeline := &Pipeline{}
		for _, base := range pipeline.Bases {
			basePipeline, err := read(config, base, fs, visitedPaths)
			if err != nil {
				return nil, err
			}
			if err := mergePipelines(mergedPipeline, basePipeline); err != nil {
				return nil, fmt.Errorf("could not merge pipeline '%s' with base '%s': %v", path, base, err)
			}
		}

		if err := mergePipelines(mergedPipeline, pipeline); err != nil {
			return nil, fmt.Errorf("could not merge pipeline '%s' with its bases: %v", path, err)
		}

		pipeline = mergedPipeline
	}

	pipeline.Path = fullPath

	if pipeline.Deploy.Container != nil && pipeline.Deploy.Container.Manifest != "" {
		path := config.Path(pipeline.Deploy.Container.Manifest)
		pipeline.Deploy.Container.ParsedManifest, err = parseContainerManifest(fs, path)
		if err != nil {
			return nil, fmt.Errorf("cannot parse container manifest '%s': %v", path, err)
		}
	}

	return pipeline, nil
}

func parseContainerManifest(fs afero.Fs, path string) (ContainerManifest, error) {
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	var manifest ContainerManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

func mergePipelines(dest, patch *Pipeline) error {
	return mergo.Merge(dest, patch, mergo.WithOverride, mergo.WithAppendSlice)
}
