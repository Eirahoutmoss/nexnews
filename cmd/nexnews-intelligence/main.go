// Command nexnews-intelligence imports a NexNews snapshot, persists it in an
// independent Intelligence database, and runs the conservative local analysis.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nexnews/intelligence"
)

const maxSnapshotSize int64 = 64 << 20

type commandReport struct {
	OK           bool                       `json:"ok"`
	SnapshotPath string                     `json:"snapshotPath"`
	DatabasePath string                     `json:"databasePath"`
	Sync         intelligence.SyncStats     `json:"sync"`
	Analysis     intelligence.PipelineStats `json:"analysis"`
	Error        string                     `json:"error,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("nexnews-intelligence", flag.ContinueOnError)
	flags.SetOutput(stderr)
	snapshotPath := flags.String("snapshot", "", "NexNews IntelligenceSnapshot JSON file (required)")
	databasePath := flags.String("db", "nexnews-intelligence.json", "separate Intelligence database file")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: nexnews-intelligence -snapshot <nexnews-snapshot.json> [-db <intelligence.json>]")
		fmt.Fprintln(stderr, "Imports a local NexNews snapshot, then analyzes stored titles/summaries.")
		fmt.Fprintln(stderr, "No network requests are made. Claims remain source-reported; no relations are inferred.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "Unexpected positional arguments.")
		flags.Usage()
		return 2
	}
	if strings.TrimSpace(*snapshotPath) == "" {
		fmt.Fprintln(stderr, "The -snapshot flag is required.")
		flags.Usage()
		return 2
	}
	if strings.TrimSpace(*databasePath) == "" {
		fmt.Fprintln(stderr, "The -db flag cannot be empty.")
		return 2
	}

	inputPath, err := filepath.Abs(*snapshotPath)
	if err != nil {
		fmt.Fprintf(stderr, "Resolve snapshot path: %v\n", err)
		return 2
	}
	dbPath, err := filepath.Abs(*databasePath)
	if err != nil {
		fmt.Fprintf(stderr, "Resolve database path: %v\n", err)
		return 2
	}
	payload, err := readSnapshot(inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "Read snapshot: %v\n", err)
		return 1
	}

	repo, err := intelligence.NewJSONRepository(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Open Intelligence repository: %v\n", err)
		return 1
	}
	report := commandReport{
		SnapshotPath: inputPath,
		DatabasePath: dbPath,
	}
	report.Sync, err = repo.SyncSnapshotJSON(payload)
	if err != nil {
		report.Error = fmt.Sprintf("Import snapshot: %v", err)
		_ = writeReport(stdout, report)
		fmt.Fprintln(stderr, report.Error)
		return 1
	}
	report.Analysis, err = repo.AnalyzeStoredContent()
	if err != nil {
		report.Error = fmt.Sprintf("Analyze stored content: %v", err)
		_ = writeReport(stdout, report)
		fmt.Fprintln(stderr, report.Error)
		return 1
	}
	report.OK = true
	if err := writeReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "Write result: %v\n", err)
		return 1
	}
	return 0
}

func readSnapshot(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxSnapshotSize {
		return nil, fmt.Errorf("snapshot exceeds %d-byte limit", maxSnapshotSize)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSnapshotSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSnapshotSize {
		return nil, fmt.Errorf("snapshot exceeds %d-byte limit", maxSnapshotSize)
	}
	if len(data) == 0 {
		return nil, errors.New("snapshot file is empty")
	}
	if !json.Valid(data) {
		return nil, errors.New("snapshot file is not valid JSON")
	}
	return data, nil
}

func writeReport(dst io.Writer, report commandReport) error {
	enc := json.NewEncoder(dst)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
