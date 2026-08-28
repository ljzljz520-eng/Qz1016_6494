package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,63}$`)

func ValidateRecord(record Record) error {
	if !identifierPattern.MatchString(record.ID) {
		return errors.New("record id must be 2-64 safe characters")
	}
	if strings.TrimSpace(record.Line) == "" {
		return errors.New("line is required")
	}
	if strings.TrimSpace(record.Station) == "" {
		return errors.New("station is required")
	}
	if record.Severity < SeverityInfo || record.Severity > SeverityCritical {
		return fmt.Errorf("invalid severity %d", record.Severity)
	}
	if strings.TrimSpace(record.Summary) == "" {
		return errors.New("summary is required")
	}
	if len(record.Summary) > 180 {
		return errors.New("summary exceeds 180 characters")
	}
	if record.Status == "" {
		return errors.New("status is required")
	}
	if !KnownStatus(record.Status) {
		return fmt.Errorf("unknown status %q", record.Status)
	}
	return nil
}

func ValidateUser(user User) error {
	if !identifierPattern.MatchString(user.ID) {
		return errors.New("user id must be 2-64 safe characters")
	}
	if strings.TrimSpace(user.Name) == "" {
		return errors.New("user name is required")
	}
	switch user.Role {
	case "operator", "supervisor", "auditor", "administrator":
	default:
		return fmt.Errorf("unsupported role %q", user.Role)
	}
	if user.Email != "" {
		if _, err := mail.ParseAddress(user.Email); err != nil {
			return fmt.Errorf("invalid user email: %w", err)
		}
	}
	return nil
}

func ValidateEvent(event Event) error {
	if !identifierPattern.MatchString(event.ID) {
		return errors.New("event id is invalid")
	}
	if !identifierPattern.MatchString(event.RecordID) {
		return errors.New("event record id is invalid")
	}
	if strings.TrimSpace(event.Type) == "" {
		return errors.New("event type is required")
	}
	if event.Sequence < 1 {
		return errors.New("event sequence must be positive")
	}
	if event.At.IsZero() {
		return errors.New("event timestamp is required")
	}
	return nil
}

func ValidateAudit(audit Audit) error {
	if !identifierPattern.MatchString(audit.ID) {
		return errors.New("audit id is invalid")
	}
	if !identifierPattern.MatchString(audit.RecordID) {
		return errors.New("audit record id is invalid")
	}
	if strings.TrimSpace(audit.Action) == "" {
		return errors.New("audit action is required")
	}
	if strings.TrimSpace(audit.Digest) == "" {
		return errors.New("audit digest is required")
	}
	return nil
}

func NormalizeRecord(record Record) Record {
	record.ID = strings.TrimSpace(record.ID)
	record.Line = strings.TrimSpace(record.Line)
	record.Station = strings.TrimSpace(record.Station)
	record.Machine = strings.TrimSpace(record.Machine)
	record.Summary = strings.TrimSpace(record.Summary)
	record.Details = strings.TrimSpace(record.Details)
	record.OwnerID = strings.TrimSpace(record.OwnerID)
	record.Labels = NormalizeLabels(record.Labels)
	return record
}

func NormalizeLabels(labels []string) []string {
	seen := make(map[string]struct{}, len(labels))
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label == "" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	sort.Strings(result)
	return result
}

func KnownStatus(status Status) bool {
	switch status {
	case StatusNew, StatusAcknowledged, StatusInvestigating, StatusMitigated, StatusResolved, StatusArchived, StatusRejected:
		return true
	default:
		return false
	}
}

func CanAct(user User, line string, requiredRole string) bool {
	if !user.Active {
		return false
	}
	if user.Role == "administrator" {
		return true
	}
	if requiredRole != "" && user.Role != requiredRole {
		return false
	}
	if len(user.Lines) == 0 {
		return true
	}
	for _, assigned := range user.Lines {
		if assigned == line || assigned == "*" {
			return true
		}
	}
	return false
}
