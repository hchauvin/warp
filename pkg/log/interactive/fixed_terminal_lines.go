// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package interactive

import (
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"regexp"
	"strings"
)

type FixedTerminalLines struct {
	curLineCount int
}

var ansi = regexp.MustCompile(`^\x1b\[[0-9;]*[a-zA-Z]`)

func (fixed *FixedTerminalLines) Replace(lines []string) error {
	screenWidth, screenHeight, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}

	finalLines := make([]string, 0, len(lines))
	for _, line := range lines {
		var curLine strings.Builder
		curLineLength := 0
		for i := 0; i < len(line); {
			c := line[i]
			if c != '\x1b' {
				curLine.WriteByte(c)
				curLineLength++
				i++
			} else {
				match := ansi.FindString(line[i:])
				curLine.WriteString(match)
				i += len(match)
			}
			if curLineLength == screenWidth || i == len(line) {
				finalLines = append(finalLines, curLine.String())
				curLine.Reset()
				curLineLength = 0
			}
		}
	}

	if len(finalLines) > screenHeight && screenHeight > 0 {
		finalLines = append([]string{"..."}, finalLines[len(finalLines)-screenHeight+1:]...)
	}

	var output strings.Builder
	output.WriteString(fixed.clearStr())
	for _, line := range finalLines {
		output.WriteString(line)
		output.WriteRune('\n')
	}

	if _, err := os.Stdout.Write([]byte(output.String())); err != nil {
		return err
	}

	fixed.curLineCount = len(finalLines)

	return nil
}

func (fixed *FixedTerminalLines) clearStr() string {
	var s strings.Builder
	for i := 0; i < fixed.curLineCount; i++ {
		s.WriteString("\x1B[1A\x1B[K")
	}
	return s.String()
}
