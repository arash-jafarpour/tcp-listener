# tcp-listener

Zero-dependency Go TCP sniffer/debugger that logs structured JSON events for every connection lifecycle.

Useful for debugging network protocols, monitoring incoming connections, simple protocol fingerprinting, and observing connection patterns without external tools like `tcpdump` or `ngrep`.

## Features

- **JSON-structured logging** — machine-parseable log output for every connection event
- **Connection lifecycle tracking** — open, data, protocol, EOF, and close events with timestamps
- **L7 protocol detection** — automatically identifies TLS, HTTP, and SSH on the first read
- **MTU/MSS discovery** — logs interface MTU and socket MSS per connection
- **Payload dump modes** — optional hex or hexdump payload logging for deep inspection
- **Inter-read timing and read size statistics** — per-connection min/max/avg/first/last read sizes and inter-read delays
- **Periodic server stats** — configurable interval for goroutine count, active connections, and throughput
- **Graceful shutdown** — drains active connections on SIGINT/SIGTERM before exiting
- **Zero external dependencies** — pure Go standard library
- **CGO-free build** — optional static binary with `CGO_ENABLED=0`

## Quick Start

```bash
go build -o tcp-listener
./tcp-listener
```

Listens on `0.0.0.0:9000` by default. Send traffic to it and watch the JSON log output.

## Usage

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
./tcp-listener --verbose --dump hex
```

### Multi-line hexdump view

```bash
./tcp-listener --verbose --dump hexdump
```

### Custom bind address and port

```bash
./tcp-listener --bind 127.0.0.1 --port 8080
```

### Periodic stats every 30 seconds

```bash
./tcp-listener --stats-interval 30s
```

## Makefile

| Target             | Description                                   |
| ------------------ | --------------------------------------------- |
| `make build`       | Build the binary                              |
| `make build-nocgo` | Build with `CGO_ENABLED=0` (static binary)    |
| `make run`         | Build and run with defaults                   |
| `make run-verbose` | Build and run with `--verbose`                |
| `make run-hex`     | Build and run with `--verbose --dump hex`     |
| `make run-hexdump` | Build and run with `--verbose --dump hexdump` |

## Event Reference

All log output is newline-delimited JSON (NDJSON).

Each event contains at minimum:

- `event`
- `time`

### `start`

Server started and begins accepting connections.

```json
{
    "event": "start",
    "bind": "0.0.0.0:9000",
    "verbose": false,
    "dump_mode": "none"
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
    "socket_mss": 1460
}
```

### `protocol`

L7 protocol detected from the first data chunk.

```json
{
    "event": "protocol",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "protocol": "http"
}
```

Detected protocols:

- `tls`
- `http`
- `ssh`
- `unknown`

### `data`

A read from the connection (only emitted with `--verbose`).

```json
{
    "event": "data",
    "conn_id": "tc-17a1b2c3d4e-0001",
    "dir": "read",
    "bytes": 256,
    "delta_ms": 12
}
```

- `delta_ms` — milliseconds since the previous read on this connection (omitted on the first read)

When using `--dump hex` or `--dump hexdump`, a `payload` field is included.

### `eof`

Remote side closed the connection.

```json
{
    "event": "eof",
    "conn_id": "tc-17a1b2c3d4e-0001"
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
    "max_inter_read_ms": 5000
}
```

Field descriptions:

- `read_count` — total number of reads performed on this connection
- `avg_read_size` — average bytes per read
- `min_read_size` / `max_read_size` — smallest and largest read size
- `first_read_size` / `last_read_size` — size of the first and last read
- `avg_inter_read_ms` — average milliseconds between reads (omitted if fewer than 2 reads)
- `min_inter_read_ms` / `max_inter_read_ms` — minimum and maximum inter-read delay

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

## Project Structure

| File          | Purpose                                               |
| ------------- | ----------------------------------------------------- |
| `main.go`     | Entry point, signal handling, and accept loop         |
| `config.go`   | CLI flag parsing                                      |
| `conn.go`     | Per-connection read loop, MTU probing, and statistics |
| `connid.go`   | Unique connection ID generator                        |
| `events.go`   | JSON event type definitions                           |
| `logger.go`   | JSON-structured log output                            |
| `protocol.go` | L7 protocol detection logic                           |
| `stats.go`    | Global server statistics counters                     |
| `Makefile`    | Build and run convenience targets                     |
