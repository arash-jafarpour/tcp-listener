# tcp-listener

Zero-dependency Go TCP sniffer/debugger that logs structured JSON events for every connection lifecycle.

Useful for debugging network protocols, monitoring incoming connections, simple protocol fingerprinting, and observing connection patterns without external tools like `tcpdump` or `ngrep`.

## Features

- **Subcommand-based CLI** — `tcp-listener listen` and `tcp-listener analyze <logfile>`
- **JSON-structured logging** — machine-parseable NDJSON log output for every connection event
- **Connection lifecycle tracking** — open, data, protocol, EOF, error, and close events with nanosecond timestamps
- **L7 protocol detection** — automatically identifies TLS, HTTP, and SSH on the first read
- **MTU/MSS discovery** — logs interface MTU and socket MSS per connection via syscalls
- **Payload dump modes** — optional hex or hexdump payload logging for deep inspection
- **Inter-read timing and read size statistics** — per-connection min/max/avg/first/last read sizes and inter-read delays
- **Periodic server stats** — configurable interval for goroutine count, active connections, and throughput
- **Graceful shutdown** — drains active connections on SIGINT/SIGTERM before exiting
- **Built-in log analyzer** — post-hoc analysis with protocol distribution, close reasons, and duration percentiles
- **Zero external dependencies** — pure Go standard library

## Quick Start

```bash
go build -o tcp-listener
./tcp-listener listen
```

Listens on `0.0.0.0:9000` by default. Send traffic to it and watch the JSON log output.

## Usage

```text
tcp-listener listen [flags]
tcp-listener analyze <logfile>
```

### Flags (`listen`)

```text
--bind string          address to bind (default "0.0.0.0")
--port int             listen port (default 9000)
--verbose              log every read/write event (default false)
--dump string          payload dump mode: none|hex|hexdump (default "none")
--stats-interval       periodic stats logging interval (0 disables)
```

## Examples

### Verbose logging with hex payload dump

```bash
tcp-listener listen --verbose --dump hex
```

### Multi-line hexdump view

```bash
tcp-listener listen --verbose --dump hexdump
```

### Custom bind address and port

```bash
tcp-listener listen --bind 127.0.0.1 --port 8080
```

### Periodic stats every 30 seconds

```bash
tcp-listener listen --stats-interval 30s
```

### Analyze a session log

```bash
tcp-listener analyze logs/2026-06-09_150645.log
```

## Makefile

| Target             | Description                                   |
| ------------------ | --------------------------------------------- |
| `make build`       | Build the binary                              |
| `make build-nocgo` | Build with `CGO_ENABLED=0`                    |
| `make run`         | Build and run with defaults                   |
| `make run-verbose` | Build and run with `--verbose`                |
| `make run-hex`     | Build and run with `--verbose --dump hex`     |
| `make run-hexdump` | Build and run with `--verbose --dump hexdump` |

## Event Reference

All log output is newline-delimited JSON (NDJSON).

Each event contains at minimum:

- `event`
- `time` (RFC3339Nano format)

### `start`

Server started and begins accepting connections.

```json
{
    "event": "start",
    "bind": "0.0.0.0:9000",
    "verbose": false,
    "dump_mode": "none",
    "session_file": "logs/2026-06-09_150405.log"
}
```

### `open`

A new TCP connection was accepted.

```json
{
    "event": "open",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "src": "192.168.1.5:54321",
    "dst": "0.0.0.0:9000",
    "iface_mtu": 1500,
    "socket_mss": 1460,
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

### `protocol`

L7 protocol detected from the first data chunk.

```json
{
    "event": "protocol",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "protocol": "http",
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

Detected protocols:

- `tls`
- `http`
- `ssh`
- `unknown`

### `data`

A read from the connection (only emitted with `--verbose` or a dump mode).

```json
{
    "event": "data",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "dir": "read",
    "bytes": 256,
    "delta_ms": 12,
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

- `delta_ms` — milliseconds since the previous read on this connection (omitted on the first read)

When using `--dump hex` or `--dump hexdump`, a `hex` or `hexdump` field is included respectively.

### `eof`

Remote side closed the connection cleanly.

```json
{
    "event": "eof",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

### `error`

Unexpected read error occurred on the connection.

```json
{
    "event": "error",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "error": "read: connection reset by peer",
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

### `close`

Connection fully closed with transfer statistics.

```json
{
    "event": "close",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "src": "192.168.1.5:54321",
    "dst": "0.0.0.0:9000",
    "bytes_read": 1024,
    "bytes_written": 0,
    "duration_ms": 5000,
    "read_count": 42,
    "avg_read_size": 24,
    "min_read_size": 1,
    "max_read_size": 1460,
    "first_read_size": 512,
    "last_read_size": 128,
    "avg_inter_read_ms": 119,
    "min_inter_read_ms": 2,
    "max_inter_read_ms": 5000,
    "close_reason": "eof",
    "time": "2026-06-09T15:04:05.123456789Z"
}
```

#### Field Descriptions

- `read_count` — total number of reads performed on this connection
- `avg_read_size` — average bytes per read
- `min_read_size` / `max_read_size` — smallest and largest read size
- `first_read_size` / `last_read_size` — size of the first and last read
- `avg_inter_read_ms` — average milliseconds between reads (omitted if fewer than 2 reads)
- `min_inter_read_ms` / `max_inter_read_ms` — minimum and maximum inter-read delay
- `error` — raw error string for abnormal closes (omitted on clean EOF)
- `close_reason` — standardized close category

#### Close Reasons

| Reason                     | Description                  |
| -------------------------- | ---------------------------- |
| `eof`                      | Remote side closed cleanly   |
| `timeout`                  | I/O timeout                  |
| `connection_reset_by_peer` | TCP RST received             |
| `broken_pipe`              | Write to a closed connection |
| _(empty)_                  | Other or unexpected error    |

### `stats`

Periodic server statistics (emitted at `--stats-interval`).

```json
{
    "event": "stats",
    "active_connections": 3,
    "total_connections": 42,
    "bytes_read": 1048576,
    "bytes_written": 0,
    "goroutines": 5
}
```

### `shutdown`

Server shutdown lifecycle events.

```json
{
    "event": "shutdown",
    "status": "draining"
}
```

```json
{
    "event": "shutdown",
    "status": "done"
}
```

### `accept_error`

Error accepting a new connection.

```json
{
    "event": "accept_error",
    "error": "accept: too many open files"
}
```

### `listener_error`

Error closing the listener during shutdown.

```json
{
    "event": "listener_error",
    "error": "close: use of closed network connection"
}
```

### `unexpected_conn_type`

Received a non-TCP connection from the listener.

```json
{
    "event": "unexpected_conn_type",
    "type": "*net.UDPConn"
}
```

## Log Analyzer

The `analyze` subcommand processes a session log file and prints a summary report.

### Example Output

```text
Lines Processed: 1520

Summary
-------
Connections: 12
Bytes Read: 87432

Protocols
-------
tls                      8
http                     3
unknown                  1

Close Reasons
-------
eof                      10
connection_reset_by_peer 2

MSS Distribution
-------
1460                     10
1440                     2

Top Payload Sizes
-----------------
1024                     45
512                      32
1460                     28
256                      18

Connection Durations
--------------------
Min: 12 ms
Avg: 4502 ms
P50: 3200 ms
P95: 15000 ms
Max: 60000 ms
```
