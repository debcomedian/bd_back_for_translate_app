package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func RunStage31(cfg Config) (Stage31Stats, error) {
	stats := Stage31Stats{
		EnglishInputPath: cfg.EnglishInputPath,
		TargetInputPaths: map[string]string{},
		WantedCounts:     map[string]int{},
		TargetScanned:    map[string]int{},
		TargetMatched:    map[string]int{},
		StatusCounts:     map[string]int{},
	}
	specs := cfg.TargetSpecs()
	for _, spec := range specs {
		stats.TargetInputPaths[spec.LangCode] = spec.InputPath
	}

	for _, path := range []string{cfg.CoreOutputPath, cfg.ExtendedOutputPath, cfg.StatsPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return stats, fmt.Errorf("create output dir for %s: %w", path, err)
		}
	}

	wantedSets, englishScanned, err := collectWantedTranslationKeys(cfg, specs)
	if err != nil {
		return stats, err
	}
	stats.EnglishScanned = englishScanned
	targetIndexes := map[string]map[string]*TargetLemmaInfo{}
	for _, spec := range specs {
		stats.WantedCounts[spec.LangCode] = len(wantedSets[spec.LangCode])
		idx, scanned, matched, err := scanTargetDump(spec, wantedSets[spec.LangCode], cfg)
		if err != nil {
			return stats, err
		}
		targetIndexes[spec.LangCode] = idx
		stats.TargetScanned[spec.LangCode] = scanned
		stats.TargetMatched[spec.LangCode] = matched
	}

	debug, err := newDebugWriters(cfg)
	if err != nil {
		return stats, err
	}
	defer debug.Close()

	r, closeFn, err := openPossiblyGzipped(cfg.EnglishInputPath)
	if err != nil {
		return stats, err
	}
	defer closeFn()

	coreFile, err := os.Create(cfg.CoreOutputPath)
	if err != nil {
		return stats, fmt.Errorf("create core output file: %w", err)
	}
	defer coreFile.Close()

	extendedFile, err := os.Create(cfg.ExtendedOutputPath)
	if err != nil {
		return stats, fmt.Errorf("create extended output file: %w", err)
	}
	defer extendedFile.Close()

	coreWriter := bufio.NewWriterSize(coreFile, 1024*1024)
	defer coreWriter.Flush()

	extendedWriter := bufio.NewWriterSize(extendedFile, 1024*1024)
	defer extendedWriter.Flush()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			stats.SkippedEmptyLine++
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			stats.SkippedInvalidJSON++
			continue
		}

		if entry.LangCode != "en" {
			stats.SkippedByLang++
			continue
		}

		pos, ok := canonicalPOS(entry.Pos)
		if !ok {
			stats.SkippedByPOS++
			_ = debug.Write("skipped_by_pos", DebugSkippedEntry{
				Reason:         "skipped_by_pos",
				Word:           entry.Word,
				NormalizedWord: normalizeEnglishLemma(entry.Word),
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				Glosses:        collectGlosses(entry, cfg.MaxGlosses),
				Translations:   limitRawTranslations(entry.Translations, 20),
				Note:           "raw pos is not in allowedPOS",
			})
			continue
		}

		lemma := normalizeEnglishLemma(entry.Word)
		if !isSimpleEnglishLemma(lemma, cfg.AllowMultiwordEN) {
			stats.SkippedByLemma++
			_ = debug.Write("skipped_by_lemma", DebugSkippedEntry{
				Reason:         "skipped_by_lemma",
				Word:           entry.Word,
				NormalizedWord: lemma,
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				CanonicalPOS:   pos,
				Glosses:        collectGlosses(entry, cfg.MaxGlosses),
				Translations:   limitRawTranslations(entry.Translations, 20),
				Note:           "english lemma did not pass isSimpleEnglishLemma",
			})
			continue
		}

		glosses := collectGlosses(entry, cfg.MaxGlosses)
		if cfg.RequireGloss && len(glosses) == 0 {
			stats.SkippedByGloss++
			_ = debug.Write("skipped_by_gloss", DebugSkippedEntry{
				Reason:         "skipped_by_gloss",
				Word:           entry.Word,
				NormalizedWord: lemma,
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				CanonicalPOS:   pos,
				Translations:   limitRawTranslations(entry.Translations, 20),
				Note:           "entry has no usable glosses after collectGlosses",
			})
			continue
		}

		rawTargets := map[string][]string{}
		states := map[string]CandidateTargetState{}
		rawAny := false

		for _, spec := range specs {
			relaxed := collectTranslationsRelaxed(entry, spec.LangCode, cfg.AllowMultiwordExtended, cfg.MaxTranslationsPerLang)
			strict := collectTranslationsStrict(entry, spec.LangCode, cfg.AllowMultiwordCore, cfg.MaxTranslationsPerLang)
			if len(relaxed) > 0 {
				rawAny = true
			}
			rawTargets[spec.LangCode] = relaxed
			states[spec.LangCode] = CandidateTargetState{
				FromEnglish: relaxed,
				StrictValidated: []string{},
				SoftValidated: []string{},
				Completed: []string{},
			}
			for _, tr := range strict {
				if info, ok := targetIndexes[spec.LangCode][normalizeLexemeKey(tr)]; ok && info.HasGloss {
					st := states[spec.LangCode]
					st.StrictValidated = mergeUniqueOrdered(st.StrictValidated, []string{info.Lemma})
					st.SoftValidated = mergeUniqueOrdered(st.SoftValidated, []string{info.Lemma})
					states[spec.LangCode] = st
				}
			}
			for _, tr := range relaxed {
				if info, ok := targetIndexes[spec.LangCode][normalizeLexemeKey(tr)]; ok {
					st := states[spec.LangCode]
					st.SoftValidated = mergeUniqueOrdered(st.SoftValidated, []string{info.Lemma})
					states[spec.LangCode] = st
				}
			}
		}

		if !rawAny {
			stats.SkippedByNoCandidateTranslations++
			_ = debug.Write("skipped_by_no_candidate_translations", DebugSkippedEntry{
				Reason:         "skipped_by_no_candidate_translations",
				Word:           entry.Word,
				NormalizedWord: lemma,
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				CanonicalPOS:   pos,
				Glosses:        glosses,
				Translations:   limitRawTranslations(entry.Translations, 20),
				Note:           "english entry has no usable target-language candidates after relaxed extraction",
			})
			continue
		}

		completeStatesSoft(lemma, states, targetIndexes, specs, cfg.MaxTranslationsPerLang)

		candidate := Stage31Candidate{
			EnLemma:    lemma,
			Pos:        pos,
			Glosses:    glosses,
			Targets:    states,
			Source:     "english candidates + soft target validation",
		}
		candidate.Status, candidate.Layer, candidate.Confidence, candidate.Flags = classifyCandidate(candidate, specs)
		stats.StatusCounts[candidate.Status]++

		if candidate.Status == "candidate_only" {
			_ = debug.Write("candidate_only", DebugSkippedEntry{
				Reason:         "candidate_only",
				Word:           entry.Word,
				NormalizedWord: lemma,
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				CanonicalPOS:   pos,
				Glosses:        glosses,
				Translations:   limitRawTranslations(entry.Translations, 30),
				RawTargets:     rawTargets,
				States:         states,
				Note:           "kept in extended, but no target side reached soft validation",
			})
		}
		if candidate.Status == "half_validated" {
			_ = debug.Write("half_validated", DebugSkippedEntry{
				Reason:         "half_validated",
				Word:           entry.Word,
				NormalizedWord: lemma,
				RawLangCode:    entry.LangCode,
				RawPOS:         entry.Pos,
				CanonicalPOS:   pos,
				Glosses:        glosses,
				Translations:   limitRawTranslations(entry.Translations, 30),
				RawTargets:     rawTargets,
				States:         states,
				Note:           "kept in extended with only one target side soft-validated/completed",
			})
		}

		if cfg.LimitExtended == 0 || stats.WrittenExtendedAll < cfg.LimitExtended {
			b, err := marshalLine(candidate)
			if err != nil {
				return stats, fmt.Errorf("marshal extended candidate: %w", err)
			}
			if _, err := extendedWriter.Write(b); err != nil {
				return stats, fmt.Errorf("write extended candidate: %w", err)
			}
			stats.WrittenExtendedAll++
		}

		if candidate.Layer == "core" && (cfg.LimitCore == 0 || stats.WrittenCore < cfg.LimitCore) {
			b, err := marshalLine(candidate)
			if err != nil {
				return stats, fmt.Errorf("marshal core candidate: %w", err)
			}
			if _, err := coreWriter.Write(b); err != nil {
				return stats, fmt.Errorf("write core candidate: %w", err)
			}
			stats.WrittenCore++
		}
	}

	if err := scanner.Err(); err != nil {
		return stats, fmt.Errorf("scan english input for stage3.1: %w", err)
	}
	if err := coreWriter.Flush(); err != nil {
		return stats, fmt.Errorf("flush core output: %w", err)
	}
	if err := extendedWriter.Flush(); err != nil {
		return stats, fmt.Errorf("flush extended output: %w", err)
	}
	if err := writeJSONFile(cfg.StatsPath, stats); err != nil {
		return stats, err
	}
	return stats, nil
}

