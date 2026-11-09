package httpdelivery

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"solv-backend/internal/core/services"
)

type Handlers struct {
	ServerPoliciesHandler         *ServerPoliciesHandler
	AISettingsHandler             *AISettingsHandler
	AICopilotHandler              *AICopilotHandler
	TenantLogoHandler             *TenantLogoHandler
	UserHandler                   *UserHandler
	TemplateHandler               *TemplateHandler
	AuthHandler                   *AuthHandler
	EvaluationHandler             *EvaluationHandler
	PracticeEnvironmentHandler    *PracticeEnvironmentHandler
	MetricsHandler                *MetricsHandler
	ConfigHandler                 *ConfigHandler
	SubjectHandler                *SubjectHandler
	SubmissionHandler             *SubmissionHandler
	TeacherInvitationHandler      *TeacherInvitationHandler
	ClassroomHandler              *ClassroomHandler
	ClassSessionHandler           *ClassSessionHandler
	ClassAdvanceHandler           *ClassAdvanceHandler
	AdminHandler                  *AdminHandler
	AdminAcademicHandler          *AdminAcademicHandler
	StudentHandler                *StudentHandler
	TeacherHandler                *TeacherHandler
	NotificationHandler           *NotificationHandler
	BackupHandler                 *BackupHandler
	WebSocketHandler              *WebSocketHandler
	EnvTestHandler                *EnvTestHandler
	EnvironmentRequestHandler     *EnvironmentRequestHandler
	LanguageProfileHandler        *LanguageProfileHandler
	CourseModuleHandler           *CourseModuleHandler
	RubricHandler                 *RubricHandler
	TenantMiddleware              func(http.Handler) http.Handler
	AuditMiddleware               func(http.Handler) http.Handler
	RateLimitMiddleware           func(http.Handler) http.Handler
	LoginRateLimitMiddleware      func(http.Handler) http.Handler
	SubmissionRateLimitMiddleware func(http.Handler) http.Handler
	KeystrokeRateLimitMiddleware  func(http.Handler) http.Handler
	MaintenanceMiddleware         func(http.Handler) http.Handler
	TeacherMetricsSnapshotService *services.MetricsSnapshotService
}

func SetupRoutes(mux *http.ServeMux, deps *Handlers) {
	registerUserRoutes(mux, deps)
	registerTemplateRoutes(mux, deps)
	registerAuthRoutes(mux, deps)
	registerEvaluationRoutes(mux, deps.EvaluationHandler, deps.TenantMiddleware, deps.AuditMiddleware)
	registerLanguageProfileRoutes(mux, deps)
	registerPracticeEnvironmentRoutes(mux, deps.PracticeEnvironmentHandler, deps.TenantMiddleware, deps.RateLimitMiddleware)
	registerMetricsRoutes(mux, deps.MetricsHandler)
	registerConfigRoutes(mux, deps.ConfigHandler)
	registerAcademicRoutes(mux, deps)
	registerAdminRoutes(mux, deps)
	registerStudentRoutes(mux, deps)
	registerTeacherRoutes(mux, deps)
	registerEnvironmentRequestRoutes(mux, deps)
	registerClassSessionRoutes(mux, deps)
	registerClassAdvanceRoutes(mux, deps)
	registerCourseModuleRoutes(mux, deps)
	registerRubricRoutes(mux, deps)
	registerNotificationRoutes(mux, deps)
	registerBackupRoutes(mux, deps)
	registerServerPoliciesRoutes(mux, deps)
	registerAIRoutes(mux, deps)
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
	admin := func(next http.Handler) http.Handler { return tm(adminOnly(next)) }
	adminAudited := func(next http.Handler) http.Handler { return tm(am(adminOnly(next))) }

	if deps.ServerPoliciesHandler != nil {
		mux.Handle("GET /api/v1/admin/server/policies", admin(http.HandlerFunc(deps.ServerPoliciesHandler.GetPolicies)))
		mux.Handle("PUT /api/v1/admin/server/policies", adminAudited(http.HandlerFunc(deps.ServerPoliciesHandler.UpdatePolicies)))
	}
}

