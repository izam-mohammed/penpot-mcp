// Package httpserve serves the MCP tools over streamable HTTP, for remote
// clients. It is off unless --http is given.
package httpserve

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MaxBody caps one request. Tool inputs are ids, text and local paths, never
// file contents, so 4 MB is far more than a request needs.
const MaxBody = 4 << 20

// Path is where the MCP endpoint is served.
const Path = "/mcp"

// Check refuses a bind that is not loopback unless a bearer token is set.
func Check(addr, token string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("--http wants host:port, such as 127.0.0.1:8787: %w", err)
	}
	if loopback(host) {
		return nil
	}
	if len(token) < 16 {
		return errors.New("--http on a non-loopback address needs PENPOT_MCP_HTTP_TOKEN (16 characters or more): " +
			"anyone who reaches the port could act as your Penpot account")
	}
	return nil
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Handler serves s at Path, statelessly, with the body cap and, when token
// is set, a bearer token check.
func Handler(s *mcp.Server, token string) http.Handler {
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{
		Stateless:           true,
		MaxRequestBodyBytes: MaxBody,
	})
	mux := http.NewServeMux()
	mux.HandleFunc(Path, func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
		h.ServeHTTP(w, r)
	})
	return mux
}

// Serve runs the HTTP server until ctx ends.
func Serve(ctx context.Context, addr string, s *mcp.Server, token string, log *slog.Logger) error {
	if err := Check(addr, token); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(s, token),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("serving MCP over HTTP", "addr", addr, "path", Path, "token", token != "")
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shut)
	}
}
