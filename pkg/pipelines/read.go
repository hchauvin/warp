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
	"reflect"
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

	// Let's merge the setups by the 'name' key
	var setupNames []string
	setupsByName := make(map[string]Setup)
	hasMerged = false
	for _, setup := range p.Setups {
		prev, ok := setupsByName[setup.Name]
		if !ok {
			setupsByName[setup.Name] = setup
			setupNames = append(setupNames, setup.Name)
		} else {
			hasMerged = true
			err = mergo.Merge(
				&prev,
				&setup,
				mergo.WithOverride,
				mergo.WithAppendSlice)
			if err != nil {
				return nil, err
			}
			setupsByName[setup.Name] = prev
		}
	}
	if hasMerged {
		p.Setups = nil
		for _, name := range setupNames {
			p.Setups = append(p.Setups, setupsByName[name])
		}
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

func expandSetups(p *Pipeline) error {
	for i := range p.Setups {
		expanded, err := expandSetup(p.Setups, p.Setups[i].Name, make(map[string]struct{}))
		if err != nil {
			return err
		}
		p.Setups[i] = *expanded
	}
	return nil
}

func expandSetup(setups Setups, name string, visitedSetups visited) (expanded *Setup, err error) {
	setup, err := setups.Get(name)
	if err != nil {
		return nil, err
	}
	if len(setup.Bases) == 0 {
		return setup, nil
	}

	if _, ok := visitedSetups[name]; ok {
		return nil, fmt.Errorf("cycle detected: %s", visitedSetups.String())
	}

	nextVisitedSetups := make(map[string]struct{})
	for name := range visitedSetups {
		nextVisitedSetups[name] = struct{}{}
	}
	nextVisitedSetups[setup.Name] = struct{}{}

	mergedSetup := &Setup{}
	for _, base := range setup.Bases {
		baseSetup, err := expandSetup(setups, base, nextVisitedSetups)
		if err != nil {
			return nil, err
		}
		if err := mergeSetups(mergedSetup, baseSetup); err != nil {
			return nil, fmt.Errorf("could not merge setup '%s' with base '%s': %v", name, base, err)
		}
	}

	if err := mergeSetups(mergedSetup, setup); err != nil {
		return nil, fmt.Errorf("could not merge setup '%s' with its bases: %v", name, err)
	}

	return mergedSetup, nil
}

func mergeSetups(dest, patch *Setup) error {
	err := mergo.Merge(dest, patch, mergo.WithOverride, mergo.WithAppendSlice)
	if err != nil {
		return err
	}
	return nil
}

func dedupeAndValidateCommandHooks(hooks []CommandHook) (dedupedHooks []CommandHook, err error) {
	// Validate the hooks individually
	for _, hook := range hooks {
		if err := validateCommandHook(&hook); err != nil {
			return nil, err
		}
	}

	namedHooks := make(map[string]CommandHook)
	for _, hook := range hooks {
		if hook.Name != "" {
			if duplicateHook, ok := namedHooks[hook.Name]; ok {
				if reflect.DeepEqual(duplicateHook, hook) {
					continue
				}
				return nil, fmt.Errorf("multiple hooks are named '%s'", hook.Name)
			}
			namedHooks[hook.Name] = hook
		}
		dedupedHooks = append(dedupedHooks, hook)
	}

	// Validate the DAG.  We must be able to visit all the hooks, and there
	// should be no loop.
	for i, hook := range dedupedHooks {
		hookName := hook.Name
		if hookName == "" {
			hookName = fmt.Sprintf("#%d", i)
		}
		if err := visitHookDependencies(namedHooks, hookName, &hook, make(map[string]struct{})); err != nil {
			return nil, err
		}
	}

	return dedupedHooks, nil
}

func visitHookDependencies(namedHooks map[string]CommandHook, hookName string, hook *CommandHook, visited map[string]struct{}) error {
	if _, ok := visited[hook.Name]; ok {
		path := make([]string, len(visited))
		for id := range visited {
			path = append(path, id)
		}
		return fmt.Errorf("cycle detected: %s", strings.Join(path, " -> "))
	}
	for _, dep := range hook.DependsOn {
		nextVisited := make(map[string]struct{}, len(visited)+1)
		for id := range visited {
			nextVisited[id] = struct{}{}
		}
		nextVisited[dep] = struct{}{}
		nextHook, ok := namedHooks[dep]
		if !ok {
			return fmt.Errorf(
				"hook '%s' depends on hook '%s', but this hook does not exist",
				hookName,
				dep)
		}
		if err := visitHookDependencies(namedHooks, nextHook.Name, &nextHook, visited); err != nil {
			return err
		}
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