// registerAIRoutes configura el módulo de Inteligencia Artificial Híbrida:
// settings y prueba de conexión son de administración institucional; el estado
// liviano queda disponible para cualquier usuario autenticado del tenant y la
// cascada personal BYOK (/api/v1/ai/my-settings) para docentes y administradores.
func registerAIRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	if deps.AISettingsHandler != nil {
		admin := func(next http.Handler) http.Handler { return tm(adminOnly(next)) }
		adminAudited := func(next http.Handler) http.Handler { return tm(am(adminOnly(next))) }
		mux.Handle("GET /api/v1/admin/ai/settings", admin(http.HandlerFunc(deps.AISettingsHandler.GetSettings)))
		mux.Handle("PUT /api/v1/admin/ai/settings", adminAudited(http.HandlerFunc(deps.AISettingsHandler.UpdateSettings)))
		mux.Handle("POST /api/v1/admin/ai/test-connection", adminAudited(http.HandlerFunc(deps.AISettingsHandler.TestConnection)))
		mux.Handle("POST /api/v1/admin/ai/models", adminAudited(http.HandlerFunc(deps.AISettingsHandler.ListModels)))
		mux.Handle("GET /api/v1/ai/status", tm(http.HandlerFunc(deps.AISettingsHandler.Status)))
		mux.Handle("GET /api/v1/ai/my-settings", tm(http.HandlerFunc(deps.AISettingsHandler.GetMySettings)))
		mux.Handle("PUT /api/v1/ai/my-settings", am(tm(http.HandlerFunc(deps.AISettingsHandler.UpdateMySettings))))
		mux.Handle("POST /api/v1/ai/my-settings/test-connection", am(tm(http.HandlerFunc(deps.AISettingsHandler.TestMyConnection))))
		mux.Handle("POST /api/v1/ai/my-settings/models", am(tm(http.HandlerFunc(deps.AISettingsHandler.ListMyModels))))
	}

	if deps.AICopilotHandler != nil {
		adminAudited := func(next http.Handler) http.Handler { return tm(am(adminOnly(next))) }
		mux.Handle("POST /api/v1/admin/ai/suggest-template", adminAudited(http.HandlerFunc(deps.AICopilotHandler.SuggestTemplate)))
		mux.Handle("POST /api/v1/teacher/ai/suggest-environment", am(tm(http.HandlerFunc(deps.AICopilotHandler.SuggestEnvironmentForTeacher))))
		mux.Handle("POST /api/v1/admin/ai/suggest-component", adminAudited(http.HandlerFunc(deps.AICopilotHandler.SuggestComponent)))
		mux.Handle("POST /api/v1/admin/ai/repair-template", adminAudited(http.HandlerFunc(deps.AICopilotHandler.RepairTemplate)))
		mux.Handle("POST /api/v1/teacher/ai/generate-exercise", am(tm(http.HandlerFunc(deps.AICopilotHandler.GenerateExerciseDraft))))
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
		mux.Handle("POST /api/v1/tenants/logo", tm(am(adminOnly(http.HandlerFunc(deps.TenantLogoHandler.UploadLogo)))))
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
		srl := deps.SubmissionRateLimitMiddleware
		if srl == nil {
			srl = func(next http.Handler) http.Handler { return next }
		}
		krl := deps.KeystrokeRateLimitMiddleware
		if krl == nil {
			krl = func(next http.Handler) http.Handler { return next }
		}
		mux.Handle("POST /api/v1/submissions", am(tm(srl(http.HandlerFunc(deps.SubmissionHandler.CreateSubmission)))))
		mux.Handle("GET /api/v1/exercises/{id}/submissions", tm(http.HandlerFunc(deps.SubmissionHandler.ListSubmissionsByExercise)))
		mux.Handle("GET /api/v1/submissions/{id}", tm(http.HandlerFunc(deps.SubmissionHandler.GetSubmissionByID)))
		mux.Handle("POST /api/v1/submissions/{id}/override", am(tm(http.HandlerFunc(deps.SubmissionHandler.OverrideSubmission))))
		mux.Handle("POST /api/v1/submissions/{id}/keystroke-events", tm(krl(http.HandlerFunc(deps.SubmissionHandler.SaveKeystrokeEvents))))
	}

	if deps.TeacherInvitationHandler != nil {
		mux.Handle("GET /api/v1/teachers", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.ListTeachers))))
		mux.Handle("GET /api/v1/teachers/{id}/courses", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.GetTeacherCourses))))
		mux.Handle("DELETE /api/v1/teachers/{id}", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.DeleteTeacher))))
		mux.Handle("POST /api/v1/teachers/direct", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.RegisterDirectTeacher))))
		mux.Handle("POST /api/v1/teachers/parse-roster", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.ParseRoster))))
		mux.Handle("POST /api/v1/teachers/direct-batch", am(tm(http.HandlerFunc(deps.TeacherInvitationHandler.BatchRegisterTeachers))))
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

	admin := func(next http.Handler) http.Handler { return tm(adminOnly(next)) }
	adminAudited := func(next http.Handler) http.Handler { return tm(am(adminOnly(next))) }
	staff := func(next http.Handler) http.Handler { return tm(staffOnly(next)) }

	if deps.AdminHandler != nil {
		mux.Handle("GET /api/v1/admin/audit-logs", admin(http.HandlerFunc(deps.AdminHandler.ListAuditLogs)))
		mux.Handle("GET /api/v1/admin/audit-logs/actors/{actorId}/timeline", admin(http.HandlerFunc(deps.AdminHandler.GetActorTimeline)))
		mux.Handle("PUT /api/v1/admin/branding", adminAudited(http.HandlerFunc(deps.AdminHandler.UpdateBranding)))
		mux.Handle("GET /api/v1/admin/metrics/health", admin(http.HandlerFunc(deps.AdminHandler.GetHealthMetrics)))
		mux.Handle("GET /api/v1/admin/dashboard/courses-load", admin(http.HandlerFunc(deps.AdminHandler.GetCoursesLoad)))
		mux.Handle("GET /api/v1/admin/dashboard/incidents", admin(http.HandlerFunc(deps.AdminHandler.GetIncidents)))
		mux.Handle("GET /api/v1/admin/dashboard/containers", admin(http.HandlerFunc(deps.AdminHandler.GetContainers)))
		mux.Handle("GET /api/v1/admin/courses/{id}/environments", admin(http.HandlerFunc(deps.AdminHandler.GetCoursePracticeEnvironments)))
		mux.Handle("GET /api/v1/admin/environments/{id}/logs", admin(http.HandlerFunc(deps.AdminHandler.GetPracticeEnvironmentLogs)))
		mux.Handle("POST /api/v1/admin/environments/{id}/resolve", adminAudited(http.HandlerFunc(deps.AdminHandler.ResolveIncident)))
	}

	if deps.AdminAcademicHandler != nil {
		mux.Handle("POST /api/v1/admin/maintenance/enable", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.EnableMaintenance)))
		mux.Handle("POST /api/v1/admin/maintenance/disable", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DisableMaintenance)))
		mux.Handle("GET /api/v1/admin/maintenance/status", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetMaintenanceStatus)))
		mux.Handle("GET /api/v1/admin/academic-periods", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListPeriods)))
		mux.Handle("GET /api/v1/academic-periods", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListPeriods)))
		mux.Handle("POST /api/v1/admin/academic-periods", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreatePeriod)))
		mux.Handle("PUT /api/v1/admin/academic-periods/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdatePeriod)))
		mux.Handle("POST /api/v1/admin/academic-periods/{id}/archive", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ArchivePeriod)))
		mux.Handle("DELETE /api/v1/admin/academic-periods/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DeletePeriod)))
		mux.Handle("POST /api/v1/admin/courses/{id}/reassign", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ReassignCourse)))
		mux.Handle("GET /api/v1/admin/students", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListStudents)))
		mux.Handle("POST /api/v1/admin/students", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreateStudent)))
		mux.Handle("GET /api/v1/admin/students/{id}/courses", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetStudentCourses)))
		mux.Handle("PUT /api/v1/admin/students/{id}/status", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateStudentStatus)))
		mux.Handle("POST /api/v1/admin/students/{id}/reset-oom", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ResetStudentOOM)))
		mux.Handle("POST /api/v1/admin/students/parse-roster", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ParseStudentRoster)))
		mux.Handle("POST /api/v1/admin/students/batch", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.BatchCreateStudents)))
		mux.Handle("POST /api/v1/admin/subjects/{id}/enroll-batch", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.EnrollStudentsBatch)))
		mux.Handle("POST /api/v1/admin/subjects/{id}/roster", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.RosterEnroll)))
		mux.Handle("GET /api/v1/admin/administrators", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListAdministrators)))
		mux.Handle("POST /api/v1/admin/administrators", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreateAdministrator)))
		mux.Handle("POST /api/v1/admin/administrators/{id}/revoke", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.RevokeAdministrator)))
		mux.Handle("GET /api/v1/admin/templates", staff(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplates)))
		mux.Handle("GET /api/v1/admin/templates/capabilities", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetRuntimeCapabilities)))
		mux.Handle("POST /api/v1/admin/templates", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreateTemplate)))
		mux.Handle("PUT /api/v1/admin/templates/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateOfficialTemplate)))
		mux.Handle("POST /api/v1/admin/templates/compose-preview", admin(http.HandlerFunc(deps.AdminAcademicHandler.ComposeTemplatePreview)))
		mux.Handle("GET /api/v1/admin/template-components", staff(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplateComponents)))
		mux.Handle("POST /api/v1/admin/template-components", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreateTemplateComponent)))
		mux.Handle("PUT /api/v1/admin/template-components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("PATCH /api/v1/admin/template-components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("PUT /api/v1/admin/tech-components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("PATCH /api/v1/admin/tech-components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("PUT /api/v1/admin/components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("PATCH /api/v1/admin/components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateComponent)))
		mux.Handle("DELETE /api/v1/admin/template-components/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DeleteTemplateComponent)))
		mux.Handle("POST /api/v1/admin/templates/drafts", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.SaveDraft)))
		mux.Handle("GET /api/v1/admin/templates/drafts", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetDraftByUser)))
		mux.Handle("DELETE /api/v1/admin/templates/drafts", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DeleteDraft)))
		mux.Handle("PUT /api/v1/admin/templates/{id}/review", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ReviewTemplate)))
		mux.Handle("GET /api/v1/admin/templates/local-images", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListLocalImages)))
		mux.Handle("POST /api/v1/admin/templates/verify-image", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.VerifyImage)))
		mux.Handle("GET /api/v1/admin/templates/verify-image", admin(http.HandlerFunc(deps.AdminAcademicHandler.VerifyImage)))
		mux.Handle("POST /api/v1/admin/templates/{id}/duplicate", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DuplicateTemplate)))
		mux.Handle("POST /api/v1/plantillas/{id}/duplicar", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DuplicateTemplate)))
		mux.Handle("POST /api/v1/admin/templates/{id}/promote-to-model", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.PromoteTemplateToModel)))
		mux.Handle("GET /api/v1/admin/templates/cache-status", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetTemplatesCacheStatus)))
		mux.Handle("POST /api/v1/admin/templates/{id}/warm", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.WarmTemplate)))
		mux.Handle("POST /api/v1/admin/templates/warm-all", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.WarmAllTemplates)))
		mux.Handle("GET /api/v1/admin/template-categories", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplateCategories)))
		mux.Handle("GET /api/v1/admin/manual", admin(http.HandlerFunc(deps.AdminAcademicHandler.GetAdminManual)))
		mux.Handle("POST /api/v1/admin/template-categories", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.CreateTemplateCategory)))
		mux.Handle("PUT /api/v1/admin/template-categories/reorder", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ReorderTemplateCategories)))
		mux.Handle("PUT /api/v1/admin/template-categories/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateCategory)))
		mux.Handle("DELETE /api/v1/admin/template-categories/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DeleteTemplateCategory)))
		mux.Handle("GET /api/v1/admin/template-models", admin(http.HandlerFunc(deps.AdminAcademicHandler.ListTemplateModels)))
		mux.Handle("PUT /api/v1/admin/template-models/{id}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.UpdateTemplateModel)))
		mux.Handle("POST /api/v1/admin/template-models/{id}/deactivate", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.DeactivateTemplateModel)))
		mux.Handle("POST /api/v1/admin/template-models/{id}/reactivate", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ReactivateTemplateModel)))
		mux.Handle("POST /api/v1/admin/emergency/{action}", adminAudited(http.HandlerFunc(deps.AdminAcademicHandler.ExecuteEmergencyAction)))
	}

	if deps.EnvTestHandler != nil {
		mux.Handle("POST /api/v1/jobs/env-test", staff(http.HandlerFunc(deps.EnvTestHandler.StartEnvTest)))
		mux.Handle("GET /api/v1/jobs/env-test/{id}", tm(http.HandlerFunc(deps.EnvTestHandler.GetEnvTest)))
		mux.Handle("POST /api/v1/jobs/env-test/{id}/cancel", am(tm(http.HandlerFunc(deps.EnvTestHandler.CancelEnvTest))))
		mux.Handle("POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/env-test", am(tm(http.HandlerFunc(deps.EnvTestHandler.StartPracticeEnvTest))))
		mux.Handle("GET /api/v1/teacher/exercises/{exerciseId}/env-test/latest", tm(http.HandlerFunc(deps.EnvTestHandler.GetLatestPracticeEnvTest)))
	}

	// IDE de una práctica: el docente abre el repositorio de esa práctica en una
	// pestaña nueva, con su plantilla y su volumen, sin pasar por el entorno de
	// la materia.
	if deps.PracticeEnvironmentHandler != nil {
		mux.Handle("POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide", am(tm(http.HandlerFunc(deps.PracticeEnvironmentHandler.OpenPracticeIDE))))
	}
}

