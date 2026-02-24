package main

import (
	"github.com/ppiankov/spectrehub/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
