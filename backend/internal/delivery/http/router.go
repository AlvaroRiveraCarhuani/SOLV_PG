package httpdelivery

import (
	"net/http"
)

type Handlers struct {
	ServerPoliciesHandler    *ServerPoliciesHandler
	TenantLogoHandler        *TenantLogoHandler
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
	AdminAcademicHandler     *AdminAcademicHandler
	StudentHandler           *StudentHandler
	TeacherHandler           *TeacherHandler
	NotificationHandler      *NotificationHandler
	BackupHandler            *BackupHandler
	WebSocketHandler         *WebSocketHandler
	EnvTestHandler           *EnvTestHandler
	LanguageProfileHandler   *LanguageProfileHandler
	TenantMiddleware         func(http.Handler) http.Handler
	AuditMiddleware          func(http.Handler) http.Handler
	RateLimitMiddleware      func(http.Handler) http.Handler
	MaintenanceMiddleware    func(http.Handler) http.Handler
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps)
	registerTemplateRoutes(mux, deps.TemplateHandler)
	registerAuthRoutes(mux, deps.AuthHandler)
	registerEvaluationRoutes(mux, deps.EvaluationHandler, deps.TenantMiddleware, deps.AuditMiddleware)
	registerLanguageProfileRoutes(mux, deps)
	registerWorkspaceRoutes(mux, deps.WorkspaceHandler, deps.TenantMiddleware, deps.RateLimitMiddleware)
	registerMetricsRoutes(mux, deps.MetricsHandler)
	registerConfigRoutes(mux, deps.ConfigHandler)
	registerAcademicRoutes(mux, deps)
	registerAdminRoutes(mux, deps)
	registerStudentRoutes(mux, deps)
	registerTeacherRoutes(mux, deps)
	registerNotificationRoutes(mux, deps)
	registerBackupRoutes(mux, deps)
	registerServerPoliciesRoutes(mux, deps)
	registerTenantLogoRoutes(mux, deps)
	registerWebSocketRoutes(mux, deps.WebSocketHandler)
}

func registerServerPoliciesRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	if deps.ServerPoliciesHandler != nil {
		mux.Handle("GET /api/v1/admin/server/policies", tm(http.HandlerFunc(deps.ServerPoliciesHandler.GetPolicies)))
		mux.Handle("PUT /api/v1/admin/server/policies", am(tm(http.HandlerFunc(deps.ServerPoliciesHandler.UpdatePolicies))))
	}
}

// registerTenantLogoRoutes subida autenticada del imagotipo + servicio
// público cacheable del archivo almacenado.
func registerTenantLogoRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	if deps.TenantLogoHandler != nil {
		mux.Handle("POST /api/v1/tenants/logo", am(tm(http.HandlerFunc(deps.TenantLogoHandler.UploadLogo))))
		mux.Handle("GET /api/v1/public/branding/logo/{filename}", http.HandlerFunc(deps.TenantLogoHandler.ServePublicLogo))
	}
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
		mux.Handle("PUT /api/v1/subjects/{id}", am(tm(http.HandlerFunc(deps.SubjectHandler.UpdateSubject))))
		mux.Handle("PUT /api/v1/subjects/{id}/archive", am(tm(http.HandlerFunc(deps.SubjectHandler.ArchiveSubject))))
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
		mux.Handle("GET /api/v1/teachers", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.ListTeachers))))
		mux.Handle("GET /api/v1/teachers/{id}/courses", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.GetTeacherCourses))))
		mux.Handle("DELETE /api/v1/teachers/{id}", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.DeleteTeacher))))
		mux.Handle("POST /api/v1/invitations/teachers", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.CreateInvitation))))
		mux.Handle("POST /api/v1/invitations/teachers/{id}/resend", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.ResendInvitation))))
		mux.Handle("POST /api/v1/invitations/teachers/{id}/renew", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.RenewInvitation))))
		mux.Handle("DELETE /api/v1/invitations/teachers/{id}", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.DeleteInvitation))))
		mux.Handle("POST /api/v1/invitations/teachers/accept", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.AcceptInvitation))))
	}

	if deps.ClassroomHandler != nil {
		mux.Handle("GET /api/v1/classroom/import", tm(http.HandlerFunc(deps.ClassroomHandler.ImportRosterManual)))
	}
}

func registerAdminRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	if deps.AdminHandler != nil {
		mux.Handle("GET /api/v1/admin/audit-logs", tm(http.HandlerFunc(deps.AdminHandler.ListAuditLogs)))
		mux.Handle("GET /api/v1/admin/audit-logs/actors/{actorId}/timeline", tm(http.HandlerFunc(deps.AdminHandler.GetActorTimeline)))
		mux.Handle("PUT /api/v1/admin/branding", am(tm(http.HandlerFunc(deps.AdminHandler.UpdateBranding))))
		mux.Handle("GET /api/v1/admin/metrics/health", tm(http.HandlerFunc(deps.AdminHandler.GetHealthMetrics)))
		mux.Handle("GET /api/v1/admin/dashboard/courses-load", tm(http.HandlerFunc(deps.AdminHandler.GetCoursesLoad)))
		mux.Handle("GET /api/v1/admin/dashboard/incidents", tm(http.HandlerFunc(deps.AdminHandler.GetIncidents)))
		mux.Handle("GET /api/v1/admin/dashboard/containers", tm(http.HandlerFunc(deps.AdminHandler.GetContainers)))
		mux.Handle("GET /api/v1/admin/courses/{id}/workspaces", tm(http.HandlerFunc(deps.AdminHandler.GetCourseWorkspaces)))
		mux.Handle("GET /api/v1/admin/workspaces/{id}/logs", tm(http.HandlerFunc(deps.AdminHandler.GetWorkspaceLogs)))
		mux.Handle("POST /api/v1/admin/workspaces/{id}/resolve", am(tm(http.HandlerFunc(deps.AdminHandler.ResolveIncident))))
	}

	if deps.AdminAcademicHandler != nil {
		mux.Handle("POST /api/v1/admin/maintenance/enable", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.EnableMaintenance))))
		mux.Handle("POST /api/v1/admin/maintenance/disable", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DisableMaintenance))))
		mux.Handle("GET /api/v1/admin/maintenance/status", tm(http.HandlerFunc(deps.AdminAcademicHandler.GetMaintenanceStatus)))
		mux.Handle("GET /api/v1/admin/academic-periods", tm(http.HandlerFunc(deps.AdminAcademicHandler.ListPeriods)))
		mux.Handle("POST /api/v1/admin/academic-periods", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.CreatePeriod))))
		mux.Handle("PUT /api/v1/admin/academic-periods/{id}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.UpdatePeriod))))
		mux.Handle("POST /api/v1/admin/academic-periods/{id}/archive", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ArchivePeriod))))
		mux.Handle("DELETE /api/v1/admin/academic-periods/{id}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DeletePeriod))))
		mux.Handle("POST /api/v1/admin/courses/{id}/reassign", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ReassignCourse))))
		mux.Handle("GET /api/v1/admin/students", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ListStudents))))
		mux.Handle("POST /api/v1/admin/students", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.CreateStudent))))
		mux.Handle("GET /api/v1/admin/students/{id}/courses", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.GetStudentCourses))))
		mux.Handle("PUT /api/v1/admin/students/{id}/status", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.UpdateStudentStatus))))
		mux.Handle("POST /api/v1/admin/students/{id}/reset-oom", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ResetStudentOOM))))
		mux.Handle("GET /api/v1/admin/templates", tm(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplates)))
		mux.Handle("GET /api/v1/admin/templates/capabilities", tm(http.HandlerFunc(deps.AdminAcademicHandler.GetRuntimeCapabilities)))
		mux.Handle("POST /api/v1/admin/templates", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.CreateTemplate))))
		mux.Handle("POST /api/v1/admin/templates/drafts", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.SaveDraft))))
		mux.Handle("GET /api/v1/admin/templates/drafts", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.GetDraftByUser))))
		mux.Handle("DELETE /api/v1/admin/templates/drafts", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DeleteDraft))))
		mux.Handle("PUT /api/v1/admin/templates/{id}/review", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ReviewTemplate))))
		mux.Handle("GET /api/v1/admin/templates/local-images", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ListLocalImages))))
		mux.Handle("POST /api/v1/admin/templates/verify-image", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.VerifyImage))))
		mux.Handle("GET /api/v1/admin/templates/verify-image", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.VerifyImage))))
		mux.Handle("GET /api/v1/registry/verify", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.VerifyImage))))
		mux.Handle("POST /api/v1/admin/templates/{id}/duplicate", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DuplicateTemplate))))
		mux.Handle("POST /api/v1/plantillas/{id}/duplicar", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DuplicateTemplate))))
		mux.Handle("POST /api/v1/admin/templates/{id}/promote-to-model", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.PromoteTemplateToModel))))
		mux.Handle("GET /api/v1/admin/template-categories", tm(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplateCategories)))
		mux.Handle("GET /api/v1/admin/manual", tm(http.HandlerFunc(deps.AdminAcademicHandler.GetAdminManual)))
		mux.Handle("POST /api/v1/admin/template-categories", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.CreateTemplateCategory))))
		mux.Handle("PUT /api/v1/admin/template-categories/reorder", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ReorderTemplateCategories))))
		mux.Handle("PUT /api/v1/admin/template-categories/{id}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateCategory))))
		mux.Handle("DELETE /api/v1/admin/template-categories/{id}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DeleteTemplateCategory))))
		mux.Handle("GET /api/v1/admin/template-models", tm(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplateModels)))
		mux.Handle("PUT /api/v1/admin/template-models/{id}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateModel))))
		mux.Handle("POST /api/v1/admin/template-models/{id}/deactivate", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.DeactivateTemplateModel))))
		mux.Handle("POST /api/v1/admin/template-models/{id}/reactivate", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ReactivateTemplateModel))))
		mux.Handle("POST /api/v1/admin/emergency/{action}", am(tm(http.HandlerFunc(deps.AdminAcademicHandler.ExecuteEmergencyAction))))
	}

	if deps.EnvTestHandler != nil {
		mux.Handle("POST /api/v1/jobs/env-test", am(tm(http.HandlerFunc(deps.EnvTestHandler.StartEnvTest))))
		mux.Handle("GET /api/v1/jobs/env-test/{id}", tm(http.HandlerFunc(deps.EnvTestHandler.GetEnvTest)))
		mux.Handle("POST /api/v1/jobs/env-test/{id}/cancel", am(tm(http.HandlerFunc(deps.EnvTestHandler.CancelEnvTest))))
	}
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
	mux.Handle("POST /api/v1/exercises/{id}/dry-run", am(tm(http.HandlerFunc(h.StartDryRun))))
	mux.Handle("GET /api/v1/exercises/{id}/dry-run/jobs/{jobId}", tm(http.HandlerFunc(h.GetDryRunJob)))
	mux.Handle("GET /api/v1/dry-run/jobs/{jobId}", tm(http.HandlerFunc(h.GetDryRunJob)))
}

func registerLanguageProfileRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.LanguageProfileHandler == nil {
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

	mux.Handle("GET /api/v1/judge/profiles", http.HandlerFunc(deps.LanguageProfileHandler.ListProfiles))
	mux.Handle("GET /api/v1/judge/profiles/{language}", http.HandlerFunc(deps.LanguageProfileHandler.GetProfile))
	mux.Handle("PUT /api/v1/admin/judge/profiles/{language}", am(tm(http.HandlerFunc(deps.LanguageProfileHandler.UpdateProfile))))
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
	mux.Handle("GET /api/v1/teacher/courses/{id}/grades/matrix", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseGradesMatrix)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/grades/export", tm(http.HandlerFunc(deps.TeacherHandler.ExportGrades)))



	mux.Handle("GET /api/v1/teacher/submissions/{id}/timeline", tm(http.HandlerFunc(deps.TeacherHandler.GetTimeline)))
	mux.Handle("GET /api/v1/teacher/plagiarism", tm(http.HandlerFunc(deps.TeacherHandler.AnalyzePlagiarism)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/plagiarism", tm(http.HandlerFunc(deps.TeacherHandler.AnalyzePlagiarism)))
	mux.Handle("GET /api/v1/teacher/live-sessions", tm(http.HandlerFunc(deps.TeacherHandler.GetLiveSessions)))
	mux.Handle("POST /api/v1/teacher/live-sessions/{id}/exec", tm(http.HandlerFunc(deps.TeacherHandler.PostTutorExec)))
	mux.HandleFunc("GET /api/v1/teacher/live-sessions/{id}/terminal", deps.TeacherHandler.HandleTerminalWebSocket)
	mux.HandleFunc("GET /ws/v1/teacher/live-sessions/{id}/terminal", deps.TeacherHandler.HandleTerminalWebSocket)
	mux.Handle("POST /api/v1/teacher/exercises/generate-fuzz-cases", tm(http.HandlerFunc(deps.TeacherHandler.GenerateFuzzCases)))
	mux.Handle("POST /api/v1/teacher/exercises/{id}/apply-fuzz-cases", tm(http.HandlerFunc(deps.TeacherHandler.ApplyFuzzCases)))
}

func registerNotificationRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.NotificationHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	mux.Handle("GET /api/v1/notifications", tm(http.HandlerFunc(deps.NotificationHandler.List)))
	mux.Handle("GET /api/v1/notifications/unread-count", tm(http.HandlerFunc(deps.NotificationHandler.GetUnreadCount)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", tm(http.HandlerFunc(deps.NotificationHandler.MarkRead)))
	mux.Handle("POST /api/v1/notifications/mark-all-read", tm(http.HandlerFunc(deps.NotificationHandler.MarkAllRead)))
}

func registerBackupRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.BackupHandler == nil {
		return
	}
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	mux.Handle("GET /api/v1/admin/backups/config", tm(http.HandlerFunc(deps.BackupHandler.GetConfig)))
	mux.Handle("PUT /api/v1/admin/backups/config", tm(http.HandlerFunc(deps.BackupHandler.UpdateConfig)))
	mux.Handle("GET /api/v1/admin/backups", tm(http.HandlerFunc(deps.BackupHandler.List)))
	mux.Handle("POST /api/v1/admin/backups/trigger", tm(http.HandlerFunc(deps.BackupHandler.Trigger)))
	mux.Handle("POST /api/v1/admin/backups/{id}/verify", tm(http.HandlerFunc(deps.BackupHandler.Verify)))
	mux.Handle("GET /api/v1/admin/backups/{id}/download", tm(http.HandlerFunc(deps.BackupHandler.Download)))
}
