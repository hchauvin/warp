// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import (
	"fmt"
	"github.com/imdario/mergo"
)

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
