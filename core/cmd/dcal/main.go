package main

import (
	"os"

	"github.com/AvengeMedia/dankgo/log"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	log.SetEnvPrefix("DANKCAL")
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
}
