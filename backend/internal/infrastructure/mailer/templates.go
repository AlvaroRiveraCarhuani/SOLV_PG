package mailer

import (
	"fmt"
	"html"
	"strings"

	"solv-backend/internal/core/domain"
)

// renderEmailLayout genera la estructura HTML responsiva compartida para todas las comunicaciones
// del sistema, aplicando marca blanca (White-Label) institucional y aislando los estilos de correo.
func renderEmailLayout(branding domain.EmailBranding, title, preheader, bodyContentHTML string) string {
	instName := strings.TrimSpace(branding.InstitutionName)
	if instName == "" {
		instName = "Plataforma Académica"
	}

	escapedInstName := html.EscapeString(instName)
	escapedTitle := html.EscapeString(title)
	escapedPreheader := html.EscapeString(preheader)
	absLogoURL := branding.AbsoluteLogoURL()

	var logoBlock string
	if absLogoURL != "" {
		escapedLogo := html.EscapeString(absLogoURL)
		logoBlock = fmt.Sprintf(`
      <div style="margin-bottom: 16px;">
        <img src="%s" alt="%s" height="42" style="max-height: 48px; max-width: 220px; display: inline-block; border: 0; outline: none; text-decoration: none;" />
      </div>`, escapedLogo, escapedInstName)
	}

	var supportContact string
	if strings.TrimSpace(branding.SupportEmail) != "" {
		escapedEmail := html.EscapeString(strings.TrimSpace(branding.SupportEmail))
		supportContact = fmt.Sprintf(`<br>Para asistencia técnica o consultas, puede escribir a <a href="mailto:%s" style="color: #2563eb; text-decoration: underline;">%s</a>.`, escapedEmail, escapedEmail)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>
    body { margin: 0; padding: 0; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1e293b; line-height: 1.6; }
    .wrapper { width: 100%%; max-width: 580px; margin: 32px auto; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05); }
    .header { background-color: #0f172a; padding: 28px 32px; text-align: left; }
    .header h1 { margin: 0; font-size: 18px; color: #ffffff; font-weight: 600; letter-spacing: -0.01em; }
    .header p { margin: 4px 0 0 0; font-size: 13px; color: #94a3b8; }
    .content { padding: 32px; font-size: 15px; color: #334155; }
    .greeting { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: #0f172a; }
    .panel { background-color: #f8fafc; border: 1px solid #e2e8f0; border-left: 4px solid #2563eb; padding: 16px 20px; margin: 20px 0; border-radius: 0 6px 6px 0; font-size: 14px; }
    .btn-container { text-align: center; margin: 28px 0 20px 0; }
    .btn { display: inline-block; background-color: #2563eb; color: #ffffff !important; text-decoration: none; padding: 12px 28px; border-radius: 6px; font-weight: 600; font-size: 14px; letter-spacing: 0.01em; }
    .footer { padding: 24px 32px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 12px; color: #64748b; text-align: center; line-height: 1.5; }
  </style>
</head>
<body>
  <!-- Preheader oculto para vista previa en bandeja de entrada -->
  <div style="display: none; font-size: 1px; color: #f8fafc; line-height: 1px; max-height: 0px; max-width: 0px; opacity: 0; overflow: hidden;">
    %s
  </div>
  <div class="wrapper">
    <div class="header">
      %s
      <h1>%s</h1>
      <p>Servicios Académicos Digitales</p>
    </div>
    <div class="content">
      %s
    </div>
    <div class="footer">
      Mensaje institucional automático emitido por los servicios académicos de <strong>%s</strong>.%s
      <br>Por favor, no responda directamente a este correo electrónico.
    </div>
  </div>
</body>
</html>`, escapedTitle, escapedPreheader, logoBlock, escapedInstName, bodyContentHTML, escapedInstName, supportContact)
}

// BuildTeacherWelcomeEmail genera el correo de bienvenida y habilitación de acceso para un docente.
func BuildTeacherWelcomeEmail(branding domain.EmailBranding, teacherName, loginURL string) (htmlBody, textBody string) {
	instName := strings.TrimSpace(branding.InstitutionName)
	if instName == "" {
		instName = "la institución"
	}
	if loginURL == "" {
		loginURL = "http://localhost:4200/login"
	}

	displayName := strings.TrimSpace(teacherName)
	greetingName := "docente"
	if displayName != "" {
		greetingName = displayName
	}

	escapedName := html.EscapeString(greetingName)
	escapedInst := html.EscapeString(instName)
	escapedURL := html.EscapeString(loginURL)

	textBody = fmt.Sprintf(`Estimado/a %s:

Le informamos que se ha habilitado su cuenta con perfil de Docente en la plataforma académica de %s.

Instrucciones para ingresar:
1. Acceda a la plataforma: %s
2. Seleccione la opción "Iniciar sesión con Google" utilizando su cuenta de correo institucional.
3. El sistema reconocerá automáticamente sus permisos y asignaturas asignadas.

Si usted no reconoce esta habilitación o no forma parte del cuerpo docente, por favor comuníquese con la administración académica.

Atentamente,
Servicios Académicos — %s
`, greetingName, instName, loginURL, instName)

	contentHTML := fmt.Sprintf(`
      <div class="greeting">Estimado/a %s:</div>
      <p>Le informamos que se ha habilitado su cuenta con perfil de <strong>Docente</strong> en la plataforma académica de <strong>%s</strong>.</p>
      
      <div class="panel">
        <p style="margin: 0 0 8px 0; font-weight: 600; color: #0f172a;">Instrucciones para ingresar:</p>
        <ol style="margin: 0; padding-left: 20px; color: #334155;">
          <li style="margin-bottom: 6px;">Acceda a la plataforma mediante el enlace a continuación.</li>
          <li style="margin-bottom: 6px;">Seleccione la opción <strong>Iniciar sesión con Google</strong> utilizando su cuenta institucional.</li>
          <li>El sistema reconocerá automáticamente sus permisos y asignaturas.</li>
        </ol>
      </div>

      <div class="btn-container">
        <a href="%s" class="btn" target="_blank" rel="noopener noreferrer">Acceder a la plataforma académica</a>
      </div>

      <p style="font-size: 13px; color: #64748b; margin-top: 24px;">Si el botón anterior no responde, copie y pegue el siguiente enlace en la barra de direcciones de su navegador:<br><a href="%s" style="color: #2563eb; word-break: break-all;">%s</a></p>
`, escapedName, escapedInst, escapedURL, escapedURL, escapedURL)

	htmlBody = renderEmailLayout(
		branding,
		fmt.Sprintf("Habilitación de acceso docente — %s", instName),
		"Se ha habilitado su cuenta docente en la plataforma académica.",
		contentHTML,
	)

	return htmlBody, textBody
}

// BuildGradeFeedbackEmail genera el aviso al estudiante cuando su entrega ha sido evaluada por el docente.
func BuildGradeFeedbackEmail(branding domain.EmailBranding, studentName, assignmentTitle, subjectName, reviewURL string) (htmlBody, textBody string) {
	instName := strings.TrimSpace(branding.InstitutionName)
	if instName == "" {
		instName = "la institución"
	}

	displayName := strings.TrimSpace(studentName)
	greetingName := "estudiante"
	if displayName != "" {
		greetingName = displayName
	}

	escapedName := html.EscapeString(greetingName)
	escapedAssignment := html.EscapeString(strings.TrimSpace(assignmentTitle))
	escapedSubject := html.EscapeString(strings.TrimSpace(subjectName))
	escapedURL := html.EscapeString(strings.TrimSpace(reviewURL))

	textBody = fmt.Sprintf(`Estimado/a %s:

Le informamos que el docente ha publicado la revisión y retroalimentación correspondiente al laboratorio "%s" de la materia "%s".

Puede consultar las observaciones detalladas sobre su solución, los resultados de ejecución y las indicaciones docentes ingresando a su cuenta académica en el siguiente enlace:
%s

Atentamente,
Servicios Académicos — %s
`, greetingName, assignmentTitle, subjectName, reviewURL, instName)

	contentHTML := fmt.Sprintf(`
      <div class="greeting">Estimado/a %s:</div>
      <p>Le informamos que el docente ha publicado la revisión y retroalimentación correspondiente al laboratorio <strong>%s</strong> de la materia <strong>%s</strong>.</p>
      
      <div class="panel">
        <p style="margin: 0; color: #334155;">Las observaciones detalladas, resultados de ejecución y comentarios pedagógicos sobre su entrega ya se encuentran disponibles para su consulta formativa en la plataforma.</p>
      </div>

      <div class="btn-container">
        <a href="%s" class="btn" target="_blank" rel="noopener noreferrer">Consultar revisión en la plataforma</a>
      </div>

      <p style="font-size: 13px; color: #64748b; margin-top: 24px;">Si el botón no responde, copie y pegue el siguiente enlace en su navegador:<br><a href="%s" style="color: #2563eb; word-break: break-all;">%s</a></p>
`, escapedName, escapedAssignment, escapedSubject, escapedURL, escapedURL, escapedURL)

	htmlBody = renderEmailLayout(
		branding,
		fmt.Sprintf("Evaluación publicada: %s — %s", assignmentTitle, subjectName),
		fmt.Sprintf("Se ha publicado la revisión del laboratorio %s.", assignmentTitle),
		contentHTML,
	)

	return htmlBody, textBody
}

// BuildDeadlineReminderEmail genera el aviso de urgencia a 12 horas del límite de entrega.
func BuildDeadlineReminderEmail(
	branding domain.EmailBranding,
	studentName,
	assignmentTitle,
	subjectName,
	formattedDueDate string,
	hasStartedWorkspace bool,
	workspaceURL string,
) (htmlBody, textBody string) {
	instName := strings.TrimSpace(branding.InstitutionName)
	if instName == "" {
		instName = "la institución"
	}

	displayName := strings.TrimSpace(studentName)
	greetingName := "estudiante"
	if displayName != "" {
		greetingName = displayName
	}

	escapedName := html.EscapeString(greetingName)
	escapedAssignment := html.EscapeString(strings.TrimSpace(assignmentTitle))
	escapedSubject := html.EscapeString(strings.TrimSpace(subjectName))
	escapedDueDate := html.EscapeString(strings.TrimSpace(formattedDueDate))
	escapedURL := html.EscapeString(strings.TrimSpace(workspaceURL))

	var stateTextHTML, stateTextPlain string
	if hasStartedWorkspace {
		stateTextPlain = "Registramos avances guardados en su entorno de trabajo, pero la entrega final aún no ha sido confirmada en el sistema. Le recordamos verificar sus archivos y remitir la entrega antes del plazo límite."
		stateTextHTML = `<p style="margin: 0; color: #334155;"><strong>Estado actual:</strong> Registramos avances guardados en su entorno de trabajo, pero la entrega definitiva aún no ha sido confirmada en el sistema. Le recordamos verificar sus archivos y confirmar el envío a tiempo.</p>`
	} else {
		stateTextPlain = "Actualmente no registra actividad ni entrega enviada para esta asignación. Le recomendamos acceder con anticipación para resolver y remitir la tarea."
		stateTextHTML = `<p style="margin: 0; color: #334155;"><strong>Estado actual:</strong> Actualmente no registra actividad ni entrega enviada para esta asignación. Le recomendamos acceder con suficiente anticipación para resolver y remitir la tarea.</p>`
	}

	textBody = fmt.Sprintf(`Estimado/a %s:

Le recordamos que el plazo para la entrega del laboratorio "%s" de la materia "%s" vencerá en 12 horas.

Fecha límite: %s

%s

Puede ingresar directamente a su espacio de trabajo en el siguiente enlace:
%s

Si ya realizó su entrega recientemente, puede desestimar este recordatorio.

Atentamente,
Servicios Académicos — %s
`, greetingName, assignmentTitle, subjectName, formattedDueDate, stateTextPlain, workspaceURL, instName)

	contentHTML := fmt.Sprintf(`
      <div class="greeting">Estimado/a %s:</div>
      <p>Le recordamos que el plazo para la entrega del laboratorio <strong>%s</strong> de la materia <strong>%s</strong> vencerá en aproximadamente <strong>12 horas</strong>.</p>
      
      <div class="panel" style="border-left-color: #eab308;">
        <p style="margin: 0 0 8px 0; color: #0f172a;"><strong>Fecha límite de cierre:</strong> %s</p>
        %s
      </div>

      <div class="btn-container">
        <a href="%s" class="btn" target="_blank" rel="noopener noreferrer">Acceder al entorno de trabajo</a>
      </div>

      <p style="font-size: 13px; color: #64748b; margin-top: 24px;">Si el botón no responde, copie y pegue el siguiente enlace en su navegador:<br><a href="%s" style="color: #2563eb; word-break: break-all;">%s</a></p>
      <p style="font-size: 12px; color: #94a3b8; margin-top: 12px;">Si usted ya confirmó y remitió su entrega en los últimos minutos, puede desestimar este mensaje.</p>
`, escapedName, escapedAssignment, escapedSubject, escapedDueDate, stateTextHTML, escapedURL, escapedURL, escapedURL)

	htmlBody = renderEmailLayout(
		branding,
		fmt.Sprintf("Recordatorio: Próximo cierre de %s — %s", assignmentTitle, subjectName),
		fmt.Sprintf("Faltan 12 horas para el cierre del laboratorio %s.", assignmentTitle),
		contentHTML,
	)

	return htmlBody, textBody
}
