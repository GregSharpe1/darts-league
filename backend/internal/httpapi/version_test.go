package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionEndpointReturnsConfiguredVersion(t *testing.T) {
	t.Parallel()

	handler := NewVersionHandler("v0.0.6")
	recorder := hitEndpoint(t, handler.handleVersion, httptest.NewRequest(http.MethodGet, "/api/version", nil), http.StatusOK)

	var response struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid version response, got %v", err)
	}

	if response.Version != "v0.0.6" {
		t.Fatalf("expected configured version, got %q", response.Version)
	}
}
