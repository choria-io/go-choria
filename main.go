package main

//go:generate go run plugin/gen.go
//go:generate go run config/gen.go
//go:generate go run gen_config_doc.go
//go:generate go run client/gen.go

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
