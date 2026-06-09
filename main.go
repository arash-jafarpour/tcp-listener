package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage:")
		fmt.Println("  tcp-listener listen")
		fmt.Println("  tcp-listener analyze <logfile>")
		os.Exit(1)
	}

	switch os.Args[1] {

	case "listen":
		runListen(os.Args[2:])

	case "analyze":
		runAnalyze(os.Args[2:])

	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func startStatsLogger(
	ctx context.Context,
	interval time.Duration,
) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {

			case <-ctx.Done():
				return

			case <-ticker.C:
				AppLogger.Log(StatsEvent{
					Event: "stats",

					ActiveConnections: ServerStats.ActiveConnections.Load(),
					TotalConnections:  ServerStats.TotalConnections.Load(),

					BytesRead:    ServerStats.BytesRead.Load(),
					BytesWritten: ServerStats.BytesWritten.Load(),

					Goroutines: runtime.NumGoroutine(),
				})
			}
		}
	}()
}
