package main

import (
	"encoding/hex"
	"errors"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const readBufLen = 65536

type ConnTracker struct {
	ID               string
	RemoteIP         string
	RemotePort       int
	LocalIP          string
	LocalPort        int
	ConnectedAt      time.Time
	DisconnectedAt   time.Time
	BytesRead        atomic.Uint64
	BytesWritten     atomic.Uint64
	LastReadAt       time.Time
	ReadCount        int64
	MinReadSize      int
	MaxReadSize      int
	FirstReadSize    int
	LastReadSize     int
	TotalReadDelta   time.Duration
	MinReadDelta     time.Duration
	MaxReadDelta     time.Duration
	conn             *net.TCPConn
	cfg              *Config
	ifaceMTU         int
	ifaceMSS         int
	protocolDetected atomic.Bool
}

func NewConnTracker(conn *net.TCPConn, cfg *Config) *ConnTracker {
	t := &ConnTracker{
		conn:        conn,
		cfg:         cfg,
		ConnectedAt: time.Now(),
		ID:          NewConnID(),
	}
	remote := conn.RemoteAddr().(*net.TCPAddr)
	local := conn.LocalAddr().(*net.TCPAddr)

	t.RemoteIP = remote.IP.String()
	t.RemotePort = remote.Port
	t.LocalIP = local.IP.String()
	t.LocalPort = local.Port

	t.probeMTU()

	ServerStats.ActiveConnections.Add(1)
	ServerStats.TotalConnections.Add(1)

	t.logOpen()

	return t
}

func findInterfaceByIP(ipStr string) (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.IP.String() == ipStr {
				return &iface, nil
			}
		}
	}
	return nil, io.EOF
}

func (t *ConnTracker) probeMTU() {
	t.ifaceMTU, t.ifaceMSS = t.socketInfo()

	if t.ifaceMTU == 0 || t.ifaceMSS == 0 {
		if iface, err := findInterfaceByIP(t.LocalIP); err == nil {
			if t.ifaceMTU == 0 {
				t.ifaceMTU = iface.MTU
			}

			if t.ifaceMSS == 0 {
				t.ifaceMSS = iface.MTU - 40
			}
		}
	}
}

func (t *ConnTracker) socketInfo() (mtu, mss int) {
	rawConn, err := t.conn.SyscallConn()
	if err != nil {
		return
	}

	_ = rawConn.Control(func(fd uintptr) {
		mss, _ = syscall.GetsockoptInt(
			int(fd),
			syscall.IPPROTO_TCP,
			syscall.TCP_MAXSEG,
		)

		mtu, _ = syscall.GetsockoptInt(
			int(fd),
			syscall.IPPROTO_IP,
			syscall.IP_MTU,
		)
	})

	return
}

func (t *ConnTracker) logOpen() {
	Log(OpenEvent{
		Event: "open",

		ConnID: t.ID,

		Src: t.RemoteIP + ":" + strconv.Itoa(t.RemotePort),
		Dst: t.LocalIP + ":" + strconv.Itoa(t.LocalPort),

		IfaceMTU:  t.ifaceMTU,
		SocketMSS: t.ifaceMSS,

		Time: t.ConnectedAt.Format(time.RFC3339Nano),
	})
}

func (t *ConnTracker) logClose(err error) {
	t.DisconnectedAt = time.Now()

	var errStr string
	if err != nil && err != io.EOF {
		errStr = err.Error()
	}

	ServerStats.ActiveConnections.Add(-1)

	avgReadSize := 0
	if t.ReadCount > 0 {
		avgReadSize = int(t.BytesRead.Load() / uint64(t.ReadCount))
	}

	avgReadDeltaMS := int64(0)
	if t.ReadCount > 1 {
		avgReadDeltaMS = (t.TotalReadDelta /
			time.Duration(t.ReadCount-1)).
			Milliseconds()
	}

	Log(CloseEvent{
		Event:          "close",
		ConnID:         t.ID,
		Src:            t.RemoteIP + ":" + strconv.Itoa(t.RemotePort),
		Dst:            t.LocalIP + ":" + strconv.Itoa(t.LocalPort),
		BytesRead:      t.BytesRead.Load(),
		BytesWritten:   t.BytesWritten.Load(),
		DurationMS:     t.DisconnectedAt.Sub(t.ConnectedAt).Milliseconds(),
		ReadCount:      t.ReadCount,
		AvgReadSize:    avgReadSize,
		MinReadSize:    t.MinReadSize,
		MaxReadSize:    t.MaxReadSize,
		FirstReadSize:  t.FirstReadSize,
		LastReadSize:   t.LastReadSize,
		AvgReadDeltaMS: avgReadDeltaMS,
		MinReadDeltaMS: t.MinReadDelta.Milliseconds(),
		MaxReadDeltaMS: t.MaxReadDelta.Milliseconds(),
		Error:          errStr,
		CloseReason:    closeReason(err),
		Time:           t.DisconnectedAt.Format(time.RFC3339Nano),
	})
}

