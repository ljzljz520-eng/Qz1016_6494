package service

import (
	"context"
	"fmt"
	"production43/internal/domain"
	"sort"
	"sync"
	"time"
)

type BatchResult struct {
	Accepted []domain.Record
	Failed   map[string]error
	Duration time.Duration
}

func (s *Service) RegisterBatch(ctx context.Context, records []domain.Record, actor domain.User) BatchResult {
	started := s.clock.Now()
	result := BatchResult{Accepted: make([]domain.Record, 0, len(records)), Failed: make(map[string]error)}
	for _, record := range records {
		select {
		case <-ctx.Done():
			result.Failed[record.ID] = ctx.Err()
			continue
		default:
		}
		created, err := s.Register(record, actor)
		if err != nil {
			result.Failed[record.ID] = err
		} else {
			result.Accepted = append(result.Accepted, created)
		}
	}
	result.Duration = s.clock.Now().Sub(started)
	return result
}
func (s *Service) ProcessBatch(ctx context.Context, ids []string, actor domain.User, action domain.Action) BatchResult {
	started := s.clock.Now()
	result := BatchResult{Accepted: make([]domain.Record, 0, len(ids)), Failed: make(map[string]error)}
	for _, id := range ids {
		select {
		case <-ctx.Done():
			result.Failed[id] = ctx.Err()
			continue
		default:
		}
		record, err := s.Get(id)
		if err != nil {
			result.Failed[id] = err
			continue
		}
		updated, err := s.Process(id, actor, action, "batch", record.Version)
		if err != nil {
			result.Failed[id] = err
		} else {
			result.Accepted = append(result.Accepted, updated)
		}
	}
	result.Duration = s.clock.Now().Sub(started)
	return result
}
func (s *Service) Reconcile(ctx context.Context, filter domain.RecordFilter, actor domain.User) (int, error) {
	records, err := s.List(filter)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, record := range records {
		select {
		case <-ctx.Done():
			return changed, ctx.Err()
		default:
		}
		if record.Status == domain.StatusNew {
			if _, err := s.Process(record.ID, actor, domain.ActionAcknowledge, "reconcile", record.Version); err != nil {
				return changed, err
			}
			changed++
		}
	}
	return changed, nil
}
func (s *Service) ParallelEscalations(ctx context.Context, records []domain.Record) []domain.Escalation {
	var wg sync.WaitGroup
	result := make([]domain.Escalation, 0)
	var mu sync.Mutex
	for _, record := range records {
		record := record
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			if escalation, ok := domain.EvaluateEscalation(record, s.clock.Now(), s.policy); ok {
				mu.Lock()
				result = append(result, escalation)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].RecordID < result[j].RecordID
		}
		return result[i].Priority > result[j].Priority
	})
	return result
}
func (s *Service) Assign(recordID string, owner domain.User) (domain.Record, error) {
	record, err := s.Get(recordID)
	if err != nil {
		return record, err
	}
	if !domain.CanAct(owner, record.Line, "") {
		return record, ErrUnauthorized
	}
	record.OwnerID = owner.ID
	record.Version++
	record.UpdatedAt = s.clock.Now()
	if err := s.store.SaveRecord(record); err != nil {
		return record, err
	}
	return record, nil
}
func (s *Service) AddLabel(recordID, label string, actor domain.User) (domain.Record, error) {
	record, err := s.Get(recordID)
	if err != nil {
		return record, err
	}
	if !domain.CanAct(actor, record.Line, "") {
		return record, ErrUnauthorized
	}
	record = domain.WithLabel(record, label)
	record.Version++
	record.UpdatedAt = s.clock.Now()
	if err := s.store.SaveRecord(record); err != nil {
		return record, err
	}
	return record, nil
}
func (s *Service) RemoveLabel(recordID, label string, actor domain.User) (domain.Record, error) {
	record, err := s.Get(recordID)
	if err != nil {
		return record, err
	}
	if !domain.CanAct(actor, record.Line, "") {
		return record, ErrUnauthorized
	}
	record = domain.RemoveLabel(record, label)
	record.Version++
	record.UpdatedAt = s.clock.Now()
	if err := s.store.SaveRecord(record); err != nil {
		return record, err
	}
	return record, nil
}
func (s *Service) ValidateOwnership(recordID string, userID string) error {
	record, err := s.Get(recordID)
	if err != nil {
		return err
	}
	if record.OwnerID != "" && record.OwnerID != userID {
		return fmt.Errorf("record owned by %s", record.OwnerID)
	}
	return nil
}

func (s *Service) Touch(recordID string, actor domain.User, detail string) (domain.Record, error) {
	record, err := s.Get(recordID)
	if err != nil {
		return record, err
	}
	if !domain.CanAct(actor, record.Line, "") {
		return record, ErrUnauthorized
	}
	record = domain.AddDetail(record, detail)
	record.Version++
	record.UpdatedAt = s.clock.Now()
	if err := s.store.SaveRecord(record); err != nil {
		return record, err
	}
	return record, nil
}

func (s *Service) ArchiveIfResolved(recordID string, actor domain.User) (domain.Record, bool, error) {
	record, err := s.Get(recordID)
	if err != nil {
		return record, false, err
	}
	if record.Status != domain.StatusResolved {
		return record, false, nil
	}
	updated, err := s.Process(recordID, actor, domain.ActionArchive, "automatic archive", record.Version)
	if err != nil {
		return record, false, err
	}
	return updated, true, nil
}

func (s *Service) CountsByLine() (map[string]int, error) {
	records, err := s.List(domain.RecordFilter{})
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, record := range records {
		counts[record.Line]++
	}
	return counts, nil
}

func (s *Service) OpenRecords() ([]domain.Record, error) {
	return s.List(domain.RecordFilter{Statuses: []domain.Status{domain.StatusNew, domain.StatusAcknowledged, domain.StatusInvestigating, domain.StatusMitigated}})
}