func registerStudentRoutes(mux *http.ServeMux, deps *Handlers) {
	tm := deps.TenantMiddleware
	if tm == nil {
		tm = func(next http.Handler) http.Handler { return WithAuth(next) }
	}

	// Aula de la clase: el alumno consulta si hay aula abierta y emite su
	// presencia. Sin señal, el tablero del docente no puede decir quién falta.
	// Se registra aparte del panel del alumno para que el aula funcione aunque
	// ese handler no esté montado.
	if deps.ClassSessionHandler != nil {
		mux.Handle("GET /api/v1/student/sessions/active", tm(http.HandlerFunc(deps.ClassSessionHandler.GetActiveAulas)))
		mux.Handle("POST /api/v1/student/sessions/{id}/heartbeat", tm(http.HandlerFunc(deps.ClassSessionHandler.PostHeartbeat)))
	}

	if deps.StudentHandler == nil {
		return
	}

	mux.Handle("GET /api/v1/student/dashboard", tm(http.HandlerFunc(deps.StudentHandler.GetDashboard)))
	mux.Handle("GET /api/v1/student/assignments/due", tm(http.HandlerFunc(deps.StudentHandler.GetDueAssignments)))
	mux.Handle("GET /api/v1/student/evaluations", tm(http.HandlerFunc(deps.StudentHandler.GetEvaluations)))
	mux.Handle("GET /api/v1/student/submissions", tm(http.HandlerFunc(deps.StudentHandler.GetSubmissions)))
	mux.Handle("GET /api/v1/student/submissions/{id}/comments", tm(http.HandlerFunc(deps.StudentHandler.GetSubmissionComments)))
	mux.Handle("GET /api/v1/student/courses/{id}/recommendations", tm(http.HandlerFunc(deps.StudentHandler.GetRecommendations)))
}

