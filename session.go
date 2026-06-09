package main

import (
	"os"
	"path/filepath"
	"time"
)

func OpenSessionLog() (*os.File, error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, err
	}

	name := time.Now().
		Format("2006-01-02_150405") + ".log"

	path := filepath.Join("logs", name)

	return os.Create(path)
}
