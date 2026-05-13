package main

import (
	"fmt"
	"path/filepath"
)

type Config struct {
	EnglishInputPath          string
	RussianInputPath          string
	GermanInputPath           string
	CoreOutputPath            string
	ExtendedOutputPath        string
	StatsPath                 string

	AllowMultiwordEN          bool
	AllowMultiwordCore        bool
	AllowMultiwordExtended    bool
	RequireGloss              bool
	MaxGlosses                int
	MaxTranslationsPerLang    int
	LimitCore                 int
	LimitExtended             int

	DebugDir                  string
	DebugSampleLimitPerReason int
}

func (c Config) Validate() error {
	if c.EnglishInputPath == "" {
		return fmt.Errorf("english input path is required")
	}
	if c.RussianInputPath == "" {
		return fmt.Errorf("russian input path is required")
	}
	if c.GermanInputPath == "" {
		return fmt.Errorf("german input path is required")
	}
	if c.CoreOutputPath == "" {
		return fmt.Errorf("core output path is required")
	}
	if c.ExtendedOutputPath == "" {
		return fmt.Errorf("extended output path is required")
	}
	if c.StatsPath == "" {
		return fmt.Errorf("stats path is required")
	}
	if c.MaxGlosses <= 0 {
		return fmt.Errorf("max-glosses must be > 0")
	}
	if c.MaxTranslationsPerLang <= 0 {
		return fmt.Errorf("max-translations-per-lang must be > 0")
	}
	if c.LimitCore < 0 || c.LimitExtended < 0 {
		return fmt.Errorf("limits must be >= 0")
	}
	if c.DebugSampleLimitPerReason < 0 {
		return fmt.Errorf("debug-sample-limit-per-reason must be >= 0")
	}

	ep := filepath.Clean(c.EnglishInputPath)
	if ep == filepath.Clean(c.CoreOutputPath) || ep == filepath.Clean(c.ExtendedOutputPath) {
		return fmt.Errorf("english input and output paths must differ")
	}
	if filepath.Clean(c.CoreOutputPath) == filepath.Clean(c.ExtendedOutputPath) {
		return fmt.Errorf("core and extended outputs must differ")
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
