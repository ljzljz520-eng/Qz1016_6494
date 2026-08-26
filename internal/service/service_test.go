package service

import (
	"path/filepath"
	"production43/internal/domain"
	"production43/internal/store"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }
func newService(t *testing.T) (*Service, domain.User) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	return New(st, fixedClock{now}), domain.User{ID: "op1", Name: "Operator", Role: "operator", Lines: []string{"43"}, Active: true}
}

func TestWorkflowOne(t *testing.T) {
	svc, actor := newService(t)
	record, err := svc.Register(domain.Record{ID: "flow-1", Line: "43", Station: "press", Severity: domain.SeverityWarning, Summary: "overheat"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != domain.StatusNew {
		t.Fatal(record.Status)
	}
	got, err := svc.Get(record.ID)
	if err != nil || got.ID != record.ID {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestProcessAcknowledgement(t *testing.T) {
	svc, actor := newService(t)
	record, err := svc.Register(domain.Record{ID: "flow-2", Line: "43", Station: "press", Severity: domain.SeverityInfo, Summary: "notice"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Process(record.ID, actor, domain.ActionAcknowledge, "seen", record.Version)
	if err != nil || got.Status != domain.StatusAcknowledged {
		t.Fatalf("%v %+v", err, got)
	}
}
