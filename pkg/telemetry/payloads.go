// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package telemetry

import "time"

type CLIVersion struct {
	Version string `bson:"version"`
	Commit  string `bson:"commit"`
	Date    string `bson:"date"`
}

type CLIInvocation struct {
	CLIVersion `bson:",inline"`
	User       string    `bson:"user"`
	Started    time.Time `bson:"started"`
	Args       []string  `bson:"args"`
}

type CLICompletion struct {
	CLIInvocation `bson:",inline"`
	Completed     time.Time `bson:"completed"`
	Err           *string   `bson:"err, omitempty"`
}