func completeStatesSoft(enLemma string, states map[string]CandidateTargetState, indexes map[string]map[string]*TargetLemmaInfo, specs []TargetSpec, limit int) {
	enKey := normalizeLexemeKey(enLemma)

	for _, targetSpec := range specs {
		targetState := states[targetSpec.LangCode]
		if len(targetState.SoftValidated) > 0 {
			continue
		}

		collected := []string{}
		for _, sourceSpec := range specs {
			if sourceSpec.LangCode == targetSpec.LangCode {
				continue
			}
			sourceState := states[sourceSpec.LangCode]
			sourceVals := mergeUniqueOrdered(sourceState.SoftValidated, sourceState.Completed)
			for _, sourceLemma := range sourceVals {
				sourceInfo, ok := indexes[sourceSpec.LangCode][normalizeLexemeKey(sourceLemma)]
				if !ok {
					continue
				}
				peerCandidates := sourceInfo.PeerTranslations[targetSpec.LangCode]
				for _, cand := range peerCandidates {
					targetInfo, ok := indexes[targetSpec.LangCode][normalizeLexemeKey(cand)]
					if !ok {
						continue
					}
					bridgeMatch := false
					if _, ok := sourceInfo.bridgeKeys[enKey]; ok {
						bridgeMatch = true
					}
					if _, ok := targetInfo.bridgeKeys[enKey]; ok {
						bridgeMatch = true
					}
					if !bridgeMatch && (len(sourceInfo.BridgeTranslations) > 0 || len(targetInfo.BridgeTranslations) > 0) {
						continue
					}
					collected = addUnique(collected, targetInfo.Lemma)
					if limit > 0 && len(collected) >= limit {
						break
					}
				}
				if limit > 0 && len(collected) >= limit {
					break
				}
			}
			if limit > 0 && len(collected) >= limit {
				break
			}
		}
		targetState.Completed = mergeUniqueOrdered(targetState.Completed, collected)
		states[targetSpec.LangCode] = targetState
	}
}

