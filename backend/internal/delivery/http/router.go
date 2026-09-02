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
	AdminHandler             *AdminHandler
	StudentHandler           *StudentHandler
	TeacherHandler           *TeacherHandler
	WebSocketHandler         *WebSocketHandler
	TenantMiddleware         func(http.Handler) http.Handler
	AuditMiddleware          func(http.Handler) http.Handler
	RateLimitMiddleware      func(http.Handler) http.Handler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler, deps.TenantMiddleware, deps.AuditMiddleware)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler, deps.TenantMiddleware, deps.RateLimitMiddleware)
	registerMetricsRoutes(mux, deps.MetricsHandler)
	registerConfigRoutes(mux, deps.ConfigHandler)
	registerAcademicRoutes(mux, deps)
	registerAdminRoutes(mux, deps)
	registerStudentRoutes(mux, deps)
	registerTeacherRoutes(mux, deps)
	registerWebSocketRoutes(mux, deps.WebSocketHandler)
}

func registerAcademicRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	if deps.SubjectHandler != nil {
		mux.Handle("POST /api/v1/subjects", am(tm(http.HandlerFunc(deps.SubjectHandler.CreateSubject))))
		mux.Handle("GET /api/v1/subjects", tm(http.HandlerFunc(deps.SubjectHandler.ListSubjects)))
		mux.Handle("POST /api/v1/subjects/{id}/enroll", am(tm(http.HandlerFunc(deps.SubjectHandler.EnrollStudent))))
		mux.Handle("GET /api/v1/subjects/{id}/students", tm(http.HandlerFunc(deps.SubjectHandler.ListStudents)))
	}

	if deps.SubmissionHandler != nil {
		mux.Handle("POST /api/v1/submissions", am(tm(http.HandlerFunc(deps.SubmissionHandler.CreateSubmission))))
		mux.Handle("GET /api/v1/exercises/{id}/submissions", tm(http.HandlerFunc(deps.SubmissionHandler.ListSubmissionsByExercise)))
		mux.Handle("GET /api/v1/submissions/{id}", tm(http.HandlerFunc(deps.SubmissionHandler.GetSubmissionByID)))
		mux.Handle("POST /api/v1/submissions/{id}/override", am(tm(http.HandlerFunc(deps.SubmissionHandler.OverrideSubmission))))
	}

	if deps.TeacherInvitationHandler != nil {
		mux.Handle("POST /api/v1/invitations/teachers", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.CreateInvitation))))
		mux.Handle("POST /api/v1/invitations/teachers/accept", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.AcceptInvitation))))
	}

	if deps.ClassroomHandler != nil {
		mux.Handle("GET /api/v1/classroom/import", tm(http.HandlerFunc(deps.ClassroomHandler.ImportRosterManual)))
	}
}

func registerAdminRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.AdminHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	mux.Handle("GET /api/v1/admin/audit-logs", tm(http.HandlerFunc(deps.AdminHandler.ListAuditLogs)))
	mux.Handle("PUT /api/v1/admin/branding", am(tm(http.HandlerFunc(deps.AdminHandler.UpdateBranding))))
	mux.Handle("GET /api/v1/admin/metrics/health", tm(http.HandlerFunc(deps.AdminHandler.GetHealthMetrics)))
}

func registerStudentRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.StudentHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	mux.Handle("GET /api/v1/student/dashboard", tm(http.HandlerFunc(deps.StudentHandler.GetDashboard)))
	mux.Handle("GET /api/v1/student/assignments/due", tm(http.HandlerFunc(deps.StudentHandler.GetDueAssignments)))
}

func registerUserRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.UserHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	mux.HandleFunc("POST /api/v1/users", deps.UserHandler.Create)
	mux.Handle("GET /api/v1/users/me", tm(http.HandlerFunc(deps.UserHandler.GetMe)))
}

func registerTemplateRoutes(mux *http.ServeMux, h *TemplateHandler) {
	if h != nil {
		mux.HandleFunc("POST /api/v1/templates", h.Create)
		mux.HandleFunc("GET /api/v1/templates", h.GetAll)
	}
}

func registerAuthRoutes(mux *http.ServeMux, h *AuthHandler) {
	if h != nil {
		mux.HandleFunc("GET /auth/google/login", h.HandleGoogleLogin)
		mux.HandleFunc("GET /auth/google/callback", h.HandleGoogleCallback)
		mux.HandleFunc("GET /api/v1/auth/verify", h.VerifyAuth)
		mux.HandleFunc("POST /api/v1/auth/logout", h.HandleLogout)
	}
}

