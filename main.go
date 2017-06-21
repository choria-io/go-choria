package main

import (
	"os"

	"github.com/choria-io/go-choria/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	var err error

	err = cmd.ParseCLI()
	if err != nil {
		log.Fatalf("Could not configure Choria: %s", err.Error())
		os.Exit(1)
	}

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not run Choria: %s", err.Error())
		os.Exit(1)
	}
}
