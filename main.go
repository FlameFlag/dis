package main

import (
	"context"
	"dis/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
