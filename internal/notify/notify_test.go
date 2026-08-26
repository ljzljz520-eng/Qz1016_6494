package notify

import (
	"path/filepath"
	"production43/internal/domain"
	"production43/internal/store"
	"testing"
	"time"
)

func TestNotifier(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "notify.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	n := New(st)
	record := domain.Record{ID: "n1", Line: "43", Severity: domain.SeverityCritical, Status: domain.StatusInvestigating, Station: "press", Summary: "hot"}
	users := []domain.User{{ID: "u1", Name: "Supervisor", Role: "supervisor", Email: "s@example.com", Lines: []string{"43"}, Active: true}}
	deliveries, err := n.Notify(record, users, time.Now())
	if err != nil || len(deliveries) != 2 {
		t.Fatalf("%v %d", err, len(deliveries))
	}
}
