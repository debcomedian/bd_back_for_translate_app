package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	cfg := Config{}

	flag.StringVar(&cfg.EnglishInputPath, "english-input", "", "путь к английскому файлу Kaikki JSONL или JSONL.GZ")
	flag.StringVar(&cfg.RussianInputPath, "russian-input", "", "путь к русскому файлу Kaikki JSONL или JSONL.GZ")
	flag.StringVar(&cfg.GermanInputPath, "german-input", "", "путь к немецкому файлу Kaikki JSONL или JSONL.GZ")
	flag.StringVar(&cfg.CoreOutputPath, "core-output", "data/lexicon/stage3_1_core_soft.jsonl", "путь к основному JSONL-файлу строгой выборки")
	flag.StringVar(&cfg.ExtendedOutputPath, "extended-output", "data/lexicon/stage3_1_extended_soft.jsonl", "путь к расширенному JSONL-файлу мягкой валидации")
	flag.StringVar(&cfg.StatsPath, "stats", "data/lexicon/stage3_1_stats.json", "путь к выходному JSON-файлу статистики")

	flag.BoolVar(&cfg.AllowMultiwordEN, "allow-multiword-en", false, "разрешить английские леммы с пробелами")
	flag.BoolVar(&cfg.AllowMultiwordCore, "allow-multiword-core", false, "разрешить многословные переводы в основном наборе")
	flag.BoolVar(&cfg.AllowMultiwordExtended, "allow-multiword-extended", true, "разрешить многословные переводы в расширенном наборе")
	flag.BoolVar(&cfg.RequireGloss, "require-gloss", true, "требовать минимум одно толкование")
	flag.IntVar(&cfg.MaxGlosses, "max-glosses", 3, "максимальное количество толкований для одного кандидата")
	flag.IntVar(&cfg.MaxTranslationsPerLang, "max-translations-per-lang", 12, "максимальное количество переводов для одного целевого языка")
	flag.IntVar(&cfg.LimitCore, "limit-core", 0, "необязательный лимит записей основного набора, 0 — без ограничения")
	flag.IntVar(&cfg.LimitExtended, "limit-extended", 0, "необязательный лимит записей расширенного набора, 0 — без ограничения")

	flag.StringVar(&cfg.DebugDir, "debug-dir", "", "каталог для отладочных примеров")
	flag.IntVar(&cfg.DebugSampleLimitPerReason, "debug-sample-limit-per-reason", 200, "максимальное количество отладочных примеров на одну причину, 0 отключает отладочный вывод")
	flag.Parse()

	if cfg.EnglishInputPath == "" || cfg.RussianInputPath == "" || cfg.GermanInputPath == "" {
		fmt.Fprintln(os.Stderr, "не указаны обязательные флаги: -english-input, -russian-input, -german-input")
		flag.Usage()
		os.Exit(2)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	stats, err := RunStage31(cfg)
	if err != nil {
		log.Fatalf("Ошибка выполнения этапа Stage 3.1: %v", err)
	}

	log.Printf("Готово: просмотрено_английских_записей=%d основной_набор=%d расширенный_набор=%d строгая_валидация=%d мягкая_валидация=%d частичная_валидация=%d только_кандидаты=%d",
		stats.EnglishScanned,
		stats.WrittenCore,
		stats.WrittenExtendedAll,
		stats.StatusCounts["strict_validated_all"],
		stats.StatusCounts["soft_validated_all"]+stats.StatusCounts["soft_completed_all"],
		stats.StatusCounts["half_validated"],
		stats.StatusCounts["candidate_only"],
	)
	log.Printf("Основной выходной файл:     %s", cfg.CoreOutputPath)
	log.Printf("Расширенный выходной файл:  %s", cfg.ExtendedOutputPath)
	log.Printf("Файл статистики:            %s", cfg.StatsPath)
	if cfg.DebugDir != "" && cfg.DebugSampleLimitPerReason > 0 {
		log.Printf("Каталог отладки:            %s", cfg.DebugDir)
	}
}
