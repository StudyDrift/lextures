// Package clamav implements the clamd INSTREAM scanning protocol (plan 8.6).
package clamav

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	cmdInstream     = "zINSTREAM"
	chunkSize       = 64 * 1024
	eicarTestString = "EICAR-STANDARD-ANTIVIRUS-TEST-FILE"
)

// ScanResult is the outcome of scanning a byte stream.
type ScanResult struct {
	Clean     bool
	VirusName string
}

// Client scans streams via clamd INSTREAM or a test stub.
type Client struct {
	Addr   string // host:port, e.g. localhost:3310
	Stub   bool   // when true, detect EICAR in-stream without clamd
	DialTO time.Duration
}

// NewClient returns a Client with defaults.
func NewClient(addr string, stub bool) *Client {
	if addr == "" {
		addr = "localhost:3310"
	}
	return &Client{Addr: addr, Stub: stub, DialTO: 30 * time.Second}
}

// ScanStream scans r and returns whether the content is clean.
func (c *Client) ScanStream(ctx context.Context, r io.Reader) (ScanResult, error) {
	if c.Stub {
		return scanStub(r)
	}
	return c.scanInstream(ctx, r)
}

func scanStub(r io.Reader) (ScanResult, error) {
	data, err := io.ReadAll(io.LimitReader(r, 10<<20))
	if err != nil {
		return ScanResult{}, err
	}
	if strings.Contains(string(data), eicarTestString) {
		return ScanResult{Clean: false, VirusName: "Eicar-Signature"}, nil
	}
	return ScanResult{Clean: true}, nil
}

func (c *Client) scanInstream(ctx context.Context, r io.Reader) (ScanResult, error) {
	dialer := &net.Dialer{Timeout: c.DialTO}
	conn, err := dialer.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return ScanResult{}, fmt.Errorf("clamav: dial %s: %w", c.Addr, err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(c.DialTO)); err != nil {
		return ScanResult{}, err
	}

	if _, err := conn.Write([]byte(cmdInstream)); err != nil {
		return ScanResult{}, fmt.Errorf("clamav: write command: %w", err)
	}
	if _, err := conn.Write([]byte{0}); err != nil {
		return ScanResult{}, fmt.Errorf("clamav: write nul: %w", err)
	}

	buf := make([]byte, chunkSize)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			var lenBuf [4]byte
			binary.BigEndian.PutUint32(lenBuf[:], uint32(n))
			if _, err := conn.Write(lenBuf[:]); err != nil {
				return ScanResult{}, fmt.Errorf("clamav: write chunk len: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return ScanResult{}, fmt.Errorf("clamav: write chunk: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ScanResult{}, readErr
		}
	}
	var zero [4]byte
	if _, err := conn.Write(zero[:]); err != nil {
		return ScanResult{}, fmt.Errorf("clamav: write end: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return ScanResult{}, fmt.Errorf("clamav: read response: %w", err)
	}
	return parseResponse(strings.TrimSpace(line))
}

func parseResponse(line string) (ScanResult, error) {
	if line == "" {
		return ScanResult{}, errors.New("clamav: empty response")
	}
	// "stream: OK" or "stream: Eicar-Signature FOUND"
	if strings.HasSuffix(line, " OK") {
		return ScanResult{Clean: true}, nil
	}
	if idx := strings.Index(line, " FOUND"); idx >= 0 {
		parts := strings.SplitN(line, ": ", 2)
		name := strings.TrimSuffix(parts[len(parts)-1], " FOUND")
		return ScanResult{Clean: false, VirusName: strings.TrimSpace(name)}, nil
	}
	return ScanResult{}, fmt.Errorf("clamav: unexpected response: %q", line)
}

// QuarantineKey returns the quarantine prefix for an object key.
func QuarantineKey(objectKey string) string {
	key := strings.TrimPrefix(objectKey, "/")
	if strings.HasPrefix(key, "quarantine/") {
		return key
	}
	return "quarantine/" + key
}

// ReleaseKey strips the quarantine prefix from a key.
func ReleaseKey(quarantineKey string) string {
	return strings.TrimPrefix(quarantineKey, "quarantine/")
}
