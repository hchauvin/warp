// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package main

import (
	"errors"
	"fmt"
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
	if os.Getenv("ENDPOINT") == "" {
		return errors.New("expected ENDPOINT")
	}

	expectedConfigFoo := "__foo__"
	actualConfigFoo := os.Getenv("CONFIG_FOO")
	if actualConfigFoo != actualConfigFoo {
		return fmt.Errorf("expected CONFIG_FOO to be '%s', got '%s'", expectedConfigFoo, actualConfigFoo)
	}

	expectedSecretBar := "__bar__"
	actualSecretBar := os.Getenv("SECRET_BAR")
	if actualConfigFoo != actualConfigFoo {
		return fmt.Errorf("expected SECRET_BAR to be '%s', got '%s'", expectedSecretBar, actualSecretBar)
	}

	return nil
}
