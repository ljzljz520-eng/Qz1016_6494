package domain

import "time"

type Status string

const (
	StatusNew           Status = "new"
	StatusAcknowledged  Status = "acknowledged"
	StatusInvestigating Status = "investigating"
	StatusMitigated     Status = "mitigated"
	StatusResolved      Status = "resolved"
	StatusArchived      Status = "archived"
	StatusRejected      Status = "rejected"
)

type Severity int

const (
	SeverityInfo Severity = iota + 1
	SeverityWarning
	SeverityCritical
)

type Record struct {
	ID        string    `json:"id"`
	Line      string    `json:"line"`
	Station   string    `json:"station"`
	Machine   string    `json:"machine"`
	Severity  Severity  `json:"severity"`
	Status    Status    `json:"status"`
	Summary   string    `json:"summary"`
	Details   string    `json:"details"`
	OwnerID   string    `json:"owner_id"`
	Labels    []string  `json:"labels"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Email     string    `json:"email"`
	Lines     []string  `json:"lines"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	ID         string    `json:"id"`
	RecordID   string    `json:"record_id"`
	Type       string    `json:"type"`
	FromStatus Status    `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	ActorID    string    `json:"actor_id"`
	Payload    string    `json:"payload"`
	Sequence   int       `json:"sequence"`
	At         time.Time `json:"at"`
}

type Audit struct {
	ID       string    `json:"id"`
	RecordID string    `json:"record_id"`
	Action   string    `json:"action"`
	ActorID  string    `json:"actor_id"`
	Reason   string    `json:"reason"`
	Digest   string    `json:"digest"`
	At       time.Time `json:"at"`
}

type Subscription struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	Line            string   `json:"line"`
	MinimumSeverity Severity `json:"minimum_severity"`
	Channels        []string `json:"channels"`
	Enabled         bool     `json:"enabled"`
}

type Delivery struct {
	ID          string    `json:"id"`
	RecordID    string    `json:"record_id"`
	UserID      string    `json:"user_id"`
	Channel     string    `json:"channel"`
	Message     string    `json:"message"`
	State       string    `json:"state"`
	Attempts    int       `json:"attempts"`
	DeliveredAt time.Time `json:"delivered_at"`
}

type Review struct {
	ID         string    `json:"id"`
	RecordID   string    `json:"record_id"`
	ReviewerID string    `json:"reviewer_id"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	At         time.Time `json:"at"`
}

type RecordFilter struct {
	Line         string
	Statuses     []Status
	Severities   []Severity
	OwnerID      string
	Label        string
	UpdatedAfter time.Time
	Limit        int
}

type Timeline struct {
	Record     Record     `json:"record"`
	Events     []Event    `json:"events"`
	Audits     []Audit    `json:"audits"`
	Reviews    []Review   `json:"reviews"`
	Deliveries []Delivery `json:"deliveries"`
}
