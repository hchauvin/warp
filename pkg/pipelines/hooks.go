// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"fmt"
	"reflect"
	"strings"
)

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
	var nextVisited map[string]struct{}
	if hook.Name == "" {
		nextVisited = visited
	} else {
		if _, ok := visited[hook.Name]; ok {
			path := make([]string, 0, len(visited))
			for id := range visited {
				path = append(path, id)
			}
			return fmt.Errorf("cycle detected: %s", strings.Join(path, " -> "))
		}

		nextVisited = make(map[string]struct{}, len(visited)+1)
		for id := range visited {
			nextVisited[id] = struct{}{}
		}
		nextVisited[hook.Name] = struct{}{}
	}

	for _, dep := range hook.DependsOn {
		nextHook, ok := namedHooks[dep]
		if !ok {
			return fmt.Errorf(
				"hook '%s' depends on hook '%s', but this hook does not exist",
				hookName,
				dep)
		}
		if err := visitHookDependencies(namedHooks, nextHook.Name, &nextHook, nextVisited); err != nil {
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
