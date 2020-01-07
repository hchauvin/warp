// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package pipelines

import "github.com/imdario/mergo"

// mergeCommands merges the commands by the 'name' key
func (p *Pipeline) mergeCommands() error {
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
			err := mergo.Merge(
				&prev,
				&command,
				mergo.WithOverride,
				mergo.WithAppendSlice)
			if err != nil {
				return err
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

	return nil
}

// mergeSetups merges the setups by the 'name' key
func (p *Pipeline) mergeSetups() error {
	var setupNames []string
	setupsByName := make(map[string]Setup)
	hasMerged := false
	for _, setup := range p.Setups {
		prev, ok := setupsByName[setup.Name]
		if !ok {
			setupsByName[setup.Name] = setup
			setupNames = append(setupNames, setup.Name)
		} else {
			hasMerged = true
			err := mergo.Merge(
				&prev,
				&setup,
				mergo.WithOverride,
				mergo.WithAppendSlice)
			if err != nil {
				return err
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
	return nil
}
