package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	UserHandler       *UserHandler
	TemplateHandler   *TemplateHandler
	AuthHandler       *AuthHandler
	EvaluationHandler *EvaluationHandler
	WorkspaceHandler  *WorkspaceHandler
	MetricsHandler    *MetricsHandler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler)
	registerMetricsRoutes(mux, deps.MetricsHandler)
}

func registerUserRoutes(mux *http.ServeMux, h *UserHandler) {
	mux.HandleFunc("POST /api/v1/users", h.Create)
}

func registerTemplateRoutes(mux *http.ServeMux, h *TemplateHandler) {
	mux.HandleFunc("POST /api/v1/templates", h.Create)
	mux.HandleFunc("GET /api/v1/templates", h.GetAll)
}

func registerAuthRoutes(mux *http.ServeMux, h *AuthHandler) {
	mux.HandleFunc("GET /auth/google/login", h.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)
	mux.HandleFunc("GET /api/v1/auth/verify", h.VerifyAuth)
	mux.HandleFunc("POST /api/v1/auth/logout", h.HandleLogout)
}

func registerEvaluationRoutes(mux *http.ServeMux, h *EvaluationHandler) {
	mux.Handle("POST /api/v1/evaluations", WithAuth(http.HandlerFunc(h.Evaluate)))
}

func registerWorkspaceRoutes(mux *http.ServeMux, h *WorkspaceHandler) {
	mux.Handle("POST /api/v1/workspaces/start", WithAuth(http.HandlerFunc(h.StartWorkspace)))
	mux.Handle("DELETE /api/v1/workspaces/{id}", WithAuth(http.HandlerFunc(h.TerminateWorkspace)))
	mux.Handle("GET /api/v1/workspaces/{id}/audit", WithAuth(http.HandlerFunc(h.GetSemgrepAudit)))
	mux.Handle("POST /api/v1/workspaces/{id}/heartbeat", WithAuth(http.HandlerFunc(h.Heartbeat)))
	mux.Handle("POST /api/v1/workspaces/{id}/restart", WithAuth(http.HandlerFunc(h.RestartWorkspace)))
}

func registerMetricsRoutes(mux *http.ServeMux, h *MetricsHandler) {
	if h != nil {
		mux.HandleFunc("GET /metrics", h.HandleMetrics)
	}
}
