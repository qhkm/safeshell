package main

import (
	"os"

	"github.com/qhkm/safeshell/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
