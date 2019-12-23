// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package batch

import (
	"io"
)

type NoopReporter struct{}

func (reporter *NoopReporter) EnvironmentSetupResult(result *EnvironmentSetupResult) {
}

func (reporter *NoopReporter) CommandOutput(info *CommandInfo) (io.WriteCloser, error) {
	return &discardCloser{}, nil
}

func (reporter *NoopReporter) CommandResult(result *CommandResult) {
}

func (reporter *NoopReporter) Finalize() error {
	return nil
}

type discardCloser struct{}

func (discardCloser) Write(p []byte) (int, error) { return len(p), nil }

func (discardCloser) Close() error { return nil }
