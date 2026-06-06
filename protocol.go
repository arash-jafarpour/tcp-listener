package main

import "bytes"

func DetectProtocol(data []byte) string {
	if len(data) < 3 {
		return "unknown"
	}

	if len(data) >= 3 &&
		data[0] == 0x16 &&
		data[1] == 0x03 {
		return "tls"
	}

	if bytes.HasPrefix(data, []byte("GET ")) {
		return "http"
	}

	if bytes.HasPrefix(data, []byte("POST ")) {
		return "http"
	}

	if bytes.HasPrefix(data, []byte("SSH-")) {
		return "ssh"
	}

	return "unknown"
}
