package main

import (
	"os"
	"time"

	"github.com/choria-io/go-choria/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := cmd.ParseCLI()
	if err != nil {
		log.Fatalf("Could not configure Choria: %s", err.Error())
		os.Exit(1)
	}

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not run Choria: %s", err.Error())
		os.Exit(1)
	}

	for {
		time.Sleep(60 * time.Second)
	}
}
