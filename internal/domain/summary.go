package domain

import (
	"sort"
	"time"
)

type LineSummary struct {
	Line        string    `json:"line"`
	Open        int       `json:"open"`
	Critical    int       `json:"critical"`
	Resolved    int       `json:"resolved"`
	OldestOpen  time.Time `json:"oldest_open"`
	LastUpdated time.Time `json:"last_updated"`
}

type StatusCount struct {
	Status Status `json:"status"`
	Count  int    `json:"count"`
}

func SummarizeLines(records []Record) []LineSummary {
	byLine := make(map[string]*LineSummary)
	for _, record := range records {
		summary := byLine[record.Line]
		if summary == nil {
			summary = &LineSummary{Line: record.Line}
			byLine[record.Line] = summary
		}
		if record.Status == StatusResolved || record.Status == StatusArchived {
			summary.Resolved++
		} else {
			summary.Open++
			if summary.OldestOpen.IsZero() || record.CreatedAt.Before(summary.OldestOpen) {
				summary.OldestOpen = record.CreatedAt
			}
		}
		if record.Severity == SeverityCritical && record.Status != StatusResolved && record.Status != StatusArchived {
			summary.Critical++
		}
		if record.UpdatedAt.After(summary.LastUpdated) {
			summary.LastUpdated = record.UpdatedAt
		}
	}
	result := make([]LineSummary, 0, len(byLine))
	for _, summary := range byLine {
		result = append(result, *summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Line < result[j].Line })
	return result
}

func CountStatuses(records []Record) []StatusCount {
	counts := make(map[Status]int)
	for _, record := range records {
		counts[record.Status]++
	}
	statuses := []Status{StatusNew, StatusAcknowledged, StatusInvestigating, StatusMitigated, StatusResolved, StatusArchived, StatusRejected}
	result := make([]StatusCount, 0, len(statuses))
	for _, status := range statuses {
		if count := counts[status]; count > 0 {
			result = append(result, StatusCount{Status: status, Count: count})
		}
	}
	return result
}

func SortRecords(records []Record, newestFirst bool) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].UpdatedAt.Equal(records[j].UpdatedAt) {
			return records[i].ID < records[j].ID
		}
		if newestFirst {
			return records[i].UpdatedAt.After(records[j].UpdatedAt)
		}
		return records[i].UpdatedAt.Before(records[j].UpdatedAt)
	})
}
