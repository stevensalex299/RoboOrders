package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stevensalex299/RoboOrders/backend/internal/db"
	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

type pollOpts struct {
	file        string
	from        int
	to          int
	limit       int
	delayMS     int
	resetCursor bool
	quiet       bool
}

func runPoll(args []string) int {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)
	opts := pollOpts{}
	fs.StringVar(&opts.file, "file", "", "path to api_responses jsonl")
	fs.IntVar(&opts.from, "from", 0, "first line (1-based; 0 = continue from stored cursor)")
	fs.IntVar(&opts.to, "to", 0, "last line inclusive (0 = EOF)")
	fs.IntVar(&opts.limit, "limit", 0, "max batches (default 1)")
	fs.IntVar(&opts.delayMS, "delay", 0, "ms between batches")
	fs.BoolVar(&opts.resetCursor, "reset-cursor", false, "set cursor to 0 before run")
	fs.BoolVar(&opts.quiet, "q", false, "quiet")
	_ = fs.Parse(args)

	if opts.file == "" {
		fmt.Fprintln(os.Stderr, "poll: --file is required")
		return 1
	}
	if opts.limit == 0 {
		opts.limit = 1
	}

	sqlDB, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "poll: db: %v\n", err)
		return 1
	}
	defer sqlDB.Close()
	st := store.New(sqlDB)

	cursorKey, err := pollCursorKey(opts.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poll: %v\n", err)
		return 1
	}

	ctx := context.Background()
	if opts.resetCursor {
		if err := st.SetIngestInt(ctx, cursorKey, 0); err != nil {
			fmt.Fprintf(os.Stderr, "poll: reset cursor: %v\n", err)
			return 1
		}
	}

	cursor, err := st.GetIngestInt(ctx, cursorKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poll: cursor: %v\n", err)
		return 1
	}

	startLine := opts.from
	if startLine <= 0 {
		startLine = cursor + 1
	}
	if startLine < 1 {
		startLine = 1
	}

	jobs, err := readJSONLJobs(opts.file, startLine, opts.to, opts.limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poll: %v\n", err)
		return 1
	}
	if len(jobs) == 0 {
		fmt.Fprintln(os.Stderr, "poll: no lines to ingest")
		return 1
	}

	start := time.Now()
	var okCount, errCount int

	for _, job := range jobs {
		if opts.delayMS > 0 {
			time.Sleep(time.Duration(opts.delayMS) * time.Millisecond)
		}

		batch, err := store.ParsePollBatchJSON([]byte(job.body))
		if err != nil {
			errCount++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "line %d: json: %v\n", job.lineNum, err)
			}
			continue
		}
		if batch.Error != "" && !opts.quiet {
			fmt.Fprintf(os.Stderr, "line %d: api error: %s\n", job.lineNum, batch.Error)
		}

		result, err := st.ApplyPollBatch(ctx, batch)
		if err != nil {
			errCount++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "line %d: apply: %v\n", job.lineNum, err)
			}
			continue
		}

		if result.AdvanceCursor {
			if err := st.SetIngestInt(ctx, cursorKey, job.lineNum); err != nil {
				errCount++
				if !opts.quiet {
					fmt.Fprintf(os.Stderr, "line %d: cursor: %v\n", job.lineNum, err)
				}
				continue
			}
		} else if !opts.quiet {
			fmt.Fprintf(os.Stderr, "line %d: applied partial batch; cursor unchanged\n", job.lineNum)
		}
		okCount++
	}

	printIngestSummary("poll", okCount, errCount, start, "db")
	if errCount > 0 {
		return 1
	}
	return 0
}

func pollCursorKey(file string) (string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	return "poll:" + abs, nil
}
