package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stevensalex299/RoboOrders/backend/internal/db"
	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "webhook":
		os.Exit(runWebhook(os.Args[2:]))
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
  ingest help

Webhook options:
  --file PATH       jsonl fixture (required)
  --from N          first line, 1-based (default 1)
  --to N            last line inclusive (default: end of file)
  --limit N         max lines after --from (stops before --to if set)
  --delay MS        milliseconds between each line (default 0)
  --http URL        POST each line to URL/webhooks/orders (default: upsert via DB)
  -q                summary only

Examples:
  go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 5
  go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --from 1 --to 100
  go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --http http://localhost:8080 --limit 50
`)
}

type webhookOpts struct {
	file     string
	from     int
	to       int
	limit    int
	delayMS  int
	httpBase string
	quiet    bool
}

func runWebhook(args []string) int {
	fs := flag.NewFlagSet("webhook", flag.ExitOnError)
	opts := webhookOpts{}
	fs.StringVar(&opts.file, "file", "", "path to webhook jsonl")
	fs.IntVar(&opts.from, "from", 1, "first line (1-based)")
	fs.IntVar(&opts.to, "to", 0, "last line inclusive (0 = EOF)")
	fs.IntVar(&opts.limit, "limit", 0, "max lines after from")
	fs.IntVar(&opts.delayMS, "delay", 0, "ms between lines")
	fs.StringVar(&opts.httpBase, "http", "", "POST to this API base URL")
	fs.BoolVar(&opts.quiet, "q", false, "quiet")
	_ = fs.Parse(args)

	if opts.file == "" {
		fmt.Fprintln(os.Stderr, "webhook: --file is required")
		return 1
	}
	if opts.from < 1 {
		fmt.Fprintln(os.Stderr, "webhook: --from must be >= 1")
		return 1
	}

	jobs, err := readWebhookJobs(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "webhook: %v\n", err)
		return 1
	}
	if len(jobs) == 0 {
		fmt.Fprintln(os.Stderr, "webhook: no lines to ingest")
		return 1
	}

	var st *store.Store
	if opts.httpBase == "" {
		sqlDB, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "webhook: db: %v\n", err)
			return 1
		}
		defer sqlDB.Close()
		st = store.New(sqlDB)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	if opts.httpBase != "" {
		opts.httpBase = strings.TrimRight(opts.httpBase, "/")
	}

	start := time.Now()
	var okCount, errCount int

	for _, job := range jobs {
		if opts.delayMS > 0 {
			time.Sleep(time.Duration(opts.delayMS) * time.Millisecond)
		}

		var p store.WebhookPayload
		if err := json.Unmarshal([]byte(job.body), &p); err != nil {
			errCount++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "line %d: json: %v\n", job.lineNum, err)
			}
			continue
		}

		ctx := context.Background()
		if opts.httpBase != "" {
			if err := postWebhook(ctx, client, opts.httpBase, job.body); err != nil {
				errCount++
				if !opts.quiet {
					fmt.Fprintf(os.Stderr, "line %d: http: %v\n", job.lineNum, err)
				}
				continue
			}
		} else if _, err := st.UpsertWebhook(ctx, p); err != nil {
			errCount++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "line %d: upsert: %v\n", job.lineNum, err)
			}
			continue
		}
		okCount++
	}

	elapsed := time.Since(start)
	mode := "db"
	if opts.httpBase != "" {
		mode = "http"
	}
	rps := float64(0)
	if sec := elapsed.Seconds(); sec > 0 {
		rps = float64(okCount) / sec
	}
	fmt.Printf("webhook ingest: %d/%d ok", okCount, okCount+errCount)
	if errCount > 0 {
		fmt.Printf(" (%d failed)", errCount)
	}
	fmt.Printf(" in %s (~%.0f req/s) mode=%s\n", elapsed.Round(time.Millisecond), rps, mode)

	if errCount > 0 {
		return 1
	}
	return 0
}

type webhookJob struct {
	lineNum int
	body    string
}

func readWebhookJobs(opts webhookOpts) ([]webhookJob, error) {
	f, err := os.Open(opts.file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var jobs []webhookJob
	lineNum := 0
	for sc.Scan() {
		lineNum++
		if lineNum < opts.from {
			continue
		}
		if opts.to > 0 && lineNum > opts.to {
			break
		}
		body := strings.TrimSpace(sc.Text())
		if body == "" {
			continue
		}
		jobs = append(jobs, webhookJob{lineNum: lineNum, body: body})
		if opts.limit > 0 && len(jobs) >= opts.limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func postWebhook(ctx context.Context, client *http.Client, base, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/webhooks/orders", bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
