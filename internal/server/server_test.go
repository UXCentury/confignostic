package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

type observedListener struct {
	net.Listener
	closed chan struct{}
	once   sync.Once
}

func (l *observedListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() { close(l.closed) })
	return err
}

func listen(t *testing.T) *observedListener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return &observedListener{Listener: ln, closed: make(chan struct{})}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server lifecycle event")
		var zero T
		return zero
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestServeDrainsInFlightRequest(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
			_, _ = io.WriteString(w, "drained")
		case <-r.Context().Done():
		}
	})
	stopped := make(chan error, 1)
	go func() { stopped <- Serve(ctx, ln, handler, 3*time.Second, testLogger()) }()
	type result struct {
		body string
		err  error
	}
	response := make(chan result, 1)
	go func() {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get("http://" + ln.Addr().String())
		if err != nil {
			response <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		response <- result{string(body), err}
	}()
	receive(t, entered)
	cancel()
	receive(t, ln.closed)
	select {
	case err := <-stopped:
		t.Fatalf("Serve returned before its active request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
		// Give an early-return regression time to become observable while the
		// request is deliberately blocked. Do not release it at listener close.
	}
	unblock()
	got := receive(t, response)
	if got.err != nil || got.body != "drained" {
		t.Fatalf("active response = %q, %v", got.body, got.err)
	}
	if err := receive(t, stopped); err != nil {
		t.Fatal(err)
	}
}

func TestServeClosesStalledRequestAtDeadline(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	entered, requestCanceled := make(chan struct{}), make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(requestCanceled)
	})
	stopped := make(chan error, 1)
	go func() { stopped <- Serve(ctx, ln, handler, 25*time.Millisecond, testLogger()) }()
	response := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get("http://" + ln.Addr().String())
		if resp != nil {
			_ = resp.Body.Close()
		}
		response <- err
	}()
	receive(t, entered)
	cancel()
	if err := receive(t, stopped); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected drain deadline, got %v", err)
	}
	receive(t, requestCanceled)
	if err := receive(t, response); err == nil {
		t.Fatal("stalled request unexpectedly succeeded")
	}
}

type failedListener struct {
	net.Listener
	err error
}

func (l failedListener) Accept() (net.Conn, error) { return nil, l.err }

func TestServeReportsListenerFailure(t *testing.T) {
	ln := listen(t)
	failure := errors.New("listener failed")
	err := Serve(context.Background(), failedListener{ln, failure}, http.NotFoundHandler(), time.Second, testLogger())
	if !errors.Is(err, failure) {
		t.Fatalf("expected listener failure, got %v", err)
	}
	receive(t, ln.closed)
}

func TestServeHandlesAlreadyCanceledContext(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, ln, http.NotFoundHandler(), time.Second, testLogger()); err != nil {
		t.Fatal(err)
	}
	receive(t, ln.closed)
}
