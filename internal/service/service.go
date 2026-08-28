package service

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"production43/internal/domain"
	"production43/internal/store"
)

var (
	ErrUnauthorized = errors.New("actor is not authorized")
	ErrConflict     = errors.New("record version conflict")
	ErrInvalidInput = errors.New("invalid input")
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type Service struct {
	store  *store.Store
	clock  Clock
	nextID atomic.Uint64
	policy domain.EscalationPolicy
}

func New(st *store.Store, clock Clock) *Service {
	if clock == nil {
		clock = RealClock{}
	}
	return &Service{store: st, clock: clock, policy: domain.DefaultEscalationPolicy()}
}

func (s *Service) Register(record domain.Record, actor domain.User) (domain.Record, error) {
	record = domain.NormalizeRecord(record)
	if record.ID == "" {
		record.ID = fmt.Sprintf("alert-%06d", s.nextID.Add(1))
	}
	if record.Status == "" {
		record.Status = domain.StatusNew
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = s.clock.Now()
	}
	record.UpdatedAt = record.CreatedAt
	record.Version = 1
	if err := domain.ValidateRecord(record); err != nil {
		return record, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := domain.ValidateUser(actor); err != nil || !domain.CanAct(actor, record.Line, "operator") {
		return record, ErrUnauthorized
	}
	event := store.NewEvent(record, actor.ID, "alert.created", record.Summary, "", record.Status, 1, record.CreatedAt)
	if err := s.store.SaveBundle(store.Bundle{Record: &record, User: &actor, Event: &event}); err != nil {
		return record, err
	}
	return record, nil
}

func (s *Service) Process(recordID string, actor domain.User, action domain.Action, note string, expectedVersion int) (domain.Record, error) {
	record, err := s.store.GetRecord(recordID)
	if err != nil {
		return record, err
	}
	if expectedVersion > 0 && record.Version != expectedVersion {
		return record, ErrConflict
	}
	if err := domain.ValidateUser(actor); err != nil || !domain.CanAct(actor, record.Line, domain.TransitionRole(action)) {
		return record, ErrUnauthorized
	}
	next, transitionErr := domain.NextStatus(record.Status, action)
	if transitionErr != nil {
		return record, transitionErr
	}
	now := s.clock.Now()
	previous := record.Status
	record.Status = next
	if action == domain.ActionResolve {
		record.Status = previous
	}
	record.Version++
	record.UpdatedAt = now
	sequence := record.Version
	event := store.NewEvent(record, actor.ID, domain.TransitionEventType(action), strings.TrimSpace(note), previous, next, sequence, now)
	if err := s.store.SaveBundle(store.Bundle{Record: &record, Event: &event}); err != nil {
		return record, err
	}
	return record, nil
}

func (s *Service) Review(recordID string, reviewer domain.User, decision, reason string) (domain.Review, domain.Record, error) {
	record, err := s.store.GetRecord(recordID)
	if err != nil {
		return domain.Review{}, record, err
	}
	if !domain.CanAct(reviewer, record.Line, "supervisor") {
		return domain.Review{}, record, ErrUnauthorized
	}
	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision != "approve" && decision != "reject" {
		return domain.Review{}, record, errors.New("decision must be approve or reject")
	}
	action := domain.ActionArchive
	if decision == "reject" {
		action = domain.ActionReject
	}
	updated, err := s.Process(recordID, reviewer, action, reason, record.Version)
	if err != nil {
		return domain.Review{}, record, err
	}
	now := s.clock.Now()
	review := store.NewReview(recordID, reviewer.ID, decision, reason, now)
	audit := store.NewAudit(updated, reviewer.ID, "review."+decision, reason, now)
	if err := s.store.SaveBundle(store.Bundle{Review: &review, Audit: &audit}); err != nil {
		return review, updated, err
	}
	return review, updated, nil
}

func (s *Service) Get(recordID string) (domain.Record, error) { return s.store.GetRecord(recordID) }
func (s *Service) List(filter domain.RecordFilter) ([]domain.Record, error) {
	records, err := s.store.ListRecords()
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.Record, 0, len(records))
	for _, record := range records {
		if domain.MatchFilter(record, filter) {
			filtered = append(filtered, record)
		}
	}
	domain.SortRecords(filtered, true)
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}
	return filtered, nil
}

func (s *Service) Timeline(recordID string) (domain.Timeline, error) {
	record, err := s.store.GetRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	events, err := s.store.EventsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	audits, err := s.store.AuditsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	reviews, err := s.store.ReviewsForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	deliveries, err := s.store.DeliveriesForRecord(recordID)
	if err != nil {
		return domain.Timeline{}, err
	}
	return domain.Timeline{Record: record, Events: events, Audits: audits, Reviews: reviews, Deliveries: deliveries}, nil
}

func (s *Service) Escalation(recordID string) (domain.Escalation, bool, error) {
	record, err := s.store.GetRecord(recordID)
	if err != nil {
		return domain.Escalation{}, false, err
	}
	escalation, ok := domain.EvaluateEscalation(record, s.clock.Now(), s.policy)
	return escalation, ok, nil
}

func (s *Service) SeedUser(user domain.User) error {
	if user.CreatedAt.IsZero() {
		user.CreatedAt = s.clock.Now()
	}
	if err := domain.ValidateUser(user); err != nil {
		return err
	}
	return s.store.SaveUser(user)
}
