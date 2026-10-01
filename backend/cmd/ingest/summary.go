package main

import (
	"fmt"
	"time"
)

func ingestMode(http bool) string {
	if http {
		return "http"
	}
	return "db"
}

func printIngestSummary(kind string, okCount, errCount int, start time.Time, mode string) {
	elapsed := time.Since(start)
	rps := float64(0)
	if sec := elapsed.Seconds(); sec > 0 {
		rps = float64(okCount) / sec
	}
	fmt.Printf("%s ingest: %d/%d ok", kind, okCount, okCount+errCount)
	if errCount > 0 {
		fmt.Printf(" (%d failed)", errCount)
	}
	fmt.Printf(" in %s (~%.0f req/s) mode=%s\n", elapsed.Round(time.Millisecond), rps, mode)
}
