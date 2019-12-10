// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import "github.com/hchauvin/warp/pkg/pipelines"

type Batch struct {
	Pipelines []Pipeline     `yaml:"pipelines"`
	Commands  []BatchCommand `yaml:"commands"`
}

type Pipeline struct {
	Name      string `yaml:"name" validate:"required,name"`
	Path      string `yaml:"path" validate:"required"`
	Setup     string `yaml:"setup"`
	TagFilter string `yaml:"tagFilter"`
}

type BatchCommand struct {
	pipelines.BaseCommand `yaml:",inline"`

	Name      string   `yaml:"name" validate:"required,name"`
	Exclusive bool     `yaml:"exclusive"`
	Tags      []string `yaml:"tags" validate:"name"`
	DependsOn []string `yaml:"dependsOn" validate:"name"`
	Pipelines []string `yaml:"pipelines"`
	Flaky     bool     `yaml:"flaky"`
}
