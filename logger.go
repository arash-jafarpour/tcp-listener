package main

import (
	"encoding/json"
	"io"
	"log"
)

type Logger struct {
	writer io.Writer
}

func NewLogger(w io.Writer) *Logger {
	return &Logger{
		writer: w,
	}
}

func (l *Logger) Log(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf(`{"event":"logger_error","error":"%v"}`, err)
		return
	}

	_, _ = l.writer.Write(append(b, '\n'))
}

var AppLogger *Logger
