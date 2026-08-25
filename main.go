package main

import (
	"context"
	"os"

	"github.com/4evy/dis/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.New(version).Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
