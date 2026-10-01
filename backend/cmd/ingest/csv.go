package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stevensalex299/RoboOrders/backend/internal/db"
	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

type csvFileList []string

func (c *csvFileList) String() string { return strings.Join(*c, ",") }
func (c *csvFileList) Set(v string) error {
	*c = append(*c, v)
	return nil
}

type csvOpts struct {
	files   csvFileList
	from    int
	to      int
	limit   int
	delayMS int
	quiet   bool
}

func runCSV(args []string) int {
	fs := flag.NewFlagSet("csv", flag.ExitOnError)
	opts := csvOpts{}
	fs.Var(&opts.files, "file", "path to CSV (repeatable)")
	fs.IntVar(&opts.from, "from", 1, "first data row (1-based, after header)")
	fs.IntVar(&opts.to, "to", 0, "last row inclusive (0 = EOF)")
	fs.IntVar(&opts.limit, "limit", 0, "max rows after from")
	fs.IntVar(&opts.delayMS, "delay", 0, "ms between rows")
	fs.BoolVar(&opts.quiet, "q", false, "quiet")
	_ = fs.Parse(args)

	if len(opts.files) == 0 {
		fmt.Fprintln(os.Stderr, "csv: at least one --file is required")
		return 1
	}
	if opts.from < 1 {
		fmt.Fprintln(os.Stderr, "csv: --from must be >= 1")
		return 1
	}

	sqlDB, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "csv: db: %v\n", err)
		return 1
	}
	defer sqlDB.Close()
	st := store.New(sqlDB)

	start := time.Now()
	var okCount, errCount int

	for _, path := range opts.files {
		n, fail, err := ingestCSVFile(st, path, opts)
		okCount += n
		errCount += fail
		if err != nil {
			fmt.Fprintf(os.Stderr, "csv: %s: %v\n", path, err)
			return 1
		}
	}

	printIngestSummary("csv", okCount, errCount, start, "db")
	if errCount > 0 {
		return 1
	}
	return 0
}

func ingestCSVFile(st *store.Store, path string, opts csvOpts) (ok, fail int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return 0, 0, err
	}
	if len(records) < 2 {
		return 0, 0, nil
	}

	dataRows := records[1:]
	ctx := context.Background()

	for i, rec := range dataRows {
		rowNum := i + 1
		if rowNum < opts.from {
			continue
		}
		if opts.to > 0 && rowNum > opts.to {
			break
		}
		if opts.limit > 0 && ok+fail >= opts.limit {
			break
		}

		if opts.delayMS > 0 && (ok+fail) > 0 {
			time.Sleep(time.Duration(opts.delayMS) * time.Millisecond)
		}

		row, perr := parseCSVRecord(rec)
		if perr != nil {
			fail++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "%s row %d: %v\n", path, rowNum, perr)
			}
			continue
		}

		if _, err := st.UpsertCSVRow(ctx, row); err != nil {
			fail++
			if !opts.quiet {
				fmt.Fprintf(os.Stderr, "%s row %d: upsert: %v\n", path, rowNum, err)
			}
			continue
		}
		ok++
	}
	return ok, fail, nil
}

func parseCSVRecord(rec []string) (store.CSVRow, error) {
	for len(rec) < 6 {
		rec = append(rec, "")
	}
	tomorrow := false
	if s := strings.TrimSpace(rec[4]); s != "" {
		var err error
		tomorrow, err = strconv.ParseBool(s)
		if err != nil {
			return store.CSVRow{}, fmt.Errorf("tomorrow: %w", err)
		}
	}
	return store.CSVRow{
		FirstName: strings.TrimSpace(rec[0]),
		LastName:  strings.TrimSpace(rec[1]),
		Items:     store.ParseCSVItemsField(rec[2]),
		Notes:     strings.TrimSpace(rec[3]),
		Tomorrow:  tomorrow,
		Meal:      strings.TrimSpace(rec[5]),
	}, nil
}
