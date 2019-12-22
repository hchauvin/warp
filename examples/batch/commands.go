// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package main

import (
	"errors"
	"fmt"
	"github.com/hchauvin/warp/pkg/testing"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	p, err := readParams()
	if err != nil {
		return err
	}

	file := filepath.Join(p.outputDir, p.file)
	if err := ioutil.WriteFile(file, []byte{}, 0666); err != nil {
		return fmt.Errorf("cannot write to file %s: %v", file, err)
	}

	for _, dep := range p.dependsOn {
		depFile := filepath.Join(p.outputDir, dep)
		if _, err := os.Stat(depFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("dependency '%s' was not executed: file '%s' is missing", dep, depFile)
			}
			return fmt.Errorf("cannot stat file %s: %v", depFile, err)
		}
	}

	if p.test != "" {
		switch p.test {
		case "pipeline":
			if err := testing.ExpectBody(os.Getenv("ENDPOINT"), "Hello, world!\n"); err != nil {
				return err
			}
		case "fail":
			return errors.New("__fail__")
		default:
			return fmt.Errorf("unrecognized test '%s'", p.test)
		}
	}

	return nil
}

type params struct {
	outputDir string
	file      string
	dependsOn []string
	test      string
}

func readParams() (*params, error) {
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		return nil, errors.New("OUTPUT_DIR environment variable is mandatory")
	}

	file := os.Getenv("FILE")
	if file == "" {
		return nil, errors.New("FILE environment variable is mandatory")
	}

	dependsOnStr := os.Getenv("DEPENDS_ON")
	var dependsOn []string
	if dependsOnStr != "" {
		dependsOn = strings.Split(dependsOnStr, ",")
	}

	test := os.Getenv("TEST")

	return &params{
		outputDir: outputDir,
		file:      file,
		dependsOn: dependsOn,
		test:      test,
	}, nil
}

func testPipeline(p *params) error {
	return testing.ExpectBody(os.Getenv("ENDPOINT"), "Hello, world!\n")
}
