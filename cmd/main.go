package main

import (
	"os"

	"chart-viewer/cmd/chartviewer"
)

func main() {
	cmd := chartviewer.NewRootCommand()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
