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

// Exercise real WebSocket writes, production reconnection, and RPC retry logic.
// The server authenticates normally, then drops the first RPC's response.
func droppedResponseClient(t *testing.T, mode string) (*Client, *atomic.Int32) {
	t.Helper()
	atomicMethod := strings.HasPrefix(mode, "atomic-")
	mode = strings.TrimPrefix(mode, "atomic-")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx := r.Context()
		var auth map[string]string
		if err := wsjson.Read(ctx, conn, &auth); err != nil {
			return
		}
		if err := wsjson.Write(ctx, conn, map[string]bool{"authenticated": true}); err != nil {
			return
		}
		for {
			var request Request
			if err := wsjson.Read(ctx, conn, &request); err != nil {
				return
			}
			if atomicMethod {
				if !strings.HasSuffix(request.Method, "_atomic") {
					t.Errorf("atomic call used legacy method %s", request.Method)
				}
				params, ok := request.Params.(map[string]interface{})
				if !ok {
					t.Error("atomic params are not an object")
				} else if props, ok := params["initial_properties"].(map[string]interface{}); !ok || props == nil {
					t.Error("atomic properties must be an object, not null/omitted")
				}
			}
			if calls.Add(1) == 1 {
				if mode == "api-error" || mode == "method-not-found" {
					apiError := &Error{Code: -32602, Message: "invalid parameter"}
					if mode == "method-not-found" {
						apiError = &Error{Code: -32601, Message: "method not found"}
					}
					if err := wsjson.Write(ctx, conn, map[string]any{"id": request.ID, "error": apiError}); err != nil {
						return
					}
					continue
				}
				if mode == "invalid-result" {
					if err := wsjson.Write(ctx, conn, map[string]any{"id": request.ID, "result": "not a boolean"}); err != nil {
						return
					}
					continue
				}
				if mode == "stall" {
					// Read until the client closes, without replying to the RPC.
					_, _, _ = conn.Read(ctx)
				}
				return
			}
			if err := wsjson.Write(ctx, conn, map[string]any{"id": request.ID, "result": true}); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	client := &Client{
		url: "ws" + strings.TrimPrefix(server.URL, "http"), apiKey: "fixture-token",
		pending: make(map[string]chan *Response), closeCh: make(chan struct{}),
		metrics: noopMetrics{}, retryInterval: time.Millisecond, maxRetries: 1,
	}
	if err := client.connect(); err != nil {
		t.Fatal(err)
	}
	if err := client.authenticate(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); client.readLoop() }()
	t.Cleanup(func() {
		client.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("read loop did not stop")
		}
	})
	return client, &calls
}

func TestMutationsAreNotReplayedAfterLostResponse(t *testing.T) {
	for _, method := range []string{"subvolume.create", "snapshot.clone", "share.nvmeof.create", "share.iscsi.add_lun", "subvolume.delete", "unknown.update"} {
		t.Run(method, func(t *testing.T) {
			client, calls := droppedResponseClient(t, "drop")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := client.Call(ctx, method, nil, nil)
			if !errors.Is(err, ErrUncertainOutcome) || !errors.Is(err, ErrConnectionClosed) {
				t.Fatalf("lost mutation response: %v", err)
			}
			if calls.Load() != 1 {
				t.Fatalf("server received mutation %d times", calls.Load())
			}
		})
	}
}

func TestReadOnlyRPCRetriesAfterLostResponse(t *testing.T) {
	client, calls := droppedResponseClient(t, "drop")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result bool
	if err := client.Call(ctx, "fs.get", nil, &result); err != nil {
		t.Fatal(err)
	}
	if !result || calls.Load() != 2 {
		t.Fatalf("read-only retry result=%v, calls=%d", result, calls.Load())
	}
}

func TestMutationTimeoutRetainsOutcomeAndContextErrors(t *testing.T) {
	client, calls := droppedResponseClient(t, "stall")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := client.Call(ctx, "subvolume.create", nil, nil)
	if !errors.Is(err, ErrUncertainOutcome) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("mutation timeout: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("server received mutation %d times", calls.Load())
	}
}

func TestUnsentRequestDoesNotHaveUncertainOutcome(t *testing.T) {
	client := &Client{metrics: noopMetrics{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Call(ctx, "subvolume.create", nil, nil); !errors.Is(err, context.Canceled) || errors.Is(err, ErrUncertainOutcome) {
		t.Fatalf("canceled before send: %v", err)
	}
	if err := client.callOnce(context.Background(), "subvolume.create", nil, nil); !errors.Is(err, ErrConnectionClosed) || errors.Is(err, ErrUncertainOutcome) {
		t.Fatalf("no connection before send: %v", err)
	}
}

func TestRPCReplayAllowlist(t *testing.T) {
	for _, method := range []string{"fs.get", "subvolume.list_all", "subvolume.find_by_property", "snapshot.list", "share.nfs.list", "share.smb.get", "share.iscsi.list", "share.nvmeof.list"} {
		if !readOnlyRPC(method) {
			t.Errorf("read-only method %s cannot retry", method)
		}
	}
	for _, method := range []string{"subvolume.clone", "subvolume.resize", "subvolume.set_properties", "share.iscsi.add_acl", "system.reboot", "unknown.get"} {
		if readOnlyRPC(method) {
			t.Errorf("unreviewed method %s can replay", method)
		}
	}
}

func TestDefinitiveAPIErrorIsNotUncertain(t *testing.T) {
	client, calls := droppedResponseClient(t, "api-error")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := client.Call(ctx, "subvolume.create", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Code != -32602 || errors.Is(err, ErrUncertainOutcome) {
		t.Fatalf("definitive API error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("API error was retried %d times", calls.Load())
	}
}

func TestUndecodableMutationResultIsNotReplayed(t *testing.T) {
	client, calls := droppedResponseClient(t, "invalid-result")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result bool
	err := client.Call(ctx, "subvolume.create", nil, &result)
	if !errors.Is(err, ErrUncertainOutcome) || calls.Load() != 1 {
		t.Fatalf("undecodable result: %v, calls=%d", err, calls.Load())
	}
}
