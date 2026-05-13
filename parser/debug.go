package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type debugWriters struct {
	enabled bool
	limit   int
	counts  map[string]int
	writers map[string]*bufio.Writer
	files   []*os.File
}

func newDebugWriters(cfg Config) (*debugWriters, error) {
	dw := &debugWriters{
		enabled: false,
		limit:   cfg.DebugSampleLimitPerReason,
		counts:  map[string]int{},
		writers: map[string]*bufio.Writer{},
		files:   []*os.File{},
	}

	if strings.TrimSpace(cfg.DebugDir) == "" || cfg.DebugSampleLimitPerReason == 0 {
		return dw, nil
	}

	if err := os.MkdirAll(cfg.DebugDir, 0o755); err != nil {
		return nil, fmt.Errorf("create debug dir: %w", err)
	}

	kinds := []string{
		"skipped_by_pos",
		"skipped_by_lemma",
		"skipped_by_gloss",
		"skipped_by_no_candidate_translations",
		"candidate_only",
		"half_validated",
	}

	for _, kind := range kinds {
		path := filepath.Join(cfg.DebugDir, kind+".jsonl")
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("create debug file %s: %w", path, err)
		}
		dw.files = append(dw.files, f)
		dw.writers[kind] = bufio.NewWriterSize(f, 256*1024)
	}

	dw.enabled = true
	return dw, nil
}

func (dw *debugWriters) Close() error {
	var firstErr error
	for _, w := range dw.writers {
		if err := w.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, f := range dw.files {
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (dw *debugWriters) Write(kind string, entry DebugSkippedEntry) error {
	if !dw.enabled {
		return nil
	}
	if dw.limit > 0 && dw.counts[kind] >= dw.limit {
		return nil
	}
	w, ok := dw.writers[kind]
	if !ok {
		return nil
	}
	b, err := marshalLine(entry)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	dw.counts[kind]++
	return nil
}
