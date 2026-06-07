package main

import "os"

func writeSmallFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
