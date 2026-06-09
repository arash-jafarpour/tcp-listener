package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func runListen(args []string) {
	cfg := ParseConfig(args)

	logFile, err := OpenSessionLog()
	if err != nil {
		log.Fatalf("open session log: %v", err)
	}
	defer logFile.Close()

	mw := io.MultiWriter(
		os.Stdout,
		logFile,
	)

	AppLogger = NewLogger(mw)

	addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	AppLogger.Log(StartEvent{
		Event:       "start",
		Bind:        addr,
		Verbose:     cfg.Verbose,
		DumpMode:    cfg.DumpMode,
		SessionFile: logFile.Name(),
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

		AppLogger.Log(ShutdownEvent{
			Event:  "shutdown",
			Status: "draining",
		})

		cancel()

		if err := l.Close(); err != nil {
			AppLogger.Log(ListenerErrorEvent{
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
					AppLogger.Log(AcceptErrorEvent{
						Event: "accept_error",
						Error: err.Error(),
					})
					return
				}
			}

			tcpConn, ok := conn.(*net.TCPConn)
			if !ok {
				AppLogger.Log(UnexpectedConnTypeEvent{
					Event: "unexpected_conn_type",
					Type:  fmt.Sprintf("%T", conn),
				})
				conn.Close()
				continue
			}

			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)

			tracker := NewConnTracker(tcpConn, &cfg)

			wg.Go(func() {
				tracker.Handle()
			})
			// wg.Add(1)
			//
			// go func() {
			// 	defer wg.Done()
			// 	tracker.Handle()
			// }()
		}
	}()

	<-ctx.Done()

	wg.Wait()

	AppLogger.Log(ShutdownEvent{
		Event:  "shutdown",
		Status: "done",
	})
}
