// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
	if actualConfigFoo != expectedConfigFoo {
		return fmt.Errorf("expected CONFIG_FOO to be '%s', got '%s'", expectedConfigFoo, actualConfigFoo)
	}

	expectedSecretBar := "__bar__"
	actualSecretBar := os.Getenv("SECRET_BAR")
	if actualSecretBar != expectedSecretBar {
		return fmt.Errorf("expected SECRET_BAR to be '%s', got '%s'", expectedSecretBar, actualSecretBar)
	}

	expectedServiceNameSuffix := "echo"
	actualServiceName := os.Getenv("SERVICE_NAME")
	if !strings.HasSuffix(actualServiceName, expectedServiceNameSuffix) {
		return fmt.Errorf(
			"expected SERVICE_NAME to end in '%s', got '%s'",
			expectedServiceNameSuffix,
			actualServiceName)
	}

	return nil
}
