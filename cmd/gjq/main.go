package main

import (
	"fmt"
	"os"

	"github.com/0xkowalskidev/gjq/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
