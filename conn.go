package main

import (
	"encoding/hex"
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

	Log(CloseEvent{
		Event: "close",

		ConnID: t.ID,

		Src: t.RemoteIP + ":" + strconv.Itoa(t.RemotePort),
		Dst: t.LocalIP + ":" + strconv.Itoa(t.LocalPort),

		BytesRead:    t.BytesRead.Load(),
		BytesWritten: t.BytesWritten.Load(),

		DurationMS: t.DisconnectedAt.
			Sub(t.ConnectedAt).
			Milliseconds(),

		Error: errStr,

		Time: t.DisconnectedAt.Format(time.RFC3339Nano),
	})
}

func (t *ConnTracker) logData(dir string, n int, buf []byte) {
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
			t.logData("read", n, buf)
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
