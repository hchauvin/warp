// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

import (
	"fmt"
)

type terminalReporter struct {
	fixed fixedTerminalLines
}

func (r *terminalReporter) replace(lines []string) error {
	return r.fixed.replace(lines)
}

func (r *terminalReporter) summarize(s summary) error {
	fmt.Printf("----------------------------\n")
	fmt.Printf("Total duration: %s\n", s.totalDuration)
	return nil
}