func registerEvaluationRoutes(mux *http.ServeMux, h *EvaluationHandler, tenantMiddleware func(http.Handler) http.Handler, auditMiddleware func(http.Handler) http.Handler) {
	if h == nil {
		return
	}
	tm := tenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := auditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	mux.Handle("POST /api/v1/evaluations", tm(http.HandlerFunc(h.Evaluate)))
	mux.Handle("GET /api/v1/exercises/{id}", tm(http.HandlerFunc(h.GetExerciseByID)))
	mux.Handle("POST /api/v1/exercises", am(tm(http.HandlerFunc(h.CreateExercise))))
	mux.Handle("PUT /api/v1/exercises/{id}", am(tm(http.HandlerFunc(h.UpdateExercise))))
	mux.Handle("POST /api/v1/exercises/{id}/test-cases/bulk", am(tm(http.HandlerFunc(h.BulkTestCases))))
	mux.Handle("POST /api/v1/exercises/{id}/publish", am(tm(http.HandlerFunc(h.PublishExercise))))
}

func registerWorkspaceRoutes(mux *http.ServeMux, h *WorkspaceHandler, tenantMiddleware func(http.Handler) http.Handler, rateLimitMiddleware func(http.Handler) http.Handler) {
	if h == nil {
		return
	}
	if rateLimitMiddleware == nil {
		rateLimitMiddleware = func(next http.Handler) http.Handler { return next }
	}
	if tenantMiddleware != nil {
		mux.Handle("POST /api/v1/workspaces/start", tenantMiddleware(rateLimitMiddleware(http.HandlerFunc(h.StartWorkspace))))
		mux.Handle("POST /api/v1/workspaces/{id}/pause", tenantMiddleware(http.HandlerFunc(h.PauseWorkspace)))
		mux.Handle("DELETE /api/v1/workspaces/{id}", tenantMiddleware(http.HandlerFunc(h.TerminateWorkspace)))
		mux.Handle("GET /api/v1/workspaces/{id}/audit", tenantMiddleware(http.HandlerFunc(h.GetSemgrepAudit)))
		mux.Handle("POST /api/v1/workspaces/{id}/heartbeat", tenantMiddleware(http.HandlerFunc(h.Heartbeat)))
		mux.Handle("POST /api/v1/workspaces/{id}/restart", tenantMiddleware(http.HandlerFunc(h.RestartWorkspace)))
	} else {
		mux.Handle("POST /api/v1/workspaces/start", WithAuth(rateLimitMiddleware(http.HandlerFunc(h.StartWorkspace))))
		mux.Handle("POST /api/v1/workspaces/{id}/pause", WithAuth(http.HandlerFunc(h.PauseWorkspace)))
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

func registerWebSocketRoutes(mux *http.ServeMux, h *WebSocketHandler) {
	if h != nil {
		mux.HandleFunc("GET /ws/v1/evaluations", h.HandleEvaluationWS)
		mux.HandleFunc("GET /api/v1/ws/evaluations", h.HandleEvaluationWS)
	}
}

func registerTeacherRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.TeacherHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	mux.Handle("GET /api/v1/teacher/courses", tm(http.HandlerFunc(deps.TeacherHandler.GetCourses)))
	mux.Handle("GET /api/v1/teacher/attention", tm(http.HandlerFunc(deps.TeacherHandler.GetAttention)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/labs", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseLabs)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/submissions", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseSubmissions)))
	mux.Handle("GET /api/v1/teacher/submissions/{id}/review", tm(http.HandlerFunc(deps.TeacherHandler.GetSubmissionReview)))
	mux.Handle("POST /api/v1/teacher/submissions/{id}/comments", am(tm(http.HandlerFunc(deps.TeacherHandler.AddSubmissionComment))))
	mux.Handle("GET /api/v1/teacher/submissions/{id}/comments", tm(http.HandlerFunc(deps.TeacherHandler.GetSubmissionComments)))
	mux.Handle("POST /api/v1/teacher/submissions/{id}/run-ephemeral", tm(http.HandlerFunc(deps.TeacherHandler.RunEphemeral)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/grades/export", tm(http.HandlerFunc(deps.TeacherHandler.ExportGrades)))
}
