package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReportRow struct {
	RecordID   string        `json:"record_id"`
	Line       string        `json:"line"`
	Station    string        `json:"station"`
	Severity   Severity      `json:"severity"`
	Status     Status        `json:"status"`
	Age        time.Duration `json:"age"`
	Owner      string        `json:"owner"`
	EventCount int           `json:"event_count"`
	AuditCount int           `json:"audit_count"`
}

type Report struct {
	Title       string         `json:"title"`
	GeneratedAt time.Time      `json:"generated_at"`
	Rows        []ReportRow    `json:"rows"`
	Totals      map[string]int `json:"totals"`
	Warnings    []string       `json:"warnings"`
}

func BuildReport(records []Record, events map[string][]Event, audits map[string][]Audit, now time.Time) Report {
	rows := make([]ReportRow, 0, len(records))
	totals := map[string]int{}
	for _, record := range records {
		row := ReportRow{RecordID: record.ID, Line: record.Line, Station: record.Station, Severity: record.Severity, Status: record.Status, Age: now.Sub(record.CreatedAt), Owner: record.OwnerID, EventCount: len(events[record.ID]), AuditCount: len(audits[record.ID])}
		if row.Age < 0 {
			row.Age = 0
		}
		rows = append(rows, row)
		totals[string(record.Status)]++
		totals[record.Line]++
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Line == rows[j].Line {
			return rows[i].RecordID < rows[j].RecordID
		}
		return rows[i].Line < rows[j].Line
	})
	warnings := make([]string, 0)
	for _, row := range rows {
		if row.Severity == SeverityCritical && row.Status != StatusResolved && row.Status != StatusArchived {
			warnings = append(warnings, fmt.Sprintf("%s requires supervisor attention", row.RecordID))
		}
		if row.EventCount == 0 {
			warnings = append(warnings, fmt.Sprintf("%s has no lifecycle events", row.RecordID))
		}
	}
	return Report{Title: "Production Line Alert Report", GeneratedAt: now, Rows: rows, Totals: totals, Warnings: warnings}
}

func RenderReport(report Report) string {
	var builder strings.Builder
	builder.WriteString(report.Title)
	builder.WriteByte('\n')
	builder.WriteString(report.GeneratedAt.Format(time.RFC3339))
	builder.WriteByte('\n')
	for _, row := range report.Rows {
		builder.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%d\t%d\n", row.RecordID, row.Line, row.Station, row.Severity, row.Status, row.Age, row.Owner, row.EventCount, row.AuditCount))
	}
	if len(report.Warnings) > 0 {
		builder.WriteString("Warnings:\n")
		for _, warning := range report.Warnings {
			builder.WriteString("- " + warning + "\n")
		}
	}
	return builder.String()
}