// registerClassSessionRoutes expone el aula del docente: apertura, cierre,
// tablero y la sesión abierta actual.
func registerClassSessionRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.ClassSessionHandler == nil {
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

	mux.Handle("GET /api/v1/teacher/sessions/open", tm(http.HandlerFunc(deps.ClassSessionHandler.GetCurrentAula)))
	mux.Handle("GET /api/v1/teacher/sessions/last-closed", tm(http.HandlerFunc(deps.ClassSessionHandler.GetLastClosedAula)))
	mux.Handle("GET /api/v1/teacher/sessions/{id}/board", tm(http.HandlerFunc(deps.ClassSessionHandler.GetBoard)))
	mux.Handle("POST /api/v1/teacher/sessions", am(tm(http.HandlerFunc(deps.ClassSessionHandler.OpenAula))))
	mux.Handle("PATCH /api/v1/teacher/sessions/{id}/close", am(tm(http.HandlerFunc(deps.ClassSessionHandler.CloseAula))))
	// Congelamiento del examen (ADR-005 §5): al cerrar, los volúmenes pasan a
	// solo lectura. La ruta explícita repite la operación y la de alumnos es la
	// excepción individual con motivo.
	mux.Handle("POST /api/v1/teacher/sessions/{id}/exam-freeze", am(tm(http.HandlerFunc(deps.ClassSessionHandler.FreezeExam))))
	mux.Handle("POST /api/v1/teacher/sessions/{id}/students/{studentID}/unfreeze", am(tm(http.HandlerFunc(deps.ClassSessionHandler.UnfreezeExamStudent))))
}

