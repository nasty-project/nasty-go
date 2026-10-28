package nastygo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// keepaliveClient connects a client with its read and ping loops to a server
// that authenticates and then runs serve on the connection.
func keepaliveClient(t *testing.T, retryInterval time.Duration, serve func(context.Context, *websocket.Conn)) (*Client, *atomic.Int32) {
	t.Helper()
	var accepts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		accepts.Add(1)
		ctx := r.Context()
		var auth map[string]string
		if err := wsjson.Read(ctx, conn, &auth); err != nil {
			return
		}
		if err := wsjson.Write(ctx, conn, map[string]bool{"authenticated": true}); err != nil {
			return
		}
		serve(ctx, conn)
	}))
	t.Cleanup(server.Close)
	client := &Client{
		url: "ws" + strings.TrimPrefix(server.URL, "http"), apiKey: "fixture-token",
		pending: make(map[string]chan *Response), closeCh: make(chan struct{}),
		metrics: noopMetrics{}, retryInterval: retryInterval, maxRetries: 1,
		pingInterval: 20 * time.Millisecond, pingTimeout: 20 * time.Millisecond,
	}
	if err := client.connect(); err != nil {
		t.Fatal(err)
	}
	if err := client.authenticate(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); client.readLoop() }()
	go client.pingLoop()
	t.Cleanup(func() {
		client.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("read loop did not stop")
		}
	})
	return client, &accepts
}

func TestSilentPeerIsReconnected(t *testing.T) {
	// The server stops reading after authentication, so it never answers pings.
	_, accepts := keepaliveClient(t, time.Millisecond, func(ctx context.Context, _ *websocket.Conn) {
		<-ctx.Done()
	})

	deadline := time.Now().Add(5 * time.Second)
	for accepts.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("client never reconnected to a silent peer (connections: %d)", accepts.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLostConnectionFailsPendingCallsBeforeBackoff(t *testing.T) {
	const backoff = 2 * time.Second
	client, _ := keepaliveClient(t, backoff, func(ctx context.Context, conn *websocket.Conn) {
		// Drop the connection as soon as a request arrives.
		var request Request
		_ = wsjson.Read(ctx, conn, &request)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	err := client.Call(ctx, "subvolume.create", nil, nil)
	if !errors.Is(err, ErrUncertainOutcome) {
		t.Fatalf("lost mutation response: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= backoff {
		t.Fatalf("pending call failed after %v, want before the %v reconnect backoff", elapsed, backoff)
	}
}
