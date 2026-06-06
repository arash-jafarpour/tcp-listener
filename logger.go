package main

import (
	"encoding/json"
	"log"
)

func Log(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf(`{"event":"logger_error","error":"%v"}`, err)
		return
	}

	log.Println(string(b))
}
