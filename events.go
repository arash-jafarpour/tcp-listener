package main

// NOTE: omitempty removes empty fields automatically.

type OpenEvent struct {
	Event     string `json:"event"`
	ConnID    string `json:"conn_id"`
	Src       string `json:"src"`
	Dst       string `json:"dst"`
	IfaceMTU  int    `json:"iface_mtu"`
	SocketMSS int    `json:"socket_mss"`
	Time      string `json:"time"`
}

type DataEvent struct {
	Event   string `json:"event"`
	ConnID  string `json:"conn_id"`
	Dir     string `json:"dir"`
	Bytes   int    `json:"bytes"`
	DeltaMS int64  `json:"delta_ms,omitempty"`
	Hex     string `json:"hex,omitempty"`
	Hexdump string `json:"hexdump,omitempty"`
	Time    string `json:"time"`
}

type CloseEvent struct {
	Event          string `json:"event"`
	ConnID         string `json:"conn_id"`
	Src            string `json:"src"`
	Dst            string `json:"dst"`
	BytesRead      uint64 `json:"bytes_read"`
	BytesWritten   uint64 `json:"bytes_written"`
	DurationMS     int64  `json:"duration_ms"`
	ReadCount      int64  `json:"read_count,omitempty"`
	AvgReadSize    int    `json:"avg_read_size,omitempty"`
	MinReadSize    int    `json:"min_read_size,omitempty"`
	MaxReadSize    int    `json:"max_read_size,omitempty"`
	FirstReadSize  int    `json:"first_read_size,omitempty"`
	LastReadSize   int    `json:"last_read_size,omitempty"`
	AvgReadDeltaMS int64  `json:"avg_inter_read_ms,omitempty"`
	MinReadDeltaMS int64  `json:"min_inter_read_ms,omitempty"`
	MaxReadDeltaMS int64  `json:"max_inter_read_ms,omitempty"`
	Error          string `json:"error,omitempty"`
	Time           string `json:"time"`
}

type EOFEvent struct {
	Event  string `json:"event"`
	ConnID string `json:"conn_id"`
	Time   string `json:"time"`
}

type ErrorEvent struct {
	Event  string `json:"event"`
	ConnID string `json:"conn_id"`
	Error  string `json:"error"`
	Time   string `json:"time"`
}

type StartEvent struct {
	Event    string `json:"event"`
	Bind     string `json:"bind"`
	Verbose  bool   `json:"verbose"`
	DumpMode string `json:"dump_mode"`
}

type ShutdownEvent struct {
	Event  string `json:"event"`
	Status string `json:"status"`
}

type AcceptErrorEvent struct {
	Event string `json:"event"`
	Error string `json:"error"`
}

type ListenerErrorEvent struct {
	Event string `json:"event"`
	Error string `json:"error"`
}

type UnexpectedConnTypeEvent struct {
	Event string `json:"event"`
	Type  string `json:"type"`
}

type ProtocolEvent struct {
	Event    string `json:"event"`
	ConnID   string `json:"conn_id"`
	Protocol string `json:"protocol"`
	Time     string `json:"time"`
}

type StatsEvent struct {
	Event             string `json:"event"`
	ActiveConnections int64  `json:"active_connections"`
	TotalConnections  uint64 `json:"total_connections"`
	BytesRead         uint64 `json:"bytes_read"`
	BytesWritten      uint64 `json:"bytes_written"`
	Goroutines        int    `json:"goroutines"`
}
