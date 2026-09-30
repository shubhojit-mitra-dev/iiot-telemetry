package api

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	called := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	cors := CORSMiddleware(dummyHandler)

	// Test 1: Standard GET request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	rr := httptest.NewRecorder()
	cors.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected origin *, got %s", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Test 2: Preflight OPTIONS request
	called = false
	optionsReq := httptest.NewRequest(http.MethodOptions, "/api/v1/telemetry", nil)
	optionsRR := httptest.NewRecorder()
	cors.ServeHTTP(optionsRR, optionsReq)

	if called {
		t.Fatal("preflight OPTIONS request should not execute downstream handler")
	}
	if optionsRR.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for preflight, got %d", optionsRR.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	logging := LoggingMiddleware(dummyHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", nil)
	rr := httptest.NewRecorder()
	logging.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated fatal crash in handler")
	})

	recovery := RecoveryMiddleware(panickingHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	// Assert that ServeHTTP does not crash the test process
	recovery.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", rr.Code)
	}
}

type mockHijackFlusher struct {
	http.ResponseWriter
	hijacked bool
	flushed  bool
}

func (m *mockHijackFlusher) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	c1, _ := net.Pipe()
	return c1, bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1)), nil
}

func (m *mockHijackFlusher) Flush() {
	m.flushed = true
}

func TestResponseWriter_HijackerAndFlusher(t *testing.T) {
	// Case 1: Underlying ResponseWriter does NOT implement Hijacker
	baseRec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: baseRec, statusCode: http.StatusOK}

	_, _, err := rw.Hijack()
	if err == nil {
		t.Fatal("expected error when underlying writer does not implement http.Hijacker")
	}

	// Unwrap check
	if rw.Unwrap() != baseRec {
		t.Fatal("expected Unwrap to return underlying recorder")
	}

	// Case 2: Underlying ResponseWriter implements Hijacker and Flusher
	mock := &mockHijackFlusher{ResponseWriter: baseRec}
	rwWithMock := &responseWriter{ResponseWriter: mock, statusCode: http.StatusOK}

	conn, bufrw, err := rwWithMock.Hijack()
	if err != nil {
		t.Fatalf("unexpected hijack error: %v", err)
	}
	if conn != nil {
		defer conn.Close()
	}
	if bufrw == nil || !mock.hijacked {
		t.Fatal("expected successful hijack on mock")
	}

	rwWithMock.Flush()
	if !mock.flushed {
		t.Fatal("expected Flush to be called on mock")
	}
}
