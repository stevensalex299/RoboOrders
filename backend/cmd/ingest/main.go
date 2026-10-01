package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "webhook":
		os.Exit(runWebhook(os.Args[2:]))
	case "poll":
		os.Exit(runPoll(os.Args[2:]))
	case "csv":
		os.Exit(runCSV(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ingest webhook --file PATH [options]
  ingest poll    --file PATH [options]
  ingest csv     --file PATH [--file PATH ...] [options]
  ingest help

Common options (webhook, poll, csv):
  --from N          first record (webhook/csv: 1-based default 1; poll: 0 = use cursor)
  --to N            last record inclusive (0 = end)
  --limit N         max records (poll default 1)
  --delay MS        pause between records
  -q                summary only

Webhook:
  --http URL        POST to URL/webhooks/orders (default: upsert via DB)

Poll:
  --reset-cursor    set poll cursor to 0 before run

Examples:
  go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 5
  go run ./cmd/ingest poll    --file ../fixtures/api_responses.jsonl --limit 5
  go run ./cmd/ingest poll    --file ../fixtures/api_responses.jsonl --from 21 --limit 1
  go run ./cmd/ingest csv     --file ../fixtures/orders_1.csv --file ../fixtures/orders_2.csv
`)
}
