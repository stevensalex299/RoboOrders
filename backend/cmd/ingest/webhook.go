package main

import (
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

	jobs, err := readJSONLJobs(opts.file, opts.from, opts.to, opts.limit)
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

	printIngestSummary("webhook", okCount, errCount, start, ingestMode(opts.httpBase != ""))
	if errCount > 0 {
		return 1
	}
	return 0
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
