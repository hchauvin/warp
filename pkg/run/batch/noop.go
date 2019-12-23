// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package batch

import (
	"io"
)

// NoopReporter is a no-op reporter.
type NoopReporter struct{}

// EnvironmentSetupResult implements Reporter.
func (reporter *NoopReporter) EnvironmentSetupResult(result *EnvironmentSetupResult) {
}

// CommandOutput implements Reporter.
func (reporter *NoopReporter) CommandOutput(info *CommandInfo) (io.WriteCloser, error) {
	return &discardCloser{}, nil
}

// CommandResult implements Reporter.
func (reporter *NoopReporter) CommandResult(result *CommandResult) {
}

// Finalize implements Reporter.
func (reporter *NoopReporter) Finalize() error {
	return nil
}

type discardCloser struct{}

func (discardCloser) Write(p []byte) (int, error) { return len(p), nil }

func (discardCloser) Close() error { return nil }
