package main

import (
	"os"

	"github.com/safeshell/safeshell/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
