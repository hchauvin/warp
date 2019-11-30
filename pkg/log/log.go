// log implements logging.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package log

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"io"
	"os"
	"os/exec"
)

type Logger struct{}

const logDomain = "log"

var formatPrefix = color.New(color.Bold).SprintFunc()
var formatWarningPrefix = color.New(color.FgMagenta).SprintFunc()
var formatErrorPrefix = color.New(color.FgRed).SprintFunc()

// Info logs an info message for the given log domain.
func (l *Logger) Info(domain string, message string, args ...interface{}) {
	fmt.Printf(formatPrefix(domain+": ")+message+"\n", args...)
}

// Info logs a warning message for the given log domain.
func (l *Logger) Warning(domain string, message string, args ...interface{}) {
	fmt.Printf(formatPrefix(domain+":")+formatWarningPrefix("WARNING: ")+message+"\n", args...)
}

// Info logs an error message for the given log domain.
func (l *Logger) Error(domain string, message string, args ...interface{}) {
	fmt.Printf(formatPrefix(domain+":")+formatErrorPrefix("ERROR: ")+message+"\n", args...)
}

// Pipe logs the stdin and stdout of a process.  The combined output
// is split by line, and each line is logged as an info message for
// the given log domain.
//
// In the context of a CLI, this prefixes every line of output with the
// log domain, and helps readability when the outputs are multiplexed.
func (l *Logger) Pipe(domain string, cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		l.Error(logDomain, "could not pipe command stdout")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		l.Error(logDomain, "could not pipe command stderr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return
	}

	l.PipeReader(domain, stdout)
	l.PipeReader(domain, stderr)
}

// PipeReader logs the lines scanned from a reader.  Each line
// is logged as an info message for the given log domain.
//
// In the context of a CLI, this prefixes every line of output with the
// log domain, and helps readability when the outputs are multiplexed.
func (l *Logger) PipeReader(domain string, r io.Reader) {
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			l.Info(domain, string(scanner.Bytes()))
		}
	}()
}
