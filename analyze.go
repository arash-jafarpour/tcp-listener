package main

import (
	"bufio"
	"log"
	"os"
)

func AnalyzeFile(path string) (*Analyzer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	analyzer := NewAnalyzer()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if err := analyzer.ProcessLine(scanner.Bytes()); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return analyzer, nil
}

func runAnalyze(args []string) {
	if len(args) != 1 {
		log.Fatal("usage: tcp-listener analyze <logfile>")
	}

	analyzer, err := AnalyzeFile(args[0])
	if err != nil {
		log.Fatal(err)
	}

	analyzer.Print()
}
