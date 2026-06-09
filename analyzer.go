package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

type Analyzer struct {
	Lines          int
	EventCount     map[string]int
	Connections    int
	Protocols      map[string]int
	CloseReasons   map[string]int
	MSSCount       map[int]int
	TotalBytesRead uint64
	PayloadSizes   map[int]int
	Durations      []int64
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		EventCount:   make(map[string]int),
		Protocols:    make(map[string]int),
		CloseReasons: make(map[string]int),
		MSSCount:     make(map[int]int),
		PayloadSizes: make(map[int]int),
	}
}

func (a *Analyzer) ProcessLine(line []byte) error {
	var hdr EventHeader

	if err := json.Unmarshal(line, &hdr); err != nil {
		return err
	}

	a.Lines++
	a.EventCount[hdr.Event]++

	switch hdr.Event {

	case "open":
		var ev OpenEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return err
		}

		a.Connections++
		a.MSSCount[ev.SocketMSS]++

	case "protocol":
		var ev ProtocolEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return err
		}

		a.Protocols[ev.Protocol]++

	case "close":
		var ev CloseEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return err
		}

		a.TotalBytesRead += ev.BytesRead
		a.CloseReasons[ev.CloseReason]++
		a.Durations = append(a.Durations, ev.DurationMS)

	case "data":
		var ev DataEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return err
		}

		a.PayloadSizes[ev.Bytes]++
	}

	return nil
}

func (a *Analyzer) Print() {
	fmt.Printf("Lines Processed: %d\n\n", a.Lines)

	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Printf("Connections: %d\n", a.Connections)
	fmt.Printf("Bytes Read: %d\n\n", a.TotalBytesRead)

	printStringMap("Protocols", a.Protocols)
	printStringMap("Close Reasons", a.CloseReasons)
	printIntMap("MSS Distribution", a.MSSCount)
	printTopPayloadSizes(a.PayloadSizes)

	if len(a.Durations) > 0 {
		printDurationStats(a.Durations)
	}
}

func printStringMap(title string, m map[string]int) {
	fmt.Println(title)
	fmt.Println("-------")

	type kv struct {
		Key   string
		Value int
	}

	var items []kv

	for k, v := range m {
		items = append(items, kv{k, v})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	for _, item := range items {
		fmt.Printf("%-25s %d\n", item.Key, item.Value)
	}

	fmt.Println()
}

func printIntMap(title string, m map[int]int) {
	fmt.Println(title)
	fmt.Println("-------")

	type kv struct {
		Key   int
		Value int
	}

	var items []kv

	for k, v := range m {
		items = append(items, kv{k, v})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	for _, item := range items {
		fmt.Printf("%-10d %d\n", item.Key, item.Value)
	}

	fmt.Println()
}

func printTopPayloadSizes(m map[int]int) {
	fmt.Println("Top Payload Sizes")
	fmt.Println("-----------------")

	type kv struct {
		Size  int
		Count int
	}

	var items []kv

	for size, count := range m {
		items = append(items, kv{
			Size:  size,
			Count: count,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	limit := min(10, len(items))

	for i := range limit {
		fmt.Printf("%-10d %d\n",
			items[i].Size,
			items[i].Count,
		)
	}

	fmt.Println()
}

func printDurationStats(durations []int64) {
	slices.Sort(durations)

	var total int64

	for _, d := range durations {
		total += d
	}

	avg := total / int64(len(durations))

	fmt.Println("Connection Durations")
	fmt.Println("--------------------")

	fmt.Printf("Min: %d ms\n", durations[0])
	fmt.Printf("Avg: %d ms\n", avg)
	fmt.Printf("P50: %d ms\n", percentile(durations, 50))
	fmt.Printf("P95: %d ms\n", percentile(durations, 95))
	fmt.Printf("Max: %d ms\n", durations[len(durations)-1])

	fmt.Println()
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}

	idx := (len(sorted) - 1) * p / 100
	return sorted[idx]
}
