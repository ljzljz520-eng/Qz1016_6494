package store

import (
	"path/filepath"
	"production43/internal/domain"
	"testing"
	"time"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	record := domain.Record{ID: "persist-1", Line: "43", Station: "weld", Severity: domain.SeverityCritical, Status: domain.StatusInvestigating, Summary: "sensor", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := st.SaveRecord(record); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveEvent(NewEvent(record, "u1", "test", "payload", domain.StatusNew, record.Status, 1, time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.GetRecord(record.ID)
	if err != nil || got.Status != domain.StatusInvestigating {
		t.Fatalf("%v %+v", err, got)
	}
	events, err := st.EventsForRecord(record.ID)
	if err != nil || len(events) != 1 {
		t.Fatalf("%v %d", err, len(events))
	}
}
