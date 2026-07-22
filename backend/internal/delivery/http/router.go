package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	UserHandler       *UserHandler
	TemplateHandler   *TemplateHandler
	LabHandler        *LabHandler
	AuthHandler       *AuthHandler
	EvaluationHandler *EvaluationHandler
	WorkspaceHandler  *WorkspaceHandler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerLabRoutes(mux, deps.LabHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler)
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
	mux.Handle("DELETE /api/v1/labs/{id}", WithAuth(http.HandlerFunc(h.Delete)))
}

func registerAuthRoutes(mux *http.ServeMux, h *AuthHandler) {
	mux.HandleFunc("GET /auth/google/login", h.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)
}

func registerEvaluationRoutes(mux *http.ServeMux, h *EvaluationHandler) {
	mux.Handle("POST /api/v1/evaluations", WithAuth(http.HandlerFunc(h.Evaluate)))
}

func registerWorkspaceRoutes(mux *http.ServeMux, h *WorkspaceHandler) {
	mux.Handle("POST /api/v1/workspaces/start", WithAuth(http.HandlerFunc(h.StartWorkspace)))
	mux.Handle("POST /api/v1/workspaces/{id}/heartbeat", WithAuth(http.HandlerFunc(h.Heartbeat)))
	mux.Handle("POST /api/v1/workspaces/{id}/restart", WithAuth(http.HandlerFunc(h.RestartWorkspace)))
}
