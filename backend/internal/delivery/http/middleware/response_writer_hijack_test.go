package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// testUpgrader mirrors the backend upgrade behavior closely enough to
// reproduce the http.Hijacker requirement.
var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func echoUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := testUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	mt, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	_ = conn.WriteMessage(mt, msg)
}

func dialAndEcho(t *testing.T, serverURL string) {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if resp == nil || resp.StatusCode != http.StatusSwitchingProtocols {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		t.Fatalf("expected 101 Switching Protocols, got %d", code)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}

	mt, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if mt != websocket.TextMessage || string(msg) != "ping" {
		t.Fatalf("unexpected echo: type=%d msg=%q", mt, string(msg))
	}
}

// TestObservabilityMiddlewareAllowsWebSocketUpgrade reproduces the reported
// bug: before the fix the interceptor hid http.Hijacker and gorilla/websocket
// returned "response does not implement http.Hijacker" (HTTP 500).
func TestObservabilityMiddlewareAllowsWebSocketUpgrade(t *testing.T) {
	srv := httptest.NewServer(ObservabilityMiddleware(http.HandlerFunc(echoUpgrade)))
	defer srv.Close()

	dialAndEcho(t, srv.URL)
}

// TestStatusCapturingResponseWriterAllowsWebSocketUpgrade covers the audit
// wrapper directly (the audit middleware itself only wraps mutations).
func TestStatusCapturingResponseWriterAllowsWebSocketUpgrade(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		echoUpgrade(rec, r)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	dialAndEcho(t, srv.URL)
}

// TestResponseWriterInterceptorHijackUnsupported verifies we return an error,
// not a panic or a fabricated connection, when the underlying writer cannot
// be hijacked.
func TestResponseWriterInterceptorHijackUnsupported(t *testing.T) {
	w := &responseWriterInterceptor{ResponseWriter: httptest.NewRecorder()}
	conn, brw, err := w.Hijack()
	if err == nil {
		t.Fatal("expected error when underlying writer does not implement http.Hijacker")
	}
	if conn != nil || brw != nil {
		t.Fatal("expected nil connection and read-writer on unsupported hijack")
	}
}

func TestStatusCapturingResponseWriterHijackUnsupported(t *testing.T) {
	w := &statusCapturingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	conn, brw, err := w.Hijack()
	if err == nil {
		t.Fatal("expected error when underlying writer does not implement http.Hijacker")
	}
	if conn != nil || brw != nil {
		t.Fatal("expected nil connection and read-writer on unsupported hijack")
	}
}

// flushCounter records how many times Flush was delegated.
type flushCounter struct {
	http.ResponseWriter
	flushes int
}

func (f *flushCounter) Flush() { f.flushes++ }

func TestResponseWriterInterceptorFlushDelegates(t *testing.T) {
	fc := &flushCounter{ResponseWriter: httptest.NewRecorder()}
	w := &responseWriterInterceptor{ResponseWriter: fc}
	w.Flush()
	if fc.flushes != 1 {
		t.Fatalf("expected underlying Flush to be called once, got %d", fc.flushes)
	}
}

func TestStatusCapturingResponseWriterFlushDelegates(t *testing.T) {
	fc := &flushCounter{ResponseWriter: httptest.NewRecorder()}
	w := &statusCapturingResponseWriter{ResponseWriter: fc}
	w.Flush()
	if fc.flushes != 1 {
		t.Fatalf("expected underlying Flush to be called once, got %d", fc.flushes)
	}
}

func TestInterceptorsUnwrapUnderlyingWriter(t *testing.T) {
	rec := httptest.NewRecorder()

	obs := &responseWriterInterceptor{ResponseWriter: rec}
	if obs.Unwrap() != rec {
		t.Fatal("responseWriterInterceptor.Unwrap did not return the underlying writer")
	}

	aud := &statusCapturingResponseWriter{ResponseWriter: rec}
	if aud.Unwrap() != rec {
		t.Fatal("statusCapturingResponseWriter.Unwrap did not return the underlying writer")
	}
}

// TestObservabilityMiddlewarePlainHTTPStillWorks guards against regressions in
// normal HTTP handling while the wrapper gains the new interfaces.
func TestObservabilityMiddlewarePlainHTTPStillWorks(t *testing.T) {
	srv := httptest.NewServer(ObservabilityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}
