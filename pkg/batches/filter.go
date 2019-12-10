package batches

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/tags"
)

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
