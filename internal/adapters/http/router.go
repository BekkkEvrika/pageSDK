package http

import (
	"github.com/behzod/pageSDK/internal/application"
	"github.com/gorilla/mux"
	"net/http"
)

// NewRouter собирает маршруты приложения.
func NewRouter(pageSvc *application.PageService, processor *application.EventProcessor) http.Handler {
	r := mux.NewRouter()
	r.Handle("/health", &HealthHandler{}).Methods(http.MethodGet)
	r.Handle("/pages/{page_id}", NewPageHandler(pageSvc)).Methods(http.MethodGet)
	r.Handle("/events", NewEventHandler(processor)).Methods(http.MethodPost)
	return r
}
