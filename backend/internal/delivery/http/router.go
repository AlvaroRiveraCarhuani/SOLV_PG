package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	UserHandler              *UserHandler
	TemplateHandler          *TemplateHandler
	AuthHandler              *AuthHandler
	EvaluationHandler        *EvaluationHandler
	WorkspaceHandler         *WorkspaceHandler
	MetricsHandler           *MetricsHandler
	ConfigHandler            *ConfigHandler
	SubjectHandler           *SubjectHandler
	SubmissionHandler        *SubmissionHandler
	TeacherInvitationHandler *TeacherInvitationHandler
	ClassroomHandler         *ClassroomHandler
	TenantMiddleware         func(http.Handler) http.Handler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps.UserHandler)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler, deps.TenantMiddleware)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler, deps.TenantMiddleware)
	registerMetricsRoutes(mux, deps.MetricsHandler)
	registerConfigRoutes(mux, deps.ConfigHandler)
	registerAcademicRoutes(mux, deps)
}

func registerAcademicRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	if deps.SubjectHandler != nil {
		mux.Handle("POST /api/v1/subjects", tm(http.HandlerFunc(deps.SubjectHandler.CreateSubject)))
		mux.Handle("GET /api/v1/subjects", tm(http.HandlerFunc(deps.SubjectHandler.ListSubjects)))
		mux.Handle("POST /api/v1/subjects/{id}/enroll", tm(http.HandlerFunc(deps.SubjectHandler.EnrollStudent)))
		mux.Handle("GET /api/v1/subjects/{id}/students", tm(http.HandlerFunc(deps.SubjectHandler.ListStudents)))
	}

	if deps.SubmissionHandler != nil {
		mux.Handle("POST /api/v1/submissions", tm(http.HandlerFunc(deps.SubmissionHandler.CreateSubmission)))
		mux.Handle("GET /api/v1/exercises/{id}/submissions", tm(http.HandlerFunc(deps.SubmissionHandler.ListSubmissionsByExercise)))
	}

	if deps.TeacherInvitationHandler != nil {
		mux.Handle("POST /api/v1/invitations/teachers", tm(http.HandlerFunc(deps.TeacherInvitationHandler.CreateInvitation)))
		mux.Handle("POST /api/v1/invitations/teachers/accept", tm(http.HandlerFunc(deps.TeacherInvitationHandler.AcceptInvitation)))
	}

	if deps.ClassroomHandler != nil {
		mux.Handle("GET /api/v1/classroom/import", tm(http.HandlerFunc(deps.ClassroomHandler.ImportRosterManual)))
	}
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
