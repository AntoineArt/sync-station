package main

import (
	"os"

	"github.com/AntoineArt/syncstation/cmd/syncstation"
)

func main() {
	// Set the program name for help text
	os.Args[0] = "syncstation"

	// Execute the main command
	syncstation.Execute()
}
