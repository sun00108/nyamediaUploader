package main

import (
	"context"
	"fmt"
	"os"

	"nyamediaUploader/internal/cli"
)

func main() {
	app := cli.New()
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
