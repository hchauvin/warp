// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package batches defines the content of a Batch definition file.
//
// It includes functions to read and validate these files.
package batches

import "github.com/hchauvin/warp/pkg/pipelines"

// Batch is the type for a deserialized Batch definition file.
type Batch struct {
	// Pipelines are the pipelines that the commands can use
	// as prerequisites.
	Pipelines []Pipeline     `yaml:"pipelines"`

	// Commands is a slice of commands to execute in batch mode.
	// The commands are executed concurrently (with a parallelism
	// set elsewhere).  They can depend on each other, giving rise
	// to an acyclic dependency graph.
	Commands  []BatchCommand `yaml:"commands"`
}

// Pipeline defines a pipeline that the commands can use as
// a prerequisite.  Pipelines are used to set up and tear down
// Kubernetes resources, and external resources in general.
type Pipeline struct {
	// Name is the name of the pipeline.  It is used to
	// reference the pipeline in the batch commands.
	Name  string `yaml:"name" validate:"required,name"`

	// Path is the path to the pipeline folder or pipeline file.
	// In the case of a folder "<folder>", the pipeline file
	// is sought at "<folder>/pipeline.yml".  The path is given
	// relative to the workspace dir.
	Path  string `yaml:"path" validate:"required"`

	// Setup is the name of the setup to use.  Setups are defined
	// in the pipeline config.
	Setup string `yaml:"setup"`
}

// BatchCommand is a command to execute in batch mode.
type BatchCommand struct {
	pipelines.BaseCommand `yaml:",inline"`

	// Name is the name of the batch commands.  Names are used
	// for filtering and reporting purposes.
	Name      string   `yaml:"name" validate:"required,name"`

	// Tags is a slice of tags.  Tags are used for filtering.
	Tags      []string `yaml:"tags" validate:"name"`

	// Exclusive is true if the command needs an exclusive hold
	// on the stacks created from its pipelines.  In the case
	// of an exclusive hold, not two commands can execute on the
	// same pipeline.
	//
	// Typically, read-only tests are non-exclusive, but read-write
	// tests might be exclusive.  As an example, one can have a
	// read-write test that inserts an entry in a database.  In this
	// case, the insert can wreck havoc on read-only tests that
	// assume the entry is not present.
	Exclusive bool     `yaml:"exclusive"`

	// DependsOn is a slice of batch commands, referred to by name,
	// that this batch command depends on.  This batch command
	// is only executed when all the dependencies complete successfully.
	DependsOn []string `yaml:"dependsOn" validate:"name"`

	// Pipelines is a slice of pipelines, referred to by name,
	// that this batch command depends on.  This batch command
	// is only executed when all the pipelines' "before hooks"
	// complete successfully.  Moreover, the command executes with the
	// environment variables that are set by the setup for the pipeline.
	Pipelines []string `yaml:"pipelines"`

	// Flaky should be set to true if the test is flaky, that is, if
	// it fails intermittently.  Flaky tests are retried twice after they
	// error.  Flakiness should be avoided by redesigning the test.
	Flaky     bool     `yaml:"flaky"`
}
