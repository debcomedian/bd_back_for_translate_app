package main

import (
	"fmt"
	"path/filepath"
)

type Config struct {
	EnglishInputPath   string
	RussianInputPath   string
	GermanInputPath    string
	CoreOutputPath     string
	ExtendedOutputPath string
	StatsPath          string

	AllowMultiwordEN       bool
	AllowMultiwordCore     bool
	AllowMultiwordExtended bool
	RequireGloss           bool
	MaxGlosses             int
	MaxTranslationsPerLang int
	LimitCore              int
	LimitExtended          int

	DebugDir                  string
	DebugSampleLimitPerReason int
}

func (c Config) Validate() error {
	if c.EnglishInputPath == "" {
		return fmt.Errorf("не указан путь к английскому входному файлу")
	}
	if c.RussianInputPath == "" {
		return fmt.Errorf("не указан путь к русскому входному файлу")
	}
	if c.GermanInputPath == "" {
		return fmt.Errorf("не указан путь к немецкому входному файлу")
	}
	if c.CoreOutputPath == "" {
		return fmt.Errorf("не указан путь к основному выходному файлу")
	}
	if c.ExtendedOutputPath == "" {
		return fmt.Errorf("не указан путь к расширенному выходному файлу")
	}
	if c.StatsPath == "" {
		return fmt.Errorf("не указан путь к файлу статистики")
	}
	if c.MaxGlosses <= 0 {
		return fmt.Errorf("параметр max-glosses должен быть больше 0")
	}
	if c.MaxTranslationsPerLang <= 0 {
		return fmt.Errorf("параметр max-translations-per-lang должен быть больше 0")
	}
	if c.LimitCore < 0 || c.LimitExtended < 0 {
		return fmt.Errorf("лимиты должны быть больше или равны 0")
	}
	if c.DebugSampleLimitPerReason < 0 {
		return fmt.Errorf("параметр debug-sample-limit-per-reason должен быть больше или равен 0")
	}

	ep := filepath.Clean(c.EnglishInputPath)
	if ep == filepath.Clean(c.CoreOutputPath) || ep == filepath.Clean(c.ExtendedOutputPath) {
		return fmt.Errorf("пути английского входного и выходного файлов должны отличаться")
	}
	if filepath.Clean(c.CoreOutputPath) == filepath.Clean(c.ExtendedOutputPath) {
		return fmt.Errorf("пути основного и расширенного выходных файлов должны отличаться")
	}
	return nil
}

func (c Config) TargetSpecs() []TargetSpec {
	return []TargetSpec{
		{
			LangCode:   "ru",
			InputPath:  c.RussianInputPath,
			BridgeLang: "en",
			PeerLangs:  []string{"de"},
		},
		{
			LangCode:   "de",
			InputPath:  c.GermanInputPath,
			BridgeLang: "en",
			PeerLangs:  []string{"ru"},
		},
	}
}
