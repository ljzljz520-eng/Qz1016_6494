package service

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"production43/internal/domain"
)

type ImportResult struct {
	Records []domain.Record
	Errors  []string
}

func ParseCSV(reader io.Reader) ImportResult {
	csvReader := csv.NewReader(bufio.NewReader(reader))
	csvReader.FieldsPerRecord = -1
	result := ImportResult{}
	lineNumber := 0
	for {
		lineNumber++
		fields, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNumber, err))
			continue
		}
		if lineNumber == 1 && strings.EqualFold(strings.TrimSpace(fields[0]), "id") {
			continue
		}
		if len(fields) < 6 {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: expected 6 fields", lineNumber))
			continue
		}
		record := domain.Record{ID: strings.TrimSpace(fields[0]), Line: strings.TrimSpace(fields[1]), Station: strings.TrimSpace(fields[2]), Machine: strings.TrimSpace(fields[3]), Summary: strings.TrimSpace(fields[5]), Status: domain.StatusNew}
		if fields[4] == "3" {
			record.Severity = domain.SeverityCritical
		} else if fields[4] == "2" {
			record.Severity = domain.SeverityWarning
		} else {
			record.Severity = domain.SeverityInfo
		}
		if err := domain.ValidateRecord(record); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNumber, err))
			continue
		}
		result.Records = append(result.Records, record)
	}
	return result
}

func (s *Service) Import(reader io.Reader, actor domain.User) ImportResult {
	parsed := ParseCSV(reader)
	accepted := make([]domain.Record, 0, len(parsed.Records))
	for _, record := range parsed.Records {
		created, err := s.Register(record, actor)
		if err != nil {
			parsed.Errors = append(parsed.Errors, fmt.Sprintf("%s: %v", record.ID, err))
			continue
		}
		accepted = append(accepted, created)
	}
	parsed.Records = accepted
	return parsed
}
