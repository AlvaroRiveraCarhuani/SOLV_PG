package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	UserHandler     *UserHandler
	TemplateHandler *TemplateHandler
	LabHandler      *LabHandler
	AuthHandler     *AuthHandler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerLabRoutes(mux, deps.LabHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
}

func registerUserRoutes(mux *http.ServeMux, h *UserHandler) {
	mux.HandleFunc("POST /api/v1/users", h.Create)
}

func registerTemplateRoutes(mux *http.ServeMux, h *TemplateHandler) {
	mux.HandleFunc("POST /api/v1/templates", h.Create)
	mux.HandleFunc("GET /api/v1/templates", h.GetAll)
}

func registerLabRoutes(mux *http.ServeMux, h *LabHandler) {
	mux.Handle("POST /api/v1/labs/start", WithAuth(http.HandlerFunc(h.Start)))
}

func registerAuthRoutes(mux *http.ServeMux, h *AuthHandler) {
	mux.HandleFunc("GET /auth/google/login", h.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)
}
