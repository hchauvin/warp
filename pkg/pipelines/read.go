// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package pipelines

import (
	"encoding/json"
	"errors"
	"fmt"
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
	p, err := read(config, path, fs, make(map[string]struct{}))
	if err != nil {
		return nil, err
	}

	// Let's merge the commands by the 'name' key
	var commandNames []string
	commandsByName := make(map[string]Command)
	hasMerged := false
	for _, command := range p.Commands {
		prev, ok := commandsByName[command.Name]
		if !ok {
			commandsByName[command.Name] = command
			commandNames = append(commandNames, command.Name)
		} else {
			hasMerged = true
			err = mergo.Merge(
				&prev,
				&command,
				mergo.WithOverride,
				mergo.WithAppendSlice)
			if err != nil {
				return nil, err
			}
			commandsByName[command.Name] = prev
		}
	}
	if hasMerged {
		p.Commands = nil
		for _, name := range commandNames {
			p.Commands = append(p.Commands, commandsByName[name])
		}
	}

	// Generic validation
	if err := validate.Struct(p); err != nil {
		return nil, fmt.Errorf("%s: invalid pipeline config: %v", path, err)
	}

	// Ad hoc validation
	if p.Stack.Name == "" && p.Stack.Family == "" {
		return nil, fmt.Errorf("%s: either stack.name or stack.family must be given", path)
	}
	for _, command := range p.Commands {
		for _, hook := range command.Before {
			if err := validateCommandHook(&hook); err != nil {
				return nil, err
			}
		}
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

func validateCommandHook(hook *CommandHook) error {
	actionCount := 0
	if hook.WaitFor != nil {
		actionCount++
	}
	if hook.Run != nil {
		actionCount++
	}
	if hook.HTTPGet != nil {
		actionCount++
	}
	if actionCount != 1 {
		return fmt.Errorf("there must be one and only one action per command hook")
	}
	return nil
}
