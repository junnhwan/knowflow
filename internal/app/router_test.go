package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestNewRouter_PlaygroundPage(t *testing.T) {
	router := NewRouter(&App{})

	req := httptest.NewRequest(http.MethodGet, "/playground", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "KnowFlow Playground") {
		t.Fatalf("expected playground html to be returned")
	}
}

func TestNewRouter_PlaygroundAssets(t *testing.T) {
	router := NewRouter(&App{})

	req := httptest.NewRequest(http.MethodGet, "/playground/assets/playground.css", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "--bg") {
		t.Fatalf("expected playground css to be returned")
	}
}

func TestNewRouter_PlaygroundScript(t *testing.T) {
	router := NewRouter(&App{})

	req := httptest.NewRequest(http.MethodGet, "/playground/assets/playground.js", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "const state") {
		t.Fatalf("expected playground js to be returned")
	}
}
