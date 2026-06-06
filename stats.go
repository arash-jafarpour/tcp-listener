package main

import (
	"sync/atomic"
)

type Stats struct {
	ActiveConnections atomic.Int64
	TotalConnections  atomic.Uint64
	BytesRead         atomic.Uint64
	BytesWritten      atomic.Uint64
}

var ServerStats Stats
