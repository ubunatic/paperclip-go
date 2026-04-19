// Command paperclip-go is the entry point for the Paperclip Go control plane.
package main

import "github.com/ubunatic/paperclip-go/internal/cli"

// Version is the build version, set via ldflags during build.
var Version = "dev"

func main() {
	cli.Execute(Version)
}
