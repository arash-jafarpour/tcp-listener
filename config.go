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

func ParseConfig(args []string) Config {
	cfg := Config{}

	fs := flag.NewFlagSet("listen", flag.ExitOnError)

	fs.StringVar(&cfg.BindAddr, "bind", "0.0.0.0", "address to bind")
	fs.IntVar(&cfg.Port, "port", 9000, "listen port")

	fs.BoolVar(
		&cfg.Verbose,
		"verbose",
		false,
		"log every read/write event",
	)

	fs.StringVar(
		&cfg.DumpMode,
		"dump",
		"none",
		"payload dump mode: none|hex|hexdump",
	)

	fs.DurationVar(
		&cfg.StatsInterval,
		"stats-interval",
		0,
		"periodic stats logging interval",
	)

	fs.Parse(args)

	return cfg
}
