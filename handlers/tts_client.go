package handlers

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

var TtsClient *TTSClient

type TTSClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

/* ---------- start daemon + log ---------- */
func NewTTSClient(pyScript string) (*TTSClient, error) {
	log.Printf("[TTS] launching daemon: %s", pyScript)

	cmd := exec.Command("python", pyScript)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errOut, _ := cmd.StderrPipe() // выводим stderr демона в логи

	if err = cmd.Start(); err != nil {
		return nil, err
	}
	go func() { // зеркалим stderr демона в логи Go — удобно дебажить
		sc := bufio.NewScanner(errOut)
		for sc.Scan() {
			log.Printf("[TTS‑daemon] %s", sc.Text())
		}
	}()
	log.Printf("[TTS] daemon pid=%d started", cmd.Process.Pid)
	return &TTSClient{cmd: cmd, stdin: in, stdout: bufio.NewReader(out)}, nil
}

/* ---------- synthesize with log ---------- */
func (c *TTSClient) Synthesize(ipa, lang string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqJSON, _ := json.Marshal(map[string]string{"text": ipa, "lang": lang})
	log.Printf("[TTS] → %s", reqJSON)
	if _, err := c.stdin.Write(append(reqJSON, '\n')); err != nil {
		return nil, err
	}

	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	log.Printf("[TTS] ← %s", bytes.TrimSpace(line))

	var resp struct {
		Ok     bool   `json:"ok"`
		Error  string `json:"error"`
		WavB64 string `json:"wav_b64"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}
	return base64.StdEncoding.DecodeString(resp.WavB64)
}
