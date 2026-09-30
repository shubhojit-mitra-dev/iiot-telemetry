package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/service"
)

func setupTestApp(t *testing.T) (*Handler, *repository.MemoryRepository, *service.IngestionService, *service.Hub, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	repo := repository.NewMemoryRepository()
	hub := service.NewHub()
	go hub.Run(ctx)

	svc := service.NewIngestionService(repo, hub, 2, 100)
	svc.Start(ctx)

	handler := NewHandler(svc, repo, hub)

	cleanup := func() {
		cancel()
		svc.Stop()
		_ = repo.Close()
	}

	return handler, repo, svc, hub, cleanup
}

func TestHandleIngest(t *testing.T) {
	handler, _, _, _, cleanup := setupTestApp(t)
	defer cleanup()

	tests := []struct {
		name         string
		method       string
		body         string
		expectedCode int
	}{
		{
			name:         "successful telemetry post",
			method:       http.MethodPost,
			body:         `{"device_id":"TURB-001","timestamp":1700000000000,"temperature":84.2,"vibration":5.8,"rpm":3190}`,
			expectedCode: http.StatusAccepted,
		},
		{
			name:         "invalid method get",
			method:       http.MethodGet,
			body:         "",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "malformed json",
			method:       http.MethodPost,
			body:         `{"device_id": "TURB-001", "temp`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation error missing device id",
			method:       http.MethodPost,
			body:         `{"device_id":"","timestamp":1700000000000,"temperature":84.2,"vibration":5.8,"rpm":3190}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "validation error zero timestamp",
			method:       http.MethodPost,
			body:         `{"device_id":"TURB-001","timestamp":0,"temperature":84.2,"vibration":5.8,"rpm":3190}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/telemetry", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.HandleIngest(rr, req)

			if rr.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleIngest_QueueSaturated(t *testing.T) {
	repo := repository.NewMemoryRepository()
	defer repo.Close()

	// Service with queue size 1, workers NOT started so queue fills immediately
	svc := service.NewIngestionService(repo, nil, 1, 1)
	hub := service.NewHub()
	handler := NewHandler(svc, repo, hub)

	validBody := `{"device_id":"TURB-001","timestamp":1700000000000,"temperature":84.2,"vibration":5.8,"rpm":3190}`

	// Fill queue
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBufferString(validBody))
	rr1 := httptest.NewRecorder()
	handler.HandleIngest(rr1, req1)
	if rr1.Code != http.StatusAccepted {
		t.Fatalf("expected first ingest to succeed, got %d", rr1.Code)
	}

	// Next request should trigger 503 Service Unavailable
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewBufferString(validBody))
	rr2 := httptest.NewRecorder()
	handler.HandleIngest(rr2, req2)
	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected queue saturated 503, got %d", rr2.Code)
	}
}

func TestHandleGetDevices(t *testing.T) {
	handler, repo, _, _, cleanup := setupTestApp(t)
	defer cleanup()

	ctx := context.Background()
	_ = repo.SaveLatest(ctx, model.TelemetryPayload{
		DeviceID:    "COMP-001",
		Timestamp:   1700000000000,
		Temperature: 76.5,
		Vibration:   5.4,
		RPM:         2800,
	})

	// 1. Valid GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetDevices(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var devices map[string]model.TelemetryPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &devices); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(devices) != 1 || devices["COMP-001"].Temperature != 76.5 {
		t.Fatalf("unexpected devices payload: %+v", devices)
	}

	// 2. Invalid Method POST
	reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/devices", nil)
	rrPost := httptest.NewRecorder()
	handler.HandleGetDevices(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rrPost.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	handler, _, _, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	handler.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %v", resp["status"])
	}
}

func TestServerE2ERouting(t *testing.T) {
	handler, _, _, _, cleanup := setupTestApp(t)
	defer cleanup()

	server := httptest.NewServer(NewServer(handler))
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// 1. Health Probe
	resp, err := client.Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("failed to call health endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from health endpoint, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. Telemetry Ingest
	ingestBody := bytes.NewBufferString(`{"device_id":"WELD-001","timestamp":1700000000000,"temperature":92.0,"vibration":7.8,"rpm":2400}`)
	postResp, err := client.Post(server.URL+"/api/v1/telemetry", "application/json", ingestBody)
	if err != nil {
		t.Fatalf("failed to post telemetry: %v", err)
	}
	if postResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted from ingest endpoint, got %d", postResp.StatusCode)
	}
	_ = postResp.Body.Close()

	// 3. Query Devices
	time.Sleep(50 * time.Millisecond)
	getResp, err := client.Get(server.URL + "/api/v1/devices")
	if err != nil {
		t.Fatalf("failed to get devices: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from devices endpoint, got %d", getResp.StatusCode)
	}
	_ = getResp.Body.Close()
}
