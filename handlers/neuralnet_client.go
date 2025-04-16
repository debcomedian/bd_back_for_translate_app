package handlers

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

type NeuralNetClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

func NewNeuralNetClient(pythonScriptPath string) (*NeuralNetClient, error) {
	cmd := exec.Command("python", pythonScriptPath)
	cmd.Env = append(
		os.Environ(),
		"PYTHONIOENCODING=utf-8",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Ошибка получения STDIN: %v", err)
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Ошибка получения STDOUT: %v", err)
		return nil, err
	}

	client := &NeuralNetClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdoutPipe),
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Ошибка запуска Python процесса: %v", err)
		return nil, err
	}
	return client, nil
}

func (nc *NeuralNetClient) Process(audioPath string) (map[string]interface{}, error) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	req := map[string]string{"audio_path": audioPath}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("Ошибка маршалинга запроса: %v", err)
		return nil, err
	}

	if _, err := nc.stdin.Write(append(reqBytes, '\n')); err != nil {
		log.Printf("Ошибка записи в STDIN: %v", err)
		return nil, err
	}

	respLine, err := nc.stdout.ReadBytes('\n')
	if err != nil {
		log.Printf("Ошибка чтения ответа из STDOUT: %v", err)
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(respLine, &resp); err != nil {
		log.Printf("Ошибка распаковки JSON-ответа: %v", err)
		return nil, err
	}
	return resp, nil
}
