package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"

	"github.com/behzod/pageSDK/internal/application"
	"github.com/behzod/pageSDK/internal/domain"
)

var validate = validator.New()

// PageHandler обрабатывает GET /pages/{page_id}.
type PageHandler struct {
	pageSvc *application.PageService
}

func NewPageHandler(svc *application.PageService) *PageHandler {
	return &PageHandler{pageSvc: svc}
}

func (h *PageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageID := vars["page_id"]
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "SESSION_REQUIRED", "session_id query param is required")
		return
	}

	page, err := h.pageSvc.LoadPage(r.Context(), pageID, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PAGE_NOT_FOUND", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"page_id":    page.PageID,
		"containers": page.Containers,
		"state":      page.State,
	})
}

// EventHandler обрабатывает POST /events.
type EventHandler struct {
	processor *application.EventProcessor
}

func NewEventHandler(p *application.EventProcessor) *EventHandler {
	return &EventHandler{processor: p}
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var event domain.UIEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if err := validate.Struct(event); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.processor.Process(r.Context(), event)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "PROCESSING_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// HealthHandler — GET /health
type HealthHandler struct{}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, domain.AppError{Code: code, Message: msg})
}
