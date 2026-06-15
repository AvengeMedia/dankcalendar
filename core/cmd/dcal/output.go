package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var jsonOutput bool

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// infof prints progress/instructions; in --json mode they move to stderr so
// stdout carries only the JSON result.
func infof(format string, args ...any) {
	w := os.Stdout
	if jsonOutput {
		w = os.Stderr
	}
	fmt.Fprintf(w, format+"\n", args...)
}