// registerClassAdvanceRoutes expone el avance voluntario del docente (ADR-058):
// emitir, congelar la clase y extender la ventana de rezago del lado docente, y
// ver, aplicar con respaldo o descartar el aviso del lado del alumno.
func registerClassAdvanceRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.ClassAdvanceHandler == nil {
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

	mux.Handle("POST /api/v1/teacher/sessions/{id}/advances", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.PostAdvance))))
	mux.Handle("GET /api/v1/teacher/sessions/{id}/advances", tm(http.HandlerFunc(deps.ClassAdvanceHandler.GetSessionAdvances)))
	mux.Handle("GET /api/v1/teacher/sessions/{id}/history", tm(http.HandlerFunc(deps.ClassAdvanceHandler.GetSessionHistory)))
	mux.Handle("PATCH /api/v1/teacher/sessions/{id}/freeze", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.FreezeSession))))
	mux.Handle("PATCH /api/v1/teacher/sessions/{id}/unfreeze", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.UnfreezeSession))))
	mux.Handle("PATCH /api/v1/teacher/sessions/{id}/grace", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.ExtendGrace))))
	mux.Handle("GET /api/v1/class-advances/pending", tm(http.HandlerFunc(deps.ClassAdvanceHandler.GetPendingAdvances)))
	mux.Handle("GET /api/v1/class-advances/{id}/files", tm(http.HandlerFunc(deps.ClassAdvanceHandler.PreviewAdvance)))
	mux.Handle("POST /api/v1/class-advances/{id}/apply", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.ApplyAdvance))))
	mux.Handle("POST /api/v1/class-advances/{id}/rollback", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.RollbackAdvance))))
	mux.Handle("POST /api/v1/class-advances/{id}/dismiss", am(tm(http.HandlerFunc(deps.ClassAdvanceHandler.DismissAdvance))))
}

func registerUserRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.UserHandler == nil {
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

	mux.Handle("POST /api/v1/users", am(tm(http.HandlerFunc(deps.UserHandler.Create))))
	mux.Handle("GET /api/v1/users/me", tm(http.HandlerFunc(deps.UserHandler.GetMe)))
}

func registerTemplateRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.TemplateHandler == nil {
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

	mux.Handle("POST /api/v1/templates", am(tm(http.HandlerFunc(deps.TemplateHandler.Create))))
	mux.Handle("GET /api/v1/templates", tm(http.HandlerFunc(deps.TemplateHandler.GetAll)))
}

// registerEnvironmentRequestRoutes expone el pedido de entorno del docente.
// Vive fuera de registerTemplateRoutes a propósito: la solicitud es una
// intención, no una plantilla, y se autentica y aísla por tenant.
func registerEnvironmentRequestRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.EnvironmentRequestHandler == nil {
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

	h := deps.EnvironmentRequestHandler

	mux.Handle("POST /api/v1/teacher/environment-requests/preview", tm(http.HandlerFunc(h.Preview)))
	mux.Handle("POST /api/v1/teacher/environment-requests", am(tm(http.HandlerFunc(h.Create))))
	mux.Handle("GET /api/v1/teacher/environment-requests", tm(http.HandlerFunc(h.ListMine)))
	mux.Handle("GET /api/v1/teacher/environment-requests/{id}", tm(http.HandlerFunc(h.GetByID)))

	mux.Handle("GET /api/v1/admin/environment-requests", tm(adminOnly(http.HandlerFunc(h.ListForReview))))
	mux.Handle("PATCH /api/v1/admin/environment-requests/{id}", tm(am(adminOnly(http.HandlerFunc(h.Review)))))
}

func registerAuthRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.AuthHandler != nil {
		loginRL := deps.LoginRateLimitMiddleware
		if loginRL == nil {
			loginRL = func(next http.Handler) http.Handler { return next }
		}
		mux.Handle("POST /api/v1/auth/login", loginRL(http.HandlerFunc(deps.AuthHandler.HandleGoogleLogin)))
		mux.Handle("GET /auth/google/login", loginRL(http.HandlerFunc(deps.AuthHandler.HandleGoogleLogin)))
		mux.HandleFunc("GET /api/v1/auth/callback", deps.AuthHandler.HandleGoogleCallback)
		mux.HandleFunc("GET /auth/google/callback", deps.AuthHandler.HandleGoogleCallback)
		mux.HandleFunc("GET /api/v1/auth/verify", deps.AuthHandler.VerifyAuth)
		mux.HandleFunc("POST /api/v1/auth/logout", deps.AuthHandler.HandleLogout)
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
	mux.Handle("POST /api/v1/exercises/calculate-outputs", am(tm(http.HandlerFunc(h.CalculateOutputs))))
	mux.Handle("POST /api/v1/exercises/validate-input", am(tm(http.HandlerFunc(h.ValidateInputFormat))))
	mux.Handle("POST /api/v1/exercises/generate-cases", am(tm(http.HandlerFunc(h.GenerateCases))))
	mux.Handle("POST /api/v1/exercises/{id}/checklist", am(tm(http.HandlerFunc(h.GetExerciseChecklist))))
	mux.Handle("POST /api/v1/exercises/{id}/dry-run", am(tm(http.HandlerFunc(h.StartDryRun))))
	mux.Handle("GET /api/v1/exercises/{id}/dry-run/jobs/{jobId}", tm(http.HandlerFunc(h.GetDryRunJob)))
	mux.Handle("GET /api/v1/dry-run/jobs/{jobId}", tm(http.HandlerFunc(h.GetDryRunJob)))
	mux.Handle("POST /api/v1/teacher/courses/{courseId}/exercises/import", am(tm(http.HandlerFunc(h.ImportExercises))))
	mux.Handle("POST /api/v1/teacher/exercises/{exerciseId}/generate-from-script", am(tm(http.HandlerFunc(h.GenerateCasesFromScript))))
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
	mux.Handle("PUT /api/v1/admin/judge/profiles/{language}", tm(am(adminOnly(http.HandlerFunc(deps.LanguageProfileHandler.UpdateProfile)))))
}

