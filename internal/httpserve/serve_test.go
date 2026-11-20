package httpserve

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func TestServeRefusesOpenBind(t *testing.T) {
	err := Serve(context.Background(), "0.0.0.0:0", server(), "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "PENPOT_MCP_HTTP_TOKEN") {
		t.Fatalf("got %v", err)
	}
}

func TestServeStopsWithContext(t *testing.T) {
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, addr, server(), "", slog.New(slog.NewTextHandler(io.Discard, nil))) }()

	var resp *http.Response
	var err error
	for range 50 {
		resp, err = http.Get("http://" + addr + "/other") // #nosec G107 -- a loopback test server
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 off the MCP path", resp.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not stop")
	}
}

func TestServeReportsListenError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	err = Serve(context.Background(), l.Addr().String(), server(), "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("want an error when the port is taken")
	}
}
