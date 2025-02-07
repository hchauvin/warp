// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package log implements logging.
package log

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"io"
	"os"
	"os/exec"
)

// Logger parameterizes logging.
type Logger struct {
	// Writer is the writer to write the logs to.
	Writer      io.Writer
	interactive bool
}

var formatPrefix = color.New(color.Bold).SprintFunc()
var formatWarningPrefix = color.New(color.FgMagenta).SprintFunc()
var formatErrorPrefix = color.New(color.FgRed).SprintFunc()

// SetInteractive turns the logger into an interactive logger.
// In interactive mode, the cursor can move in the console terminal.
func (l *Logger) SetInteractive(int bool) {
	l.interactive = int
}

// Info logs an info message for the given log domain.
func (l *Logger) Info(domain string, message string, args ...interface{}) {
	args = append([]interface{}{domain}, args...)
	l.printf(formatPrefix("%s: ")+message+"\n", args...)
}

// Warning logs a warning message for the given log domain.
func (l *Logger) Warning(domain string, message string, args ...interface{}) {
	args = append([]interface{}{domain}, args...)
	l.printf(formatPrefix("%s: ")+formatWarningPrefix("WARNING: ")+message+"\n", args...)
}

// Error logs an error message for the given log domain.
func (l *Logger) Error(domain string, message string, args ...interface{}) {
	args = append([]interface{}{domain}, args...)
	l.printf(formatPrefix("%s: ")+formatErrorPrefix("ERROR: ")+message+"\n", args...)
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
		// This means that Pipe was invoked on a cmd that has either
		// its os.Stdout already set, or has already been started.
		// Here, that is a logic error.
		panic(fmt.Errorf("could not pipe command stdout: %v", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		// This means that Pipe was invoked on a cmd that has either
		// its os.Stdout already set, or has already been started.
		// Here, that is a logic error.
		panic(fmt.Errorf("could not pipe command stderr: %v", err))
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
			l.Info(domain, "%s", string(scanner.Bytes()))
		}
	}()
}

func (l *Logger) printf(message string, args ...interface{}) {
	if l.interactive {
		return
	}

	w := l.Writer
	if w == nil {
		w = os.Stdout
	}
	s := fmt.Sprintf(message, args...)
	if _, err := fmt.Fprint(w, s); err != nil {
		panic(fmt.Sprintf("cannot write log message: %v; message: %s", err, s))
	}
}
