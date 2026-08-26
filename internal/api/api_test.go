package api

import (
	"net/http/httptest"
	"path/filepath"
	"production43/internal/service"
	"production43/internal/store"
	"strings"
	"testing"
)

func TestHTTPFlow(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	handler := NewHandler(service.New(st, nil), 1<<20).Routes()
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
