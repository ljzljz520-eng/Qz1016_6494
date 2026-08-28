package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"production43/internal/domain"
	"production43/internal/service"
)

type Handler struct {
	service *service.Service
	maxBody int64
}

func NewHandler(svc *service.Service, maxBody int64) *Handler {
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	return &Handler{service: svc, maxBody: maxBody}
}
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/api/v1/alerts", h.alerts)
	mux.HandleFunc("/api/v1/alerts/", h.alertByID)
	mux.HandleFunc("/api/v1/dashboard", h.dashboard)
	mux.HandleFunc("/api/v1/export", h.export)
	return mux
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) alerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := domain.RecordFilter{Line: r.URL.Query().Get("line"), OwnerID: r.URL.Query().Get("owner"), Label: r.URL.Query().Get("label")}
		if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
			filter.Limit = limit
		}
		records, err := h.service.List(filter)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, records)
	case http.MethodPost:
		var request struct {
			domain.Record
			Actor domain.User
		}
		r.Body = http.MaxBytesReader(w, r.Body, h.maxBody)
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, 400, err)
			return
		}
		record, err := h.service.Register(request.Record, request.Actor)
		if err != nil {
			writeError(w, 422, err)
			return
		}
		writeJSON(w, 201, record)
	default:
		writeError(w, 405, errors.New("method not allowed"))
	}
}
func (h *Handler) alertByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, 404, errors.New("alert id required"))
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		record, err := h.service.Get(id)
		if err != nil {
			writeError(w, 404, err)
			return
		}
		writeJSON(w, 200, record)
		return
	}
	if len(parts) == 2 && parts[1] == "timeline" && r.Method == http.MethodGet {
		timeline, err := h.service.Timeline(id)
		if err != nil {
			writeError(w, 404, err)
			return
		}
		writeJSON(w, 200, timeline)
		return
	}
	if len(parts) == 2 && parts[1] == "process" && r.Method == http.MethodPost {
		var request struct {
			Actor   domain.User   `json:"actor"`
			Action  domain.Action `json:"action"`
			Note    string        `json:"note"`
			Version int           `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, 400, err)
			return
		}
		record, err := h.service.Process(id, request.Actor, request.Action, request.Note, request.Version)
		if err != nil {
			writeError(w, 422, err)
			return
		}
		writeJSON(w, 200, record)
		return
	}
	writeError(w, 404, errors.New("route not found"))
}
func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, errors.New("method not allowed"))
		return
	}
	records, err := h.service.List(domain.RecordFilter{})
	if err != nil {
		writeError(w, 500, err)
		return
	}
	now := time.Now().UTC()
	writeJSON(w, 200, map[string]any{"generated_at": now, "total": len(records), "records": records})
}
func decodeBody(r *http.Request, target any, max int64) error {
	r.Body = http.MaxBytesReader(nil, r.Body, max)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
