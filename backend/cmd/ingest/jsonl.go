package main

import (
	"bufio"
	"os"
	"strings"
)

type jsonlJob struct {
	lineNum int
	body    string
}

func readJSONLJobs(file string, from, to, limit int) ([]jsonlJob, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var jobs []jsonlJob
	lineNum := 0
	for sc.Scan() {
		lineNum++
		if lineNum < from {
			continue
		}
		if to > 0 && lineNum > to {
			break
		}
		body := strings.TrimSpace(sc.Text())
		if body == "" {
			continue
		}
		jobs = append(jobs, jsonlJob{lineNum: lineNum, body: body})
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}
