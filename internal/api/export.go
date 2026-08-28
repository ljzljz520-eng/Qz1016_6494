package api

import (
	"encoding/json"
	"net/http"
	"production43/internal/domain"
	"time"
)

type Export struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Records     []domain.Record `json:"records"`
	Format      string          `json:"format"`
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, errMethod)
		return
	}
	records, err := h.service.List(domain.RecordFilter{})
	if err != nil {
		writeError(w, 500, err)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format == "json" {
		writeJSON(w, 200, Export{GeneratedAt: time.Now().UTC(), Records: records, Format: format})
		return
	}
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(200)
		w.Write([]byte("id,line,station,severity,status,summary\n"))
		for _, record := range records {
			w.Write([]byte(record.ID + "," + record.Line + "," + record.Station + "," + string(rune('0'+record.Severity)) + "," + string(record.Status) + "," + record.Summary + "\n"))
		}
		return
	}
	writeError(w, 400, errFormat)
}

var errMethod = errorString("method not allowed")
var errFormat = errorString("unsupported export format")

type errorString string

func (e errorString) Error() string { return string(e) }
func EncodeExport(records []domain.Record) ([]byte, error) {
	return json.Marshal(Export{GeneratedAt: time.Now().UTC(), Records: records, Format: "json"})
}
