package main

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/testing"
	"os"
)

func main() {
	if err := testing.ExpectBody(os.Getenv("ENDPOINT"), "42\n"); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("SUCCESS!\n")
	}
}
