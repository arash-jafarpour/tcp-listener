package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var connSeq atomic.Uint64

func NewConnID() string {
	n := connSeq.Add(1)
	ms := time.Now().UnixMilli()
	return fmt.Sprintf("tc-%x-%04x", ms, n)
}
