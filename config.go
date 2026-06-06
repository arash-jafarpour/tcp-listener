package main

import (
	"flag"
	"time"
)

type Config struct {
	BindAddr      string
	Port          int
	Verbose       bool
	DumpMode      string
	StatsInterval time.Duration
}

func ParseConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.BindAddr, "bind", "0.0.0.0", "address to bind")
	flag.IntVar(&cfg.Port, "port", 9000, "listen port")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "log every read/write event")
	flag.StringVar(
		&cfg.DumpMode,
		"dump",
		"none",
		"payload dump mode: none|hex|hexdump",
	)
	flag.DurationVar(
		&cfg.StatsInterval,
		"stats-interval",
		0,
		"periodic stats logging interval (0 disables)",
	)
	flag.Parse()
	return cfg
}
