package store

import (
	"path/filepath"
	"production43/internal/domain"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "alerts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	record := domain.Record{ID: "r1", Line: "43", Station: "press", Severity: domain.SeverityWarning, Status: domain.StatusNew, Summary: "jam", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := st.SaveRecord(record); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetRecord("r1")
	if err != nil || got.Summary != "jam" {
		t.Fatalf("%v %+v", err, got)
	}
}
