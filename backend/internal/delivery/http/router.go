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
	ConfigHandler     *ConfigHandler
	TenantMiddleware  func(http.Handler) http.Handler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler, deps.TenantMiddleware)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler, deps.TenantMiddleware)
	registerMetricsRoutes(mux, deps.MetricsHandler)
	registerConfigRoutes(mux, deps.ConfigHandler)
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

func registerEvaluationRoutes(mux *http.ServeMux, h *EvaluationHandler, tenantMiddleware func(http.Handler) http.Handler) {
	if tenantMiddleware != nil {
		mux.Handle("POST /api/v1/evaluations", tenantMiddleware(http.HandlerFunc(h.Evaluate)))
	} else {
		mux.Handle("POST /api/v1/evaluations", WithAuth(http.HandlerFunc(h.Evaluate)))
	}
}

func registerWorkspaceRoutes(mux *http.ServeMux, h *WorkspaceHandler, tenantMiddleware func(http.Handler) http.Handler) {
	if tenantMiddleware != nil {
		mux.Handle("POST /api/v1/workspaces/start", tenantMiddleware(http.HandlerFunc(h.StartWorkspace)))
		mux.Handle("DELETE /api/v1/workspaces/{id}", tenantMiddleware(http.HandlerFunc(h.TerminateWorkspace)))
		mux.Handle("GET /api/v1/workspaces/{id}/audit", tenantMiddleware(http.HandlerFunc(h.GetSemgrepAudit)))
		mux.Handle("POST /api/v1/workspaces/{id}/heartbeat", tenantMiddleware(http.HandlerFunc(h.Heartbeat)))
		mux.Handle("POST /api/v1/workspaces/{id}/restart", tenantMiddleware(http.HandlerFunc(h.RestartWorkspace)))
	} else {
		mux.Handle("POST /api/v1/workspaces/start", WithAuth(http.HandlerFunc(h.StartWorkspace)))
		mux.Handle("DELETE /api/v1/workspaces/{id}", WithAuth(http.HandlerFunc(h.TerminateWorkspace)))
		mux.Handle("GET /api/v1/workspaces/{id}/audit", WithAuth(http.HandlerFunc(h.GetSemgrepAudit)))
		mux.Handle("POST /api/v1/workspaces/{id}/heartbeat", WithAuth(http.HandlerFunc(h.Heartbeat)))
		mux.Handle("POST /api/v1/workspaces/{id}/restart", WithAuth(http.HandlerFunc(h.RestartWorkspace)))
	}
}

func registerMetricsRoutes(mux *http.ServeMux, h *MetricsHandler) {
	if h != nil {
		mux.HandleFunc("GET /metrics", h.HandleMetrics)
	}
}

func registerConfigRoutes(mux *http.ServeMux, h *ConfigHandler) {
	if h != nil {
		mux.HandleFunc("GET /api/v1/config/public", h.GetPublicConfig)
	}
}
