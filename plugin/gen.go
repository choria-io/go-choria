// +build ignore

package main

import (
	"os"

	"github.com/choria-io/go-choria/plugin"
)

func main() {
	if !plugin.Generate() {
		os.Exit(1)
	}
}
