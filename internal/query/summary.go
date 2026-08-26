package query

import (
	"production43/internal/domain"
	"time"
)

type Dashboard struct {
	GeneratedAt time.Time            `json:"generated_at"`
	Total       int                  `json:"total"`
	Lines       []domain.LineSummary `json:"lines"`
	Statuses    []domain.StatusCount `json:"statuses"`
	Critical    []domain.Record      `json:"critical"`
}

func BuildDashboard(records []domain.Record, now time.Time) Dashboard {
	critical := make([]domain.Record, 0)
	for _, record := range records {
		if record.Severity == domain.SeverityCritical && record.Status != domain.StatusResolved && record.Status != domain.StatusArchived {
			critical = append(critical, record)
		}
	}
	domain.SortRecords(critical, true)
	return Dashboard{GeneratedAt: now, Total: len(records), Lines: domain.SummarizeLines(records), Statuses: domain.CountStatuses(records), Critical: critical}
}

func Health(records []domain.Record) string {
	for _, record := range records {
		if record.Severity == domain.SeverityCritical && record.Status != domain.StatusResolved && record.Status != domain.StatusArchived {
			return "degraded"
		}
	}
	return "ok"
}
