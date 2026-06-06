package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"bd_back_for_translate_app/lexicon"
)

func main() {
	input := flag.String("input", "data/lexicon/active_bank.jsonl", "path to active_bank.jsonl")
	jsonOut := flag.String("json", "", "optional path to write JSON report")
	mdOut := flag.String("md", "", "optional path to write Markdown report")
	failOnGate := flag.Bool("fail-on-gate", false, "exit with code 2 when quality gate fails")
	maxHalf := flag.Float64("max-half-pct", 20, "max allowed half_validated percent")
	maxTail := flag.Float64("max-tail-pct", 35, "max allowed tail percent")
	minFull := flag.Float64("min-full-trilingual-pct", 80, "min allowed full trilingual percent")
	maxSyn := flag.Int("max-synonyms", 10, "max allowed candidate/synonym values per language row")
	allowUnsafe := flag.Bool("allow-unsafe", false, "do not fail/report unsafe candidates as blocking quality gate")
	flag.Parse()

	opts := lexicon.DefaultBankQualityOptions()
	opts.MaxHalfValidatedPct = *maxHalf
	opts.MaxTailPct = *maxTail
	opts.MinFullTrilingualPct = *minFull
	opts.MaxSynonymsPerLang = *maxSyn
	opts.AllowUnsafeCandidates = *allowUnsafe

	report, err := lexicon.AuditActiveBankFile(*input, opts)
	if err != nil {
		log.Fatal(err)
	}

	printConsoleReport(report)

	if *jsonOut != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*jsonOut, data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\nJSON report written: %s\n", *jsonOut)
	}

	if *mdOut != "" {
		if err := report.WriteMarkdown(*mdOut); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Markdown report written: %s\n", *mdOut)
	}

	if *failOnGate && !report.Summary.QualityGatePassed {
		os.Exit(2)
	}
}

func printConsoleReport(r *lexicon.BankQualityReport) {
	fmt.Println("=== ACTIVE BANK QUALITY ===")
	fmt.Printf("input: %s\n", r.InputPath)
	fmt.Printf("rows: %d\n", r.TotalRows)
	fmt.Printf("strict_validated_all: %.2f%%\n", r.Summary.StrictValidatedAllPercent)
	fmt.Printf("half_validated: %.2f%%\n", r.Summary.HalfValidatedPercent)
	fmt.Printf("tail: %.2f%%\n", r.Summary.TailPercent)
	fmt.Printf("full_trilingual: %.2f%%\n", r.Summary.FullTrilingualPercent)
	fmt.Printf("rows_with_issues: %.2f%%\n", r.Summary.RowsWithIssuesPercent)
	fmt.Printf("quality_gate_passed: %t\n", r.Summary.QualityGatePassed)
	fmt.Printf("quality_gate_reason: %s\n", r.Summary.QualityGateReason)

	fmt.Println("\n=== STATUS COUNTS ===")
	for k, v := range r.StatusCounts {
		fmt.Printf("%s=%d\n", k, v)
	}

	fmt.Println("\n=== BUCKET COUNTS ===")
	for k, v := range r.BucketCounts {
		fmt.Printf("%s=%d\n", k, v)
	}

	fmt.Println("\n=== COVERAGE ===")
	fmt.Printf("has_en=%d has_ru=%d has_de=%d full_trilingual=%d en_ru_only=%d en_de_only=%d en_only=%d other_partial=%d\n",
		r.TargetCoverage.HasEN,
		r.TargetCoverage.HasRU,
		r.TargetCoverage.HasDE,
		r.TargetCoverage.FullTrilingual,
		r.TargetCoverage.ENRUOnly,
		r.TargetCoverage.ENDEOnly,
		r.TargetCoverage.ENOnly,
		r.TargetCoverage.OtherPartial,
	)

	fmt.Println("\n=== ISSUE COUNTS ===")
	for k, v := range r.IssueCounts {
		fmt.Printf("%s=%d\n", k, v)
	}

	fmt.Println("\n=== TOP RISK ROWS ===")
	limit := len(r.TopRiskRows)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		row := r.TopRiskRows[i]
		fmt.Printf("line=%d risk=%d lemma=%s status=%s bucket=%s labels=%v ru=%s de=%s\n", row.Line, row.RiskScore, row.Lemma, row.Status, row.Bucket, row.RiskLabels, row.RU, row.DE)
	}
}
