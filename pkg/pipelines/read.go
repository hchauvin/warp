// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/imdario/mergo"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"strings"
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
	p, err := read(config, path, fs, make(map[string]struct{}))
	if err != nil {
		return nil, err
	}

	// Let's merge the commands by the 'name' key
	if err := p.mergeCommands(); err != nil {
		return nil, err
	}

	// Let's merge the setups by the 'name' key
	if err := p.mergeSetups(); err != nil {
		return nil, err
	}

	// Let's expand the setups with the content of their bases
	if err := expandSetups(p); err != nil {
		return nil, err
	}

	// Generic validation
	if err := validate.Struct(p); err != nil {
		return nil, fmt.Errorf("%s: invalid pipeline config: %v", path, err)
	}

	// Ad hoc validation
	if p.Stack.Name == "" && p.Stack.Family == "" {
		return nil, fmt.Errorf("%s: either stack.name or stack.family must be given", path)
	}
	for i, setup := range p.Setups {
		dedupedHooks, err := dedupeAndValidateCommandHooks(setup.Before)
		if err != nil {
			return nil, err
		}
		p.Setups[i].Before = dedupedHooks
	}

	// Manifest parsing
	if p.Deploy.Container != nil && p.Deploy.Container.Manifest != "" {
		manifestPath := config.Path(p.Deploy.Container.Manifest)
		p.Deploy.Container.ParsedManifest, err = parseContainerManifest(fs, manifestPath)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s: cannot parse container manifest '%s': %v", path, manifestPath, err)
		}
	}

	return p, nil
}

func read(
	config *config.Config,
	path string,
	fs afero.Fs,
	visitedPaths visited,
) (*Pipeline, error) {
	if _, ok := visitedPaths[path]; ok {
		return nil, fmt.Errorf("pipeline bases: cycle detected: %s", visitedPaths.String())
	}

	fullPath := config.Path(path)
	if isDir, err := afero.IsDir(fs, fullPath); isDir && err == nil {
		fullPath = filepath.Join(fullPath, "pipeline.yml")
	}

	yamlPipeline, err := afero.ReadFile(fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read pipeline file %s: %v", path, err)
	}

	pipeline := &Pipeline{}
	if err := yaml.Unmarshal(yamlPipeline, pipeline); err != nil {
		return nil, err
	}

	if len(pipeline.Bases) > 0 {
		nextVisitedPaths := make(map[string]struct{})
		for p := range visitedPaths {
			nextVisitedPaths[p] = struct{}{}
		}
		nextVisitedPaths[path] = struct{}{}

		mergedPipeline := &Pipeline{}
		for _, base := range pipeline.Bases {
			basePipeline, err := read(config, base, fs, nextVisitedPaths)
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
	err := mergo.Merge(dest, patch, mergo.WithOverride, mergo.WithAppendSlice)
	if err != nil {
		return err
	}
	return nil
}

type visited map[string]struct{}

func (v visited) String() string {
	var s strings.Builder
	i := 0
	for node := range v {
		if i > 0 {
			s.WriteString(" -> ")
		}
		i++
		s.WriteString(node)
	}
	return s.String()
}
