package query

import (
	"sort"
	"strings"
	"time"

	"production43/internal/domain"
	"production43/internal/store"
)

type Engine struct{ store *store.Store }

func New(st *store.Store) *Engine { return &Engine{store: st} }

func (e *Engine) Search(filter domain.RecordFilter) ([]domain.Record, error) {
	records, err := e.store.ListRecords()
	if err != nil {
		return nil, err
	}
	result := make([]domain.Record, 0, len(records))
	for _, record := range records {
		if domain.MatchFilter(record, filter) {
			result = append(result, record)
		}
	}
	domain.SortRecords(result, true)
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (e *Engine) Timeline(recordID string) (domain.Timeline, error) {
	record, err := e.store.GetRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	events, err := e.store.EventsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	audits, err := e.store.AuditsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	reviews, err := e.store.ReviewsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	deliveries, err := e.store.DeliveriesForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	return domain.Timeline{Record: record, Events: events, Audits: audits, Reviews: reviews, Deliveries: deliveries}, nil
}

func (e *Engine) TextSearch(records []domain.Record, term string) []domain.Record {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return append([]domain.Record(nil), records...)
	}
	result := make([]domain.Record, 0)
	for _, record := range records {
		haystack := strings.ToLower(record.ID + " " + record.Line + " " + record.Station + " " + record.Machine + " " + record.Summary + " " + record.Details)
		if strings.Contains(haystack, term) {
			result = append(result, record)
		}
	}
	return result
}

func (e *Engine) Stale(records []domain.Record, now time.Time, age time.Duration) []domain.Record {
	result := make([]domain.Record, 0)
	for _, record := range records {
		if record.Status != domain.StatusResolved && now.Sub(record.UpdatedAt) >= age {
			result = append(result, record)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.Before(result[j].UpdatedAt) })
	return result
}

func GroupByStation(records []domain.Record) map[string][]domain.Record {
	result := make(map[string][]domain.Record)
	for _, record := range records {
		result[record.Station] = append(result[record.Station], record)
	}
	for station, items := range result {
		domain.SortRecords(items, true)
		result[station] = items
	}
	return result
}
