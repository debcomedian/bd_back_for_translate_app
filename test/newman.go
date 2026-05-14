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

	junitReport := filepath.Join(cfg.NewmanReportDir, "newman-report.xml")

	if runtime.GOOS == "windows" {
		bin := cfg.NewmanBin
		if bin == "" {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				return fmt.Errorf("APPDATA пустой, переменная NEWMAN_BIN не задана")
			}
			bin = filepath.Join(appData, "npm", "newman.cmd")
		}

		args := []string{
			"/C", bin,
			"run", cfg.NewmanCollection,
			"-e", cfg.NewmanEnvironment,
			"--reporters", "cli,junit",
			"--reporter-junit-export", junitReport,
		}
		cmd := exec.Command("cmd", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	bin := cfg.NewmanBin
	if bin == "" {
		bin = "newman"
	}

	cmd := exec.Command(
		bin,
		"run", cfg.NewmanCollection,
		"-e", cfg.NewmanEnvironment,
		"--reporters", "cli,junit",
		"--reporter-junit-export", junitReport,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
