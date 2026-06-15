package main

import (
	"os"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
}
