// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package pipelines

import (
	"errors"
	"github.com/hchauvin/warp/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Expand expands a pipeline definition with its bases it inherits from.
// The result of the expansion is written in a YAML file given in the config.
func (pipeline *Pipeline) Expand(cfg *config.Config) error {
	var expandedPipelineFolder string
	if pipeline.Stack.Name != "" {
		expandedPipelineFolder = pipeline.Stack.Name
	} else if pipeline.Stack.Family != "" {
		expandedPipelineFolder = pipeline.Stack.Family
	} else {
		return errors.New("invalid pipeline: neither stack.name nor stack.family is given")
	}
	expandedPipelineFolder = cfg.Path(filepath.Join(cfg.OutputRoot, "pipelines", expandedPipelineFolder))
	if err := os.MkdirAll(expandedPipelineFolder, 0777); err != nil {
		return err
	}
	pipelineYaml, err := yaml.Marshal(pipeline)
	if err != nil {
		return err
	}
	pipelinePath := filepath.Join(expandedPipelineFolder, "expanded_pipeline.yml")
	if err := ioutil.WriteFile(
		pipelinePath,
		pipelineYaml,
		0777); err != nil {
		return err
	}
	cfg.Logger().Info("pipelines", "pipeline expanded to '%s'", pipelinePath)

	return nil
}
