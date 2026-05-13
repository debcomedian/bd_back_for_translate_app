package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func collectWantedTranslationKeys(cfg Config, specs []TargetSpec) (map[string]map[string]struct{}, int, error) {
	wanted := map[string]map[string]struct{}{}
	for _, spec := range specs {
		wanted[spec.LangCode] = map[string]struct{}{}
	}
	scanned := 0

	r, closeFn, err := openPossiblyGzipped(cfg.EnglishInputPath)
	if err != nil {
		return nil, 0, err
	}
	defer closeFn()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		scanned++

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.LangCode != "en" {
			continue
		}
		if _, ok := canonicalPOS(entry.Pos); !ok {
			continue
		}
		if !isSimpleEnglishLemma(entry.Word, cfg.AllowMultiwordEN) {
			continue
		}
		if cfg.RequireGloss && len(collectGlosses(entry, cfg.MaxGlosses)) == 0 {
			continue
		}

		for _, spec := range specs {
			for _, tr := range collectTranslationsRelaxed(entry, spec.LangCode, cfg.AllowMultiwordExtended, cfg.MaxTranslationsPerLang) {
				wanted[spec.LangCode][normalizeLexemeKey(tr)] = struct{}{}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, scanned, fmt.Errorf("scan english input for wanted keys: %w", err)
	}
	return wanted, scanned, nil
}

func scanTargetDump(spec TargetSpec, wanted map[string]struct{}, cfg Config) (map[string]*TargetLemmaInfo, int, int, error) {
	result := map[string]*TargetLemmaInfo{}
	scanned := 0

	r, closeFn, err := openPossiblyGzipped(spec.InputPath)
	if err != nil {
		return nil, 0, 0, err
	}
	defer closeFn()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		scanned++

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.LangCode != spec.LangCode {
			continue
		}

		key := normalizeLexemeKey(entry.Word)
		if key == "" {
			continue
		}
		if _, ok := wanted[key]; !ok {
			continue
		}

		info, ok := result[key]
		if !ok {
			info = &TargetLemmaInfo{
				Lemma:             normalizeTranslationWord(entry.Word),
				Key:               key,
				LangCode:          spec.LangCode,
				HasGloss:          false,
				POS:               []string{},
				BridgeTranslations: []string{},
				PeerTranslations:  map[string][]string{},
				bridgeKeys:        map[string]struct{}{},
			}
			result[key] = info
		}

		if len(collectGlosses(entry, 1)) > 0 {
			info.HasGloss = true
		}
		addPos(info, strings.TrimSpace(entry.Pos))

		bridges := collectTranslationsRelaxed(entry, spec.BridgeLang, cfg.AllowMultiwordExtended, cfg.MaxTranslationsPerLang)
		info.BridgeTranslations = mergeUniqueOrdered(info.BridgeTranslations, bridges)

		for _, bridge := range bridges {
			info.bridgeKeys[normalizeLexemeKey(bridge)] = struct{}{}
		}

		for _, peer := range spec.PeerLangs {
			peerVals := collectTranslationsRelaxed(entry, peer, cfg.AllowMultiwordExtended, cfg.MaxTranslationsPerLang)
			info.PeerTranslations[peer] = mergeUniqueOrdered(info.PeerTranslations[peer], peerVals)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, scanned, len(result), fmt.Errorf("scan target dump %s: %w", spec.InputPath, err)
	}

	for _, info := range result {
		sort.Strings(info.POS)
	}
	return result, scanned, len(result), nil
}

func addPos(info *TargetLemmaInfo, pos string) {
	pos = strings.TrimSpace(pos)
	if pos == "" {
		return
	}
	for _, existing := range info.POS {
		if existing == pos {
			return
		}
	}
	info.POS = append(info.POS, pos)
}

func openPossiblyGzipped(path string) (io.Reader, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input file %s: %w", path, err)
	}

	closeFn := func() error { return f.Close() }

	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, nil, fmt.Errorf("open gzip reader %s: %w", path, err)
		}
		closeFn = func() error {
			_ = gz.Close()
			return f.Close()
		}
		return gz, closeFn, nil
	}

	return f, closeFn, nil
}
