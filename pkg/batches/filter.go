// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package batches

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/tags"
)

// Filter either removes from a Batch definition all the commands that
// do not pass the tag filter, or focus the Batch on only one command.
func (batch *Batch) Filter(tagFilter string, focus string) (*Batch, error) {
	var commands []BatchCommand
	if focus != "" {
		for _, cmd := range batch.Commands {
			if cmd.Name == focus {
				commands = []BatchCommand{cmd}
				commands[0].DependsOn = nil
				break
			}
		}
		if commands == nil {
			return nil, fmt.Errorf("cannot find command '%s' to focus on", focus)
		}
	} else {
		compiledTagFilter, err := tags.CompileFilter(tagFilter)
		if err != nil {
			return nil, fmt.Errorf("could not compile tag filter: %v", err)
		}

		for _, cmd := range batch.Commands {
			if compiledTagFilter.Apply(cmd.Tags) {
				commands = append(commands, cmd)
			}
		}
	}

	return &Batch{
		Pipelines: batch.Pipelines,
		Commands:  commands,
	}, nil
}