func ParseStatus(value string) (Status, error) {
	normalized := Status(strings.ToLower(strings.TrimSpace(value)))
	if !KnownStatus(normalized) {
		return "", fmt.Errorf("invalid status %q", value)
	}
	return normalized, nil
}
func ParseSeverity(value string) (Severity, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "info", "informational":
		return SeverityInfo, nil
	case "2", "warning", "warn":
		return SeverityWarning, nil
	case "3", "critical", "crit":
		return SeverityCritical, nil
	default:
		return 0, fmt.Errorf("invalid severity %q", value)
	}
}
func SeverityLabel(severity Severity) string {
	switch severity {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}
func StatusTerminal(status Status) bool {
	return status == StatusResolved || status == StatusArchived || status == StatusRejected
}
func StatusOpen(status Status) bool { return KnownStatus(status) && !StatusTerminal(status) }
func CopyRecord(record Record) Record {
	record.Labels = append([]string(nil), record.Labels...)
	return record
}
func CopyEvents(events []Event) []Event {
	result := make([]Event, len(events))
	copy(result, events)
	return result
}
func CopyAudits(audits []Audit) []Audit {
	result := make([]Audit, len(audits))
	copy(result, audits)
	return result
}
func LatestEvent(events []Event) (Event, bool) {
	if len(events) == 0 {
		return Event{}, false
	}
	latest := events[0]
	for _, event := range events[1:] {
		if event.Sequence > latest.Sequence || (event.Sequence == latest.Sequence && event.At.After(latest.At)) {
			latest = event
		}
	}
	return latest, true
}
func EventTypes(events []Event) []string {
	result := make([]string, 0, len(events))
	seen := map[string]bool{}
	for _, event := range events {
		if !seen[event.Type] {
			seen[event.Type] = true
			result = append(result, event.Type)
		}
	}
	sort.Strings(result)
	return result
}
func MergeLabels(records []Record) []string {
	seen := map[string]bool{}
	result := make([]string, 0)
	for _, record := range records {
		for _, label := range record.Labels {
			if !seen[label] {
				seen[label] = true
				result = append(result, label)
			}
		}
	}
	sort.Strings(result)
	return result
}
func FilterByAge(records []Record, now time.Time, min, max time.Duration) []Record {
	result := make([]Record, 0)
	for _, record := range records {
		age := now.Sub(record.CreatedAt)
		if age >= min && (max <= 0 || age <= max) {
			result = append(result, record)
		}
	}
	return result
}
func FilterByStatus(records []Record, status Status) []Record {
	result := make([]Record, 0)
	for _, record := range records {
		if record.Status == status {
			result = append(result, record)
		}
	}
	return result
}
func FilterByLine(records []Record, line string) []Record {
	result := make([]Record, 0)
	for _, record := range records {
		if record.Line == line {
			result = append(result, record)
		}
	}
	return result
}
func OrderForExport(records []Record) []Record {
	result := make([]Record, len(records))
	for i, record := range records {
		result[i] = CopyRecord(record)
	}
	SortRecords(result, false)
	return result
}
func ValidateBundle(record Record, user User, event Event) error {
	if err := ValidateRecord(record); err != nil {
		return err
	}
	if err := ValidateUser(user); err != nil {
		return err
	}
	if err := ValidateEvent(event); err != nil {
		return err
	}
	if event.RecordID != record.ID {
		return fmt.Errorf("event record mismatch")
	}
	return nil
}
func IsLine43(record Record) bool {
	return strings.TrimSpace(record.Line) == "43" || strings.EqualFold(strings.TrimSpace(record.Line), "production-43")
}
func EnsureTimestamp(value, timeFallback time.Time) time.Time {
	if value.IsZero() {
		return timeFallback
	}
	return value
}
func NormalizeStatusChain(events []Event) []Status {
	result := make([]Status, 0, len(events))
	for _, event := range events {
		if event.ToStatus != "" {
			result = append(result, event.ToStatus)
		}
	}
	return result
}
func HasTransition(events []Event, from, to Status) bool {
	for _, event := range events {
		if event.FromStatus == from && event.ToStatus == to {
			return true
		}
	}
	return false
}
func DistinctActors(events []Event) []string {
	seen := map[string]bool{}
	result := make([]string, 0)
	for _, event := range events {
		if event.ActorID != "" && !seen[event.ActorID] {
			seen[event.ActorID] = true
			result = append(result, event.ActorID)
		}
	}
	sort.Strings(result)
	return result
}
func Summaries(records []Record) []string {
	result := make([]string, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.Summary) != "" {
			result = append(result, record.Summary)
		}
	}
	return result
}
func SeverityAtLeast(record Record, minimum Severity) bool { return record.Severity >= minimum }
func HasLabel(record Record, label string) bool {
	for _, candidate := range record.Labels {
		if candidate == label {
			return true
		}
	}
	return false
}
func WithLabel(record Record, label string) Record {
	record = CopyRecord(record)
	if !HasLabel(record, label) {
		record.Labels = append(record.Labels, label)
		record.Labels = NormalizeLabels(record.Labels)
	}
	return record
}
func RemoveLabel(record Record, label string) Record {
	record = CopyRecord(record)
	kept := record.Labels[:0]
	for _, candidate := range record.Labels {
		if candidate != label {
			kept = append(kept, candidate)
		}
	}
	record.Labels = kept
	return record
}
func CompareVersion(record Record, expected int) error {
	if expected > 0 && record.Version != expected {
		return fmt.Errorf("expected version %d got %d", expected, record.Version)
	}
	return nil
}
func NextVersion(record Record) int {
	if record.Version < 1 {
		return 1
	}
	return record.Version + 1
}
func IsFresh(record Record, now time.Time, window time.Duration) bool {
	return !record.UpdatedAt.IsZero() && now.Sub(record.UpdatedAt) <= window
}
func RecordKey(record Record) string { return record.Line + "/" + record.Station + "/" + record.ID }
func EventKey(event Event) string    { return event.RecordID + "/" + fmt.Sprint(event.Sequence) }
func AuditKey(audit Audit) string {
	return audit.RecordID + "/" + audit.Action + "/" + audit.At.Format(time.RFC3339Nano)
}
func SortEvents(events []Event) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Sequence == events[j].Sequence {
			return events[i].At.Before(events[j].At)
		}
		return events[i].Sequence < events[j].Sequence
	})
}
func SortAudits(audits []Audit) {
	sort.SliceStable(audits, func(i, j int) bool { return audits[i].At.Before(audits[j].At) })
}

func OpenDuration(record Record, now time.Time) time.Duration {
	if StatusTerminal(record.Status) {
		return record.UpdatedAt.Sub(record.CreatedAt)
	}
	return now.Sub(record.CreatedAt)
}

func IsOverdue(record Record, now time.Time, limit time.Duration) bool {
	if StatusTerminal(record.Status) || record.CreatedAt.IsZero() {
		return false
	}
	return OpenDuration(record, now) > limit
}

func CountCritical(records []Record) int {
	count := 0
	for _, record := range records {
		if record.Severity == SeverityCritical && StatusOpen(record.Status) {
			count++
		}
	}
	return count
}

func CountBySeverity(records []Record) map[Severity]int {
	counts := make(map[Severity]int)
	for _, record := range records {
		counts[record.Severity]++
	}
	return counts
}

func LastUpdated(records []Record) time.Time {
	var latest time.Time
	for _, record := range records {
		if record.UpdatedAt.After(latest) {
			latest = record.UpdatedAt
		}
	}
	return latest
}

func EarliestOpen(records []Record) time.Time {
	var earliest time.Time
	for _, record := range records {
		if StatusOpen(record.Status) && (earliest.IsZero() || record.CreatedAt.Before(earliest)) {
			earliest = record.CreatedAt
		}
	}
	return earliest
}
func TruncateDetails(record Record, max int) Record {
	record = CopyRecord(record)
	if max > 0 && len(record.Details) > max {
		record.Details = record.Details[:max]
	}
	return record
}
func AddDetail(record Record, detail string) Record {
	record = CopyRecord(record)
	detail = strings.TrimSpace(detail)
	if detail != "" {
		if record.Details != "" {
			record.Details += "\n"
		}
		record.Details += detail
	}
	return record
}
func RecordEqual(a, b Record) bool {
	return a.ID == b.ID && a.Version == b.Version && a.Status == b.Status && a.UpdatedAt.Equal(b.UpdatedAt)
}
func EventEqual(a, b Event) bool {
	return a.ID == b.ID && a.Sequence == b.Sequence && a.ToStatus == b.ToStatus
}
func AuditEqual(a, b Audit) bool { return a.ID == b.ID && a.Digest == b.Digest }
