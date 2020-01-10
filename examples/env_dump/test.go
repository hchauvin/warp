// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package main

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("SUCCESS!\n")
	}
}

func run() error {
	envDumpPath := os.Getenv("ENV_DUMP_PATH")

	if envDumpPath == "" {
		return errors.New("expected ENV_DUMP_PATH")
	}

	env, err := godotenv.Read(envDumpPath)
	if err != nil {
		return err
	}

	if env["FOO"] != "bar" {
		b, err := ioutil.ReadFile(envDumpPath)
		if err != nil {
			panic(err.Error())
		}
		return fmt.Errorf("expected env file to contain FOO=bar; content: <<< %s >>>", b)
	}

	return nil
}
