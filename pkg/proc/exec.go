// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// proc extends the standard exec package to deal with processes
// in a more graceful way.
package proc

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GracefulCommandContext has the same purpose as exec.CommandContext,
// but the cancellation is done gracefully, and the process descendants
// are killed as well.
func GracefulCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	go func() {
		<-ctx.Done()

		// cmd.Process is not nil when the process has been started.
		// cmd.ProcessState is nil when the process has not yet exited.
		if cmd.Process != nil && cmd.ProcessState == nil {
			pid := cmd.Process.Pid
			// We launch a go routine to forcefully kill the process if
			// it couldn't be killed gracefully.
			done := make(chan struct{}, 1)
			go func() {
				select {
				case <-done:
					// The process was gracefully killed
				case <-time.After(3 * time.Second):
					// After a timeout, we forcefully kill the process
					if cmd.ProcessState == nil {
						Kill(pid, true)
					}
				}
			}()
			Kill(pid, true)
			done <- struct{}{}
		}
	}()

	return cmd
}

// Kill kills a process by pid.
//
// Be careful: exec.Process.Kill() (which eventually is what is called by
// exec.CommandContext) does not gracefully kill the process, and does not
// kill the child processes.  Killing child processes can be done programmatically,
// but it's an involved task (see, e.g.,
// https://groups.google.com/forum/#!topic/golang-nuts/nayHpf8dVxI).
//
// Therefore, we call the native executables to do the heavy lifting for us.
func Kill(pid int, force bool) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		args := []string{
			"taskkill",
			"/pid",
			strconv.Itoa(pid),
			"/t", // Terminates the specified process and any child processes started by it.
		}
		if force {
			args = append(args, "/f")
		}
		cmd = exec.Command("powershell", args...)
	} else {
		args := []string{"kill"}
		if force {
			args = append(args, "-9")
		}
		args = append(args, strconv.Itoa(pid))
		// We can expect sh to be present everywhere, contrary to bash
		cmd = exec.Command("sh", "-c", strings.Join(args, " "))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if runtime.GOOS == "windows" &&
			!force &&
			strings.Contains(string(out), "This process can only be terminated forcefully") {
			return Kill(pid, true)
		}
		fmt.Printf("ERROR: could not kill process with PID %d: %v - %s\n", pid, err, out)
		return fmt.Errorf("could not kill process with PID %d: %v - %s", pid, err, out)
	}
	return nil
}