func (t *ConnTracker) logData(dir string, n int, buf []byte, delta time.Duration) {
	if !t.cfg.Verbose && t.cfg.DumpMode == "none" {
		return
	}

	ev := DataEvent{
		Event:  "data",
		ConnID: t.ID,
		Dir:    dir,
		Bytes:  n,
		Time:   time.Now().Format(time.RFC3339Nano),
	}

	if delta > 0 {
		ev.DeltaMS = delta.Milliseconds()
	}

	switch t.cfg.DumpMode {
	case "hex":
		ev.Hex = hex.EncodeToString(buf[:n])
	case "hexdump":
		ev.Hexdump = hex.Dump(buf[:n])
	}

	Log(ev)
}

func (t *ConnTracker) logEOF() {
	Log(EOFEvent{
		Event:  "eof",
		ConnID: t.ID,
		Time:   time.Now().Format(time.RFC3339Nano),
	})
}

func (t *ConnTracker) logError(err error) {
	Log(ErrorEvent{
		Event:  "error",
		ConnID: t.ID,
		Error:  err.Error(),
		Time:   time.Now().Format(time.RFC3339Nano),
	})
}

func (t *ConnTracker) Handle() {
	t.readLoop()
}

func (t *ConnTracker) readLoop() {
	buf := make([]byte, readBufLen)
	for {
		n, err := t.conn.Read(buf)
		if n > 0 {
			now := time.Now()

			var delta time.Duration

			if t.ReadCount > 0 {
				delta = now.Sub(t.LastReadAt)

				t.TotalReadDelta += delta

				if t.ReadCount == 1 {
					t.MinReadDelta = delta
					t.MaxReadDelta = delta
				} else {
					if delta < t.MinReadDelta {
						t.MinReadDelta = delta
					}
					if delta > t.MaxReadDelta {
						t.MaxReadDelta = delta
					}
				}
			}

			t.LastReadAt = now
			t.ReadCount++
			t.LastReadSize = n

			if t.ReadCount == 1 {
				t.FirstReadSize = n
				t.MinReadSize = n
				t.MaxReadSize = n
			} else {
				if n < t.MinReadSize {
					t.MinReadSize = n
				}
				if n > t.MaxReadSize {
					t.MaxReadSize = n
				}
			}
			if !t.protocolDetected.Load() {
				proto := DetectProtocol(buf[:n])
				Log(ProtocolEvent{
					Event:    "protocol",
					ConnID:   t.ID,
					Protocol: proto,
					Time:     time.Now().Format(time.RFC3339Nano),
				})
				t.protocolDetected.Store(true)
			}

			t.BytesRead.Add(uint64(n))
			ServerStats.BytesRead.Add(uint64(n))
			t.logData("read", n, buf, delta)
		}
		if err == io.EOF {
			t.logEOF()
			t.logClose(nil)
			t.conn.Close()
			return
		}
		if err != nil {
			t.logError(err)
			t.logClose(err)
			t.conn.Close()
			return
		}
	}
}

func closeReason(err error) string {
	if err == nil || err == io.EOF {
		return "eof"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return "connection_reset_by_peer"
	}
	if errors.Is(err, syscall.EPIPE) {
		return "broken_pipe"
	}
	return ""
}
