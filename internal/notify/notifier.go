package notify

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"production43/internal/domain"
	"production43/internal/store"
)

type Notifier struct {
	store *store.Store
	mu    sync.Mutex
	sent  []domain.Delivery
}

func New(st *store.Store) *Notifier { return &Notifier{store: st} }

func (n *Notifier) Notify(record domain.Record, users []domain.User, now time.Time) ([]domain.Delivery, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	selected := domain.SelectEscalationUsers(users, record, domain.DefaultEscalationPolicy())
	result := make([]domain.Delivery, 0)
	for _, user := range selected {
		channels := userChannels(user)
		for _, channel := range channels {
			message := formatMessage(record, user, channel)
			delivery := store.NewDelivery(record.ID, user.ID, channel, message, now)
			if err := n.store.SaveDelivery(delivery); err != nil {
				return result, err
			}
			n.sent = append(n.sent, delivery)
			result = append(result, delivery)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func userChannels(user domain.User) []string {
	if user.Email != "" {
		return []string{"email", "dashboard"}
	}
	return []string{"dashboard"}
}
func formatMessage(record domain.Record, user domain.User, channel string) string {
	prefix := strings.ToUpper(string(record.Status))
	return fmt.Sprintf("[%s] line %s station %s: %s for %s via %s", prefix, record.Line, record.Station, record.Summary, user.Name, channel)
}
func (n *Notifier) Sent() []domain.Delivery {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]domain.Delivery(nil), n.sent...)
}
func (n *Notifier) Acknowledge(deliveryID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range n.sent {
		if n.sent[i].ID == deliveryID {
			n.sent[i].State = "acknowledged"
			return true
		}
	}
	return false
}
func (n *Notifier) RetryPending(now time.Time) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	count := 0
	for i := range n.sent {
		if n.sent[i].State == "failed" && n.sent[i].Attempts < 3 {
			n.sent[i].Attempts++
			n.sent[i].State = "delivered"
			n.sent[i].DeliveredAt = now
			count++
		}
	}
	return count
}
