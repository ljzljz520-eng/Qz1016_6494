package notify

import (
	"production43/internal/domain"
	"sort"
	"sync"
)

type Registry struct {
	mu            sync.RWMutex
	subscriptions map[string]domain.Subscription
}

func NewRegistry() *Registry { return &Registry{subscriptions: make(map[string]domain.Subscription)} }
func (r *Registry) Put(subscription domain.Subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscriptions[subscription.ID] = subscription
}
func (r *Registry) Remove(id string) { r.mu.Lock(); defer r.mu.Unlock(); delete(r.subscriptions, id) }
func (r *Registry) ForRecord(record domain.Record) []domain.Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Subscription, 0)
	for _, subscription := range r.subscriptions {
		if !subscription.Enabled || (subscription.Line != "" && subscription.Line != record.Line) || record.Severity < subscription.MinimumSeverity {
			continue
		}
		result = append(result, subscription)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func (r *Registry) Snapshot() []domain.Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Subscription, 0, len(r.subscriptions))
	for _, item := range r.subscriptions {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
