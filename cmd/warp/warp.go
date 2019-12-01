// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package main

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/urfave/cli"
	"os"
)

var (
	version = "dev"
	commit  = "<none>"
	date    = "<unknown>"
)

func main() {
	app := cli.NewApp()

	app.Version = fmt.Sprintf("%s (commit: %s; date: %s)", version, commit, date)
	app.Name = "warp"
	app.Usage = "Yet another CLI wrapper"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "TOML project-wide config file.  The parent path of the config file is used as the workspace root.  All the file paths are given relative to the workspace root.",
			Value: ".warprc.toml",
		},
		cli.StringFlag{
			Name:  "cwd",
			Usage: "Working directory",
			Value: ".",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:        "hold",
			Usage:       "Holds a stack",
			ArgsUsage:   "<pipeline file>",
			Description: "Holds a stack created from a specific pipeline.  The pipeline file is either directly the YAML specification, or a folder that contains a 'pipeline.yml' file.  The path is given relative to the workspace root (parent folder of the global TOML config).",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "tail",
					Usage: "Shows container logs",
				},
				cli.BoolFlag{
					Name:  "dev",
					Usage: "Executes the dev steps (file synchronization, port forwarding, ...)",
				},
				cli.StringSliceFlag{
					Name:  "run",
					Usage: "Runs programs in the 'run' section, given their spec name",
				},
				cli.BoolFlag{
					Name:  "wait",
					Usage: "Waits for interrupt (Ctl-C) before releasing the stack.  Always true when --run is not given.",
				},
				cli.BoolFlag{
					Name:  "rm",
					Usage: "Removes/cleans up a stack when finished",
				},
			},
			Action: func(c *cli.Context) error {
				return warp.Hold(&warp.HoldConfig{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
					Dev:          c.Bool("dev"),
					Tail:         c.Bool("tail"),
					Run:          c.StringSlice("run"),
					Wait:         c.Bool("wait"),
					Rm:           c.Bool("rm"),
				})
			},
		},
		{
			Name:        "rm",
			Usage:       "Removes/cleans stacks",
			ArgsUsage:   "<pipeline file> [short names...]",
			Description: "Removes the stacks created from a pipeline, either all of them, or a specific list of short names",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all",
					Usage: "Removes all the stacks, even the ones that are currently in use",
				},
			},
			Action: func(c *cli.Context) error {
				return warp.Rm(&warp.RmCfg{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
					ShortNames:   c.Args().Tail(),
					All:          c.Bool("all"),
				})
			},
		},
		{
			Name:      "gc",
			Usage:     "Garbage collect stacks, either from one family or all the families",
			ArgsUsage: "<family>",
			Action: func(c *cli.Context) error {
				return warp.Gc(context.Background(), &warp.GcCfg{
					WorkingDir: c.GlobalString("cwd"),
					ConfigPath: c.GlobalString("config"),
					Family:     c.Args().First(),
				})
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "%v\n", err); err != nil {
			panic(err.Error())
		}
		os.Exit(1)
	}
}
