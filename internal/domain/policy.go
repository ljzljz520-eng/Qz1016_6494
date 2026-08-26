package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type EscalationPolicy struct {
	CriticalAfter   time.Duration
	WarningAfter    time.Duration
	RepeatEvery     time.Duration
	SupervisorRoles []string
}

type Escalation struct {
	RecordID string
	Reason   string
	Priority int
	DueAt    time.Time
}

func DefaultEscalationPolicy() EscalationPolicy {
	return EscalationPolicy{
		CriticalAfter:   5 * time.Minute,
		WarningAfter:    30 * time.Minute,
		RepeatEvery:     15 * time.Minute,
		SupervisorRoles: []string{"supervisor", "administrator"},
	}
}

func EvaluateEscalation(record Record, now time.Time, policy EscalationPolicy) (Escalation, bool) {
	if record.Status == StatusResolved || record.Status == StatusArchived || record.Status == StatusRejected {
		return Escalation{}, false
	}
	age := now.Sub(record.CreatedAt)
	switch record.Severity {
	case SeverityCritical:
		if age >= policy.CriticalAfter {
			return Escalation{RecordID: record.ID, Reason: "critical alert exceeded response window", Priority: 100, DueAt: record.CreatedAt.Add(policy.CriticalAfter)}, true
		}
	case SeverityWarning:
		if age >= policy.WarningAfter {
			return Escalation{RecordID: record.ID, Reason: "warning alert exceeded response window", Priority: 50, DueAt: record.CreatedAt.Add(policy.WarningAfter)}, true
		}
	case SeverityInfo:
		if age >= 4*policy.WarningAfter && record.Status == StatusNew {
			return Escalation{RecordID: record.ID, Reason: "informational alert remains unacknowledged", Priority: 10, DueAt: record.CreatedAt.Add(4 * policy.WarningAfter)}, true
		}
	}
	return Escalation{}, false
}

func SelectEscalationUsers(users []User, record Record, policy EscalationPolicy) []User {
	roles := make(map[string]struct{}, len(policy.SupervisorRoles))
	for _, role := range policy.SupervisorRoles {
		roles[role] = struct{}{}
	}
	selected := make([]User, 0)
	for _, user := range users {
		if !user.Active {
			continue
		}
		if _, ok := roles[user.Role]; !ok {
			continue
		}
		if !CanAct(user, record.Line, "") {
			continue
		}
		selected = append(selected, user)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	return selected
}

func RecordDigest(record Record) string {
	labels := append([]string(nil), record.Labels...)
	sort.Strings(labels)
	return fmt.Sprintf("%s|%s|%s|%d|%s|%s|%d", record.ID, record.Line, record.Station, record.Severity, record.Status, strings.Join(labels, ","), record.Version)
}

func MatchFilter(record Record, filter RecordFilter) bool {
	if filter.Line != "" && record.Line != filter.Line {
		return false
	}
	if filter.OwnerID != "" && record.OwnerID != filter.OwnerID {
		return false
	}
	if !filter.UpdatedAfter.IsZero() && !record.UpdatedAt.After(filter.UpdatedAfter) {
		return false
	}
	if filter.Label != "" {
		found := false
		for _, label := range record.Labels {
			if label == filter.Label {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if record.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(filter.Severities) > 0 {
		found := false
		for _, severity := range filter.Severities {
			if record.Severity == severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