func registerPracticeEnvironmentRoutes(mux *http.ServeMux, h *PracticeEnvironmentHandler, tenantMiddleware func(http.Handler) http.Handler, rateLimitMiddleware func(http.Handler) http.Handler) {
	if h == nil {
		return
	}
	if rateLimitMiddleware == nil {
		rateLimitMiddleware = func(next http.Handler) http.Handler { return next }
	}
	if tenantMiddleware != nil {
		mux.Handle("POST /api/v1/environments/start", tenantMiddleware(rateLimitMiddleware(http.HandlerFunc(h.StartPracticeEnvironment))))
		mux.Handle("POST /api/v1/environments/{id}/pause", tenantMiddleware(http.HandlerFunc(h.PausePracticeEnvironment)))
		mux.Handle("DELETE /api/v1/environments/{id}", tenantMiddleware(http.HandlerFunc(h.TerminatePracticeEnvironment)))
		mux.Handle("GET /api/v1/environments/{id}/audit", tenantMiddleware(http.HandlerFunc(h.GetSemgrepAudit)))
		mux.Handle("POST /api/v1/environments/{id}/heartbeat", tenantMiddleware(http.HandlerFunc(h.Heartbeat)))
		mux.Handle("POST /api/v1/environments/{id}/restart", tenantMiddleware(http.HandlerFunc(h.RestartPracticeEnvironment)))
		mux.Handle("POST /api/v1/environments/{id}/submit", tenantMiddleware(WithAuth(http.HandlerFunc(h.SubmitPracticeEnvironmentProject))))
	} else {
		mux.Handle("POST /api/v1/environments/start", WithAuth(rateLimitMiddleware(http.HandlerFunc(h.StartPracticeEnvironment))))
		mux.Handle("POST /api/v1/environments/{id}/pause", WithAuth(http.HandlerFunc(h.PausePracticeEnvironment)))
		mux.Handle("DELETE /api/v1/environments/{id}", WithAuth(http.HandlerFunc(h.TerminatePracticeEnvironment)))
		mux.Handle("GET /api/v1/environments/{id}/audit", WithAuth(http.HandlerFunc(h.GetSemgrepAudit)))
		mux.Handle("POST /api/v1/environments/{id}/heartbeat", WithAuth(http.HandlerFunc(h.Heartbeat)))
		mux.Handle("POST /api/v1/environments/{id}/restart", WithAuth(http.HandlerFunc(h.RestartPracticeEnvironment)))
		mux.Handle("POST /api/v1/environments/{id}/submit", WithAuth(http.HandlerFunc(h.SubmitPracticeEnvironmentProject)))
	}
}

