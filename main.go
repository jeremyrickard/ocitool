package main

import (
	"github.com/jeremyrickard/ocitool/cmd/ocitool"
	log "github.com/sirupsen/logrus"
)

func main() {
	cmd := ocitool.New()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("error running ocimerge: %s", err)
	}
}
