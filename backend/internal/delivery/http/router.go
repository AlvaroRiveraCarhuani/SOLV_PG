package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	UserHandler     *UserHandler
	TemplateHandler *TemplateHandler
	LabHandler      *LabHandler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerLabRoutes(mux, deps.LabHandler)
}

func registerUserRoutes(mux *http.ServeMux, h *UserHandler) {
	mux.HandleFunc("POST /api/v1/users", h.Create)
}

func registerTemplateRoutes(mux *http.ServeMux, h *TemplateHandler) {
	mux.HandleFunc("POST /api/v1/templates", h.Create)
	mux.HandleFunc("GET /api/v1/templates", h.GetAll)
}

func registerLabRoutes(mux *http.ServeMux, h *LabHandler) {
	mux.HandleFunc("POST /api/v1/labs/start", h.Start)
}
