// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package main

import (
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/telemetry"
	_ "github.com/hchauvin/warp/pkg/telemetry/mongo"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"time"
)

type commandTelemetry struct {
	client     telemetry.Client
	invocation telemetry.CLIInvocation
}

func commandInvoked(c *cli.Context) *commandTelemetry {
	cfg, err := readConfig(c.String("cwd"), c.String("config"))
	if err != nil {
		return &commandTelemetry{}
	}

	if cfg.Telemetry.ConnectionString == "" {
		return &commandTelemetry{}
	}

	client, err := telemetry.NewClient(cfg.Telemetry.ConnectionString)
	if err != nil {
		cfg.Logger().Error("telemetry", "could not create telemetry client: %v", err)
		return &commandTelemetry{}
	}

	version := telemetry.CLIVersion{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	invocation := telemetry.CLIInvocation{
		CLIVersion: version,
		User:       os.Getenv("USER"),
		Started:    time.Now(),
		Args:       os.Args,
	}
	client.Send(invocation)

	return &commandTelemetry{
		client,
		invocation,
	}
}

func (cmdtel *commandTelemetry) completed(err error) {
	if cmdtel.client == nil {
		return
	}
	var errStr *string
	if err != nil {
		errS := err.Error()
		errStr = &errS
	}
	cmdtel.client.Send(telemetry.CLICompletion{
		CLIInvocation: cmdtel.invocation,
		Completed:     time.Now(),
		Err:           errStr,
	})
}

func readConfig(workingDir, configPath string) (*config.Config, error) {
	fullPath := filepath.Join(workingDir, configPath)
	return config.Read(fullPath)
}
