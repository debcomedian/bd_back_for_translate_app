package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	cfg := Config{}

	flag.StringVar(&cfg.EnglishInputPath, "english-input", "", "path to English Kaikki JSONL or JSONL.GZ file")
	flag.StringVar(&cfg.RussianInputPath, "russian-input", "", "path to Russian Kaikki JSONL or JSONL.GZ file")
	flag.StringVar(&cfg.GermanInputPath, "german-input", "", "path to German Kaikki JSONL or JSONL.GZ file")
	flag.StringVar(&cfg.CoreOutputPath, "core-output", "data/lexicon/stage3_1_core_soft.jsonl", "path to strict core output JSONL")
	flag.StringVar(&cfg.ExtendedOutputPath, "extended-output", "data/lexicon/stage3_1_extended_soft.jsonl", "path to soft-validated extended output JSONL")
	flag.StringVar(&cfg.StatsPath, "stats", "data/lexicon/stage3_1_stats.json", "path to output stats JSON file")

	flag.BoolVar(&cfg.AllowMultiwordEN, "allow-multiword-en", false, "allow English lemmas with spaces")
	flag.BoolVar(&cfg.AllowMultiwordCore, "allow-multiword-core", false, "allow multiword translations in core")
	flag.BoolVar(&cfg.AllowMultiwordExtended, "allow-multiword-extended", true, "allow multiword translations in extended")
	flag.BoolVar(&cfg.RequireGloss, "require-gloss", true, "require at least one gloss")
	flag.IntVar(&cfg.MaxGlosses, "max-glosses", 3, "maximum number of glosses to keep per candidate")
	flag.IntVar(&cfg.MaxTranslationsPerLang, "max-translations-per-lang", 12, "maximum number of translations per target language")
	flag.IntVar(&cfg.LimitCore, "limit-core", 0, "optional limit of written core candidates, 0 = no limit")
	flag.IntVar(&cfg.LimitExtended, "limit-extended", 0, "optional limit of written extended candidates, 0 = no limit")

	flag.StringVar(&cfg.DebugDir, "debug-dir", "", "directory for debug samples")
	flag.IntVar(&cfg.DebugSampleLimitPerReason, "debug-sample-limit-per-reason", 200, "maximum number of debug samples per reason, 0 disables debug output")
	flag.Parse()

	if cfg.EnglishInputPath == "" || cfg.RussianInputPath == "" || cfg.GermanInputPath == "" {
		fmt.Fprintln(os.Stderr, "missing required flags: -english-input, -russian-input, -german-input")
		flag.Usage()
		os.Exit(2)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	stats, err := RunStage31(cfg)
	if err != nil {
		log.Fatalf("stage3.1 failed: %v", err)
	}

	log.Printf("done: english_scanned=%d core=%d extended_all=%d strict_all=%d soft_all=%d half_validated=%d candidate_only=%d",
		stats.EnglishScanned,
		stats.WrittenCore,
		stats.WrittenExtendedAll,
		stats.StatusCounts["strict_validated_all"],
		stats.StatusCounts["soft_validated_all"]+stats.StatusCounts["soft_completed_all"],
		stats.StatusCounts["half_validated"],
		stats.StatusCounts["candidate_only"],
	)
	log.Printf("core output:     %s", cfg.CoreOutputPath)
	log.Printf("extended output: %s", cfg.ExtendedOutputPath)
	log.Printf("stats:           %s", cfg.StatsPath)
	if cfg.DebugDir != "" && cfg.DebugSampleLimitPerReason > 0 {
		log.Printf("debug dir:       %s", cfg.DebugDir)
	}
}
