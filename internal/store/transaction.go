package store

import (
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
	"production43/internal/domain"
)

type Bundle struct {
	Record   *domain.Record
	User     *domain.User
	Event    *domain.Event
	Audit    *domain.Audit
	Review   *domain.Review
	Delivery *domain.Delivery
}

func (s *Store) SaveBundle(bundle Bundle) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		if bundle.Record != nil {
			if err := putJSON(tx.Bucket(bucketRecords), bundle.Record.ID, bundle.Record); err != nil {
				return fmt.Errorf("record: %w", err)
			}
		}
		if bundle.User != nil {
			if err := putJSON(tx.Bucket(bucketUsers), bundle.User.ID, bundle.User); err != nil {
				return fmt.Errorf("user: %w", err)
			}
		}
		if bundle.Event != nil {
			if err := putJSON(tx.Bucket(bucketEvents), bundle.Event.ID, bundle.Event); err != nil {
				return fmt.Errorf("event: %w", err)
			}
		}
		if bundle.Audit != nil {
			if err := putJSON(tx.Bucket(bucketAudits), bundle.Audit.ID, bundle.Audit); err != nil {
				return fmt.Errorf("audit: %w", err)
			}
		}
		if bundle.Review != nil {
			if err := putJSON(tx.Bucket(bucketReviews), bundle.Review.ID, bundle.Review); err != nil {
				return fmt.Errorf("review: %w", err)
			}
		}
		if bundle.Delivery != nil {
			if err := putJSON(tx.Bucket(bucketDeliveries), bundle.Delivery.ID, bundle.Delivery); err != nil {
				return fmt.Errorf("delivery: %w", err)
			}
		}
		return nil
	})
}

func NewEvent(record domain.Record, actorID, eventType, payload string, from, to domain.Status, sequence int, now time.Time) domain.Event {
	return domain.Event{ID: fmt.Sprintf("%s-%06d", record.ID, sequence), RecordID: record.ID, Type: eventType, ActorID: actorID, Payload: payload, FromStatus: from, ToStatus: to, Sequence: sequence, At: now}
}

func NewAudit(record domain.Record, actorID, action, reason string, now time.Time) domain.Audit {
	return domain.Audit{ID: fmt.Sprintf("audit-%s-%d", record.ID, now.UnixNano()), RecordID: record.ID, Action: action, ActorID: actorID, Reason: reason, Digest: domain.RecordDigest(record), At: now}
}

func NewReview(recordID, reviewerID, decision, reason string, now time.Time) domain.Review {
	return domain.Review{ID: fmt.Sprintf("review-%s-%d", recordID, now.UnixNano()), RecordID: recordID, ReviewerID: reviewerID, Decision: decision, Reason: reason, At: now}
}

func NewDelivery(recordID, userID, channel, message string, now time.Time) domain.Delivery {
	return domain.Delivery{ID: fmt.Sprintf("delivery-%s-%s-%s", recordID, userID, channel), RecordID: recordID, UserID: userID, Channel: channel, Message: message, State: "delivered", Attempts: 1, DeliveredAt: now}
}
