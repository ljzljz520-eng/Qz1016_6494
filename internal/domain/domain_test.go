package domain

import (
	"testing"
	"time"
)

func TestValidateAndNormalize(t *testing.T) {
	record := NormalizeRecord(Record{ID: " a1 ", Line: " 43 ", Station: " S1 ", Severity: SeverityWarning, Status: StatusNew, Summary: " alert ", Labels: []string{"Hot", "hot", ""}})
	if err := ValidateRecord(record); err != nil {
		t.Fatal(err)
	}
	if record.ID != "a1" || len(record.Labels) != 1 || record.Labels[0] != "hot" {
		t.Fatalf("unexpected normalization: %+v", record)
	}
}

func TestTransitions(t *testing.T) {
	status, err := NextStatus(StatusNew, ActionAcknowledge)
	if err != nil || status != StatusAcknowledged {
		t.Fatalf("%v %s", err, status)
	}
	if _, err := NextStatus(StatusArchived, ActionResolve); err == nil {
		t.Fatal("expected invalid transition")
	}
}

func TestEscalation(t *testing.T) {
	now := time.Date(2025, 1, 1, 1, 0, 0, 0, time.UTC)
	record := Record{ID: "x1", Line: "43", Severity: SeverityCritical, Status: StatusNew, CreatedAt: now.Add(-10 * time.Minute)}
	if _, ok := EvaluateEscalation(record, now, DefaultEscalationPolicy()); !ok {
		t.Fatal("critical record should escalate")
	}
}
