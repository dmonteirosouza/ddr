package main

import (
	"fmt"
	"os"

	"ddr/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ddr: %v\n", err)
		os.Exit(1)
	}
}
