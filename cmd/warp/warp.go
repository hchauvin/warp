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
			Name:        "lint",
			Usage:       "Lints a stack",
			ArgsUsage:   "<pipeline file>",
			Description: "Lints a stack.",
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Lint(context.Background(), &warp.LintCfg{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
				})
				return
			},
		},
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
					Usage: "Runs programs in the 'commands' section, given their spec name",
				},
				cli.StringFlag{
					Name:  "setup",
					Usage: "Sets up",
				},
				cli.StringFlag{
					Name:  "dump_env",
					Usage: "Dumps env",
				},
				cli.BoolFlag{
					Name:  "persist_env",
					Usage: "Persists the dumped env variables after warp exits",
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
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Hold(&warp.HoldConfig{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
					Dev:          c.Bool("dev"),
					Tail:         c.Bool("tail"),
					Run:          c.StringSlice("run"),
					Setup:        c.String("setup"),
					DumpEnv:      c.String("dump_env"),
					PersistEnv:   c.Bool("persist_env"),
					Wait:         c.Bool("wait"),
					Rm:           c.Bool("rm"),
				})
				return
			},
		},
		{
			Name:        "deploy",
			Usage:       "Deploys a stack",
			ArgsUsage:   "<pipeline file>",
			Description: "Deploys a stack created from a specific pipeline.",
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Deploy(context.Background(), &warp.DeployCfg{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
				})
				return
			},
		},
		{
			Name:      "batch",
			Usage:     "Executes a batch of commands",
			ArgsUsage: "<batch file>",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "parallelism",
					Usage: "Parallelism",
					Value: 1,
				},
				cli.IntFlag{
					Name:  "max_stacks_per_pipeline",
					Usage: "Max stacks per pipeline",
					Value: 1,
				},
				cli.StringFlag{
					Name:  "tags",
					Usage: "Test tag filter",
				},
				cli.StringFlag{
					Name:  "focus",
					Usage: "Test focus",
				},
				cli.BoolFlag{
					Name:  "bail",
					Usage: "Bail out on first error",
				},
				cli.BoolFlag{
					Name:  "advisory",
					Usage: "Do not fail on error: the result (pass/fail) is advisory only.  If --bail is enabled, --advisory is ignored.",
				},
				cli.StringFlag{
					Name:  "report",
					Usage: "Output path to report folder",
				},
				cli.BoolFlag{
					Name:  "stream",
					Usage: "Stream results instead of being in interactive mode",
				},
			},
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Batch(context.Background(), &warp.BatchCfg{
					WorkingDir:           c.GlobalString("cwd"),
					ConfigPath:           c.GlobalString("config"),
					BatchPath:            c.Args().First(),
					Parallelism:          c.Int("parallelism"),
					MaxStacksPerPipeline: c.Int("max_stacks_per_pipeline"),
					Tags:                 c.String("tags"),
					Focus:                c.String("focus"),
					Bail:                 c.Bool("bail"),
					Advisory:             c.Bool("advisory"),
					Report:               c.String("report"),
					Stream:               c.Bool("stream"),
				})
				return err
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
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Rm(&warp.RmCfg{
					WorkingDir:   c.GlobalString("cwd"),
					ConfigPath:   c.GlobalString("config"),
					PipelinePath: c.Args().First(),
					ShortNames:   c.Args().Tail(),
					All:          c.Bool("all"),
				})
				return
			},
		},
		{
			Name:      "gc",
			Usage:     "Garbage collect stacks, either from one family or all the families",
			ArgsUsage: "[family]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "preserve_pvc",
					Usage: "Preserve persistent volume claims.  Overrides the default PVC behavior.",
				},
				cli.BoolFlag{
					Name:  "discard_pvc",
					Usage: "Discard persistent volume claims as well.  Overrides the default PVC behavior.",
				},
			},
			Action: func(c *cli.Context) (err error) {
				t := commandInvoked(c)
				defer t.completed(err)
				err = warp.Gc(context.Background(), &warp.GcCfg{
					WorkingDir:                     c.GlobalString("cwd"),
					ConfigPath:                     c.GlobalString("config"),
					Family:                         c.Args().First(),
					PreservePersistentVolumeClaims: c.Bool("preserve_pvc"),
					DiscardPersistentVolumeClaims:  c.Bool("discard_pvc"),
				})
				return
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