func registerMetricsRoutes(mux *http.ServeMux, h *MetricsHandler) {
	mux.Handle("GET /metrics", metricsAccessMiddleware(promhttp.Handler()))
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
	mux.Handle("GET /api/v1/teacher/overview", tm(http.HandlerFunc(deps.TeacherHandler.GetOverview)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/labs", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseLabs)))
	mux.Handle("GET /api/v1/teacher/exercises/bank", tm(http.HandlerFunc(deps.TeacherHandler.GetExerciseBank)))
	mux.Handle("POST /api/v1/teacher/exercises/{id}/duplicate", am(tm(http.HandlerFunc(deps.TeacherHandler.DuplicateExercise))))
	mux.Handle("GET /api/v1/teacher/reviews", tm(http.HandlerFunc(deps.TeacherHandler.GetReviewQueue)))
	mux.Handle("PATCH /api/v1/teacher/submissions/{id}/review", am(tm(http.HandlerFunc(deps.TeacherHandler.UpdateSubmissionReview))))
	mux.Handle("PUT /api/v1/teacher/grades/{studentId}/exercises/{exerciseId}", am(tm(http.HandlerFunc(deps.TeacherHandler.SetExerciseGrade))))
	mux.Handle("GET /api/v1/teacher/courses/{id}/submissions", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseSubmissions)))
	mux.Handle("GET /api/v1/teacher/submissions/{id}/review", tm(http.HandlerFunc(deps.TeacherHandler.GetSubmissionReview)))
	mux.Handle("POST /api/v1/teacher/submissions/{id}/comments", am(tm(http.HandlerFunc(deps.TeacherHandler.AddSubmissionComment))))
	mux.Handle("GET /api/v1/teacher/submissions/{id}/comments", tm(http.HandlerFunc(deps.TeacherHandler.GetSubmissionComments)))
	mux.Handle("POST /api/v1/teacher/submissions/{id}/run-ephemeral", tm(http.HandlerFunc(deps.TeacherHandler.RunEphemeral)))
	mux.Handle("POST /api/v1/teacher/submissions/{id}/test-environment", tm(http.HandlerFunc(deps.TeacherHandler.StartReviewEnvironment)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/grades/matrix", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseGradesMatrix)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/grades/export", tm(http.HandlerFunc(deps.TeacherHandler.ExportGrades)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/analytics", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseAnalytics)))
	mux.Handle("GET /api/v1/courses/{id}/analytics", tm(http.HandlerFunc(deps.TeacherHandler.GetCourseAnalytics)))

	mux.Handle("GET /api/v1/teacher/submissions/{id}/timeline", tm(http.HandlerFunc(deps.TeacherHandler.GetTimeline)))
	mux.Handle("GET /api/v1/teacher/submissions/{id}/keystroke-events", tm(http.HandlerFunc(deps.TeacherHandler.GetKeystrokeEvents)))
	mux.Handle("GET /api/v1/teacher/plagiarism", tm(http.HandlerFunc(deps.TeacherHandler.AnalyzePlagiarism)))
	mux.Handle("GET /api/v1/teacher/courses/{id}/plagiarism", tm(http.HandlerFunc(deps.TeacherHandler.AnalyzePlagiarism)))
	mux.Handle("GET /api/v1/teacher/plagiarism/pairs/{submission_a}/{submission_b}", tm(http.HandlerFunc(deps.TeacherHandler.GetPlagiarismPair)))
	mux.Handle("GET /api/v1/teacher/live-sessions", tm(http.HandlerFunc(deps.TeacherHandler.GetLiveSessions)))
	mux.Handle("GET /api/v1/teacher/live-sessions/metrics", tm(http.HandlerFunc(deps.TeacherHandler.GetLiveSessionsMetrics)))
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
	am := deps.AuditMiddleware
	if am == nil {
		am = func(next http.Handler) http.Handler { return next }
	}

	mux.Handle("GET /api/v1/admin/backups/config", tm(adminOnly(http.HandlerFunc(deps.BackupHandler.GetConfig))))
	mux.Handle("PUT /api/v1/admin/backups/config", tm(am(adminOnly(http.HandlerFunc(deps.BackupHandler.UpdateConfig)))))
	mux.Handle("GET /api/v1/admin/backups", tm(adminOnly(http.HandlerFunc(deps.BackupHandler.List))))
	mux.Handle("POST /api/v1/admin/backups/trigger", tm(am(adminOnly(http.HandlerFunc(deps.BackupHandler.Trigger)))))
	mux.Handle("POST /api/v1/admin/backups/{id}/verify", tm(am(adminOnly(http.HandlerFunc(deps.BackupHandler.Verify)))))
	mux.Handle("GET /api/v1/admin/backups/{id}/download", tm(adminOnly(http.HandlerFunc(deps.BackupHandler.Download))))
}

func registerCourseModuleRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.CourseModuleHandler == nil {
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

	// Teacher module routes
	mux.Handle("POST /api/v1/teacher/courses/{id}/modules", am(tm(http.HandlerFunc(deps.CourseModuleHandler.CreateModule))))
	mux.Handle("GET /api/v1/teacher/courses/{id}/modules", tm(http.HandlerFunc(deps.CourseModuleHandler.ListModules)))
	mux.Handle("PUT /api/v1/teacher/modules/{id}", am(tm(http.HandlerFunc(deps.CourseModuleHandler.UpdateModule))))
	mux.Handle("DELETE /api/v1/teacher/modules/{id}", am(tm(http.HandlerFunc(deps.CourseModuleHandler.DeleteModule))))
	mux.Handle("POST /api/v1/teacher/modules/{id}/prerequisites", am(tm(http.HandlerFunc(deps.CourseModuleHandler.SetPrerequisites))))
	mux.Handle("DELETE /api/v1/teacher/modules/{id}/prerequisites", am(tm(http.HandlerFunc(deps.CourseModuleHandler.DeletePrerequisites))))
	mux.Handle("PUT /api/v1/teacher/exercises/{id}/module", am(tm(http.HandlerFunc(deps.CourseModuleHandler.AssignExerciseModule))))

	// Student curricular map route
	mux.Handle("GET /api/v1/student/courses/{id}/map", tm(http.HandlerFunc(deps.CourseModuleHandler.GetStudentCurricularMap)))
}

func registerRubricRoutes(mux *http.ServeMux, deps *Handlers) {
	if deps.RubricHandler == nil {
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

	mux.Handle("GET /api/v1/exercises/{id}/rubric", tm(http.HandlerFunc(deps.RubricHandler.GetRubric)))
	mux.Handle("PUT /api/v1/exercises/{id}/rubric", am(tm(http.HandlerFunc(deps.RubricHandler.SaveRubric))))
	mux.Handle("POST /api/v1/submissions/{id}/rubric-evaluation", am(tm(http.HandlerFunc(deps.RubricHandler.EvaluateSubmission))))
}
