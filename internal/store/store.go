package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"production43/internal/domain"
)

var (
	bucketRecords    = []byte("records")
	bucketUsers      = []byte("users")
	bucketEvents     = []byte("events")
	bucketAudits     = []byte("audits")
	bucketReviews    = []byte("reviews")
	bucketDeliveries = []byte("deliveries")
)

var ErrNotFound = errors.New("entity not found")

type Store struct {
	db   *bbolt.DB
	path string
	mu   sync.RWMutex
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 2 * time.Second, NoSync: false})
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}
	store := &Store{db: db, path: path}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range [][]byte{bucketRecords, bucketUsers, bucketEvents, bucketAudits, bucketReviews, bucketDeliveries} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *Store) Path() string { return s.path }

func putJSON(bucket *bbolt.Bucket, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

func getJSON(bucket *bbolt.Bucket, key string, target any) error {
	data := bucket.Get([]byte(key))
	if data == nil {
		return ErrNotFound
	}
	return json.Unmarshal(data, target)
}

func (s *Store) SaveRecord(record domain.Record) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.Update(func(tx *bbolt.Tx) error { return putJSON(tx.Bucket(bucketRecords), record.ID, record) })
}

func (s *Store) GetRecord(id string) (domain.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var record domain.Record
	if s.db == nil {
		return record, errors.New("store is closed")
	}
	err := s.db.View(func(tx *bbolt.Tx) error { return getJSON(tx.Bucket(bucketRecords), id, &record) })
	return record, err
}

func (s *Store) DeleteRecord(id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.Update(func(tx *bbolt.Tx) error { return tx.Bucket(bucketRecords).Delete([]byte(id)) })
}

func (s *Store) ListRecords() ([]domain.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return nil, errors.New("store is closed")
	}
	result := make([]domain.Record, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketRecords).ForEach(func(_, data []byte) error {
			var record domain.Record
			if err := json.Unmarshal(data, &record); err != nil {
				return err
			}
			result = append(result, record)
			return nil
		})
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, err
}

func (s *Store) SaveUser(user domain.User) error { return s.save(bucketUsers, user.ID, user) }
func (s *Store) GetUser(id string) (domain.User, error) {
	var user domain.User
	err := s.load(bucketUsers, id, &user)
	return user, err
}
func (s *Store) SaveEvent(event domain.Event) error { return s.save(bucketEvents, event.ID, event) }
func (s *Store) SaveAudit(audit domain.Audit) error { return s.save(bucketAudits, audit.ID, audit) }
func (s *Store) SaveReview(review domain.Review) error {
	return s.save(bucketReviews, review.ID, review)
}
func (s *Store) SaveDelivery(delivery domain.Delivery) error {
	return s.save(bucketDeliveries, delivery.ID, delivery)
}

func (s *Store) save(bucket []byte, key string, value any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.Update(func(tx *bbolt.Tx) error { return putJSON(tx.Bucket(bucket), key, value) })
}

func (s *Store) load(bucket []byte, key string, target any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.View(func(tx *bbolt.Tx) error { return getJSON(tx.Bucket(bucket), key, target) })
}

func (s *Store) list(bucket []byte, target func([]byte) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	return s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).ForEach(func(_, value []byte) error { return target(value) })
	})
}

func (s *Store) EventsForRecord(recordID string) ([]domain.Event, error) {
	result := make([]domain.Event, 0)
	err := s.list(bucketEvents, func(data []byte) error {
		var item domain.Event
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.RecordID == recordID {
			result = append(result, item)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, err
}

func (s *Store) AuditsForRecord(recordID string) ([]domain.Audit, error) {
	result := make([]domain.Audit, 0)
	err := s.list(bucketAudits, func(data []byte) error {
		var item domain.Audit
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.RecordID == recordID {
			result = append(result, item)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].At.Before(result[j].At) })
	return result, err
}

func (s *Store) ReviewsForRecord(recordID string) ([]domain.Review, error) {
	result := make([]domain.Review, 0)
	err := s.list(bucketReviews, func(data []byte) error {
		var item domain.Review
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.RecordID == recordID {
			result = append(result, item)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].At.Before(result[j].At) })
	return result, err
}

func (s *Store) DeliveriesForRecord(recordID string) ([]domain.Delivery, error) {
	result := make([]domain.Delivery, 0)
	err := s.list(bucketDeliveries, func(data []byte) error {
		var item domain.Delivery
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.RecordID == recordID {
			result = append(result, item)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, err
}
