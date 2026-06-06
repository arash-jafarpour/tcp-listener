package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg := ParseConfig()

	addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	Log(StartEvent{
		Event:    "start",
		Bind:     addr,
		Verbose:  cfg.Verbose,
		DumpMode: cfg.DumpMode,
	})

	ctx, cancel := context.WithCancel(context.Background())
	startStatsLogger(
		ctx,
		cfg.StatsInterval,
	)
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	var wg sync.WaitGroup

	go func() {
		<-stop

		Log(ShutdownEvent{
			Event:  "shutdown",
			Status: "draining",
		})

		cancel()

		if err := l.Close(); err != nil {
			Log(ListenerErrorEvent{
				Event: "listener_error",
				Error: err.Error(),
			})
		}
	}()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					Log(AcceptErrorEvent{
						Event: "accept_error",
						Error: err.Error(),
					})
					return
				}
			}

			tcpConn, ok := conn.(*net.TCPConn)
			if !ok {
				Log(UnexpectedConnTypeEvent{
					Event: "unexpected_conn_type",
					Type:  fmt.Sprintf("%T", conn),
				})
				conn.Close()
				continue
			}

			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)

			tracker := NewConnTracker(tcpConn, &cfg)

			wg.Add(1)

			go func() {
				defer wg.Done()
				tracker.Handle()
			}()
		}
	}()

	<-ctx.Done()

	wg.Wait()

	Log(ShutdownEvent{
		Event:  "shutdown",
		Status: "done",
	})
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
				Log(StatsEvent{
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
