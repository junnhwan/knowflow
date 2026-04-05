package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter_Healthz(t *testing.T) {
	router := NewRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
