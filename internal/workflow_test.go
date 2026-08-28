package internal_test

import (
	"path/filepath"
	"production43/internal/domain"
	"production43/internal/service"
	"production43/internal/store"
	"testing"
)

func TestWorkflowTwo(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "workflow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st, nil)
	operator := domain.User{ID: "op", Name: "Op", Role: "operator", Lines: []string{"43"}, Active: true}
	supervisor := domain.User{ID: "sup", Name: "Sup", Role: "supervisor", Lines: []string{"43"}, Active: true}
	record, err := svc.Register(domain.Record{ID: "wf2", Line: "43", Station: "pack", Severity: domain.SeverityInfo, Summary: "box"}, operator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Process(record.ID, operator, domain.ActionAcknowledge, "ack", record.Version); err != nil {
		t.Fatal(err)
	}
	current, err := svc.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, err = svc.Process(record.ID, operator, domain.ActionInvestigate, "reviewing", current.Version)
	if err != nil {
		t.Fatal(err)
	}
	_, updated, err := svc.Review(record.ID, supervisor, "approve", "verified")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusArchived || current.Version >= updated.Version {
		t.Fatalf("%+v %+v", current, updated)
	}
}

func TestWorkflowThree(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "workflow3.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st, nil)
	actor := domain.User{ID: "op3", Name: "Op", Role: "operator", Lines: []string{"43"}, Active: true}
	record, err := svc.Register(domain.Record{ID: "wf3", Line: "43", Station: "cut", Severity: domain.SeverityWarning, Summary: "blade"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Process(record.ID, actor, domain.ActionAcknowledge, "seen", record.Version)
	if err != nil || got.Status != domain.StatusAcknowledged {
		t.Fatalf("%v %+v", err, got)
	}
	timeline, err := svc.Timeline(record.ID)
	if err != nil || len(timeline.Events) != 2 {
		t.Fatalf("%v %+v", err, timeline)
	}
}

func TestRecordFlow43(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "regression.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st, nil)
	actor := domain.User{ID: "op43", Name: "Operator", Role: "operator", Lines: []string{"43"}, Active: true}
	record, err := svc.Register(domain.Record{ID: "record-43", Line: "43", Station: "press", Severity: domain.SeverityCritical, Summary: "line alarm"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	current, err := svc.Process(record.ID, actor, domain.ActionAcknowledge, "acknowledged", record.Version)
	if err != nil {
		t.Fatal(err)
	}
	current, err = svc.Process(record.ID, domain.User{ID: "op43", Name: "Operator", Role: "operator", Lines: []string{"43"}, Active: true}, domain.ActionInvestigate, "investigating", current.Version)
	if err != nil {
		t.Fatal(err)
	}
	current, err = svc.Process(record.ID, domain.User{ID: "sup43", Name: "Supervisor", Role: "supervisor", Lines: []string{"43"}, Active: true}, domain.ActionResolve, "resolved", current.Version)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != domain.StatusResolved {
		t.Fatalf("latest status is %s", current.Status)
	}
}
