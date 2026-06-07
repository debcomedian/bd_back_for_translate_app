package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func RunNewman(cfg Config) error {
	if err := os.MkdirAll(cfg.NewmanReportDir, 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(cfg.NewmanCollection); err != nil {
		return fmt.Errorf("Newman collection недоступен: %s: %w", cfg.NewmanCollection, err)
	}
	if _, err := os.Stat(cfg.NewmanEnvironment); err != nil {
		return fmt.Errorf("Newman environment недоступен: %s: %w", cfg.NewmanEnvironment, err)
	}

	junitReport := filepath.Join(cfg.NewmanReportDir, "newman-report.xml")
	bin, err := resolveNewmanBin(cfg)
	if err != nil {
		return err
	}

	args := []string{
		"run", cfg.NewmanCollection,
		"-e", cfg.NewmanEnvironment,
		"--reporters", "cli,junit",
		"--reporter-junit-export", junitReport,
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Newman завершился с ошибкой: %w", err)
	}

	fmt.Printf("[test] Newman report: %s\n", junitReport)
	return nil
}

func resolveNewmanBin(cfg Config) (string, error) {
	if cfg.NewmanBin != "" {
		return cfg.NewmanBin, nil
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA пустой, задайте NEWMAN_BIN или установите newman через npm install -g newman")
		}
		return filepath.Join(appData, "npm", "newman.cmd"), nil
	}

	return "newman", nil
}
