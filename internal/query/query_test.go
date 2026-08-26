package query

import (
	"production43/internal/domain"
	"testing"
	"time"
)

func TestQueryDashboard(t *testing.T) {
	records := []domain.Record{{ID: "a", Line: "43", Severity: domain.SeverityCritical, Status: domain.StatusNew, CreatedAt: time.Now(), UpdatedAt: time.Now()}, {ID: "b", Line: "44", Severity: domain.SeverityInfo, Status: domain.StatusResolved, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	dashboard := BuildDashboard(records, time.Now())
	if dashboard.Total != 2 || len(dashboard.Critical) != 1 {
		t.Fatalf("%+v", dashboard)
	}
}