func classifyCandidate(c Stage31Candidate, specs []TargetSpec) (status, layer, confidence string, flags []string) {
	allStrict := true
	allSoft := true
	anySoft := false
	anyCompleted := false

	for _, spec := range specs {
		st := c.Targets[spec.LangCode]
		hasStrict := len(st.StrictValidated) > 0
		hasSoft := len(st.SoftValidated) > 0 || len(st.Completed) > 0
		if !hasStrict {
			allStrict = false
		}
		if !hasSoft {
			allSoft = false
		}
		if hasSoft {
			anySoft = true
		}
		if len(st.Completed) > 0 {
			anyCompleted = true
		}
		if len(st.FromEnglish) > 0 {
			flags = addUnique(flags, "has_"+spec.LangCode+"_candidate")
		}
		if len(st.SoftValidated) > 0 {
			flags = addUnique(flags, "soft_validated_"+spec.LangCode)
		}
		if len(st.Completed) > 0 {
			flags = addUnique(flags, "completed_"+spec.LangCode)
		}
	}

	if allStrict {
		flags = addUnique(flags, "strict_validated_all")
		return "strict_validated_all", "core", "high", flags
	}
	if allSoft {
		if anyCompleted {
			flags = addUnique(flags, "soft_completed_all")
			return "soft_completed_all", "extended", "medium", flags
		}
		flags = addUnique(flags, "soft_validated_all")
		return "soft_validated_all", "extended", "medium", flags
	}
	if anySoft {
		flags = addUnique(flags, "half_validated")
		return "half_validated", "extended", "low", flags
	}
	flags = addUnique(flags, "candidate_only")
	return "candidate_only", "extended", "candidate", flags
}

func limitRawTranslations(in []Translation, limit int) []Translation {
	if limit <= 0 || len(in) <= limit {
		return in
	}
	out := make([]Translation, limit)
	copy(out, in[:limit])
	return out
}

func writeJSONFile(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create json file %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json file %s: %w", path, err)
	}
	return nil
}

func addUniqueFlag(flags []string, flag string) []string {
	if slices.Contains(flags, flag) {
		return flags
	}
	return append(flags, flag)
}
