package mailer

import (
	"fmt"
	"html"
	"strings"
)

// BuildTeacherWelcomeEmail genera el cuerpo HTML y texto plano para la bienvenida de un docente
func BuildTeacherWelcomeEmail(institutionName, teacherName, loginURL string) (htmlBody, textBody string) {
	if institutionName == "" {
		institutionName = "SOLV — Laboratorios Virtuales"
	}
	if loginURL == "" {
		loginURL = "http://localhost:4200/login"
	}
	displayName := strings.TrimSpace(teacherName)
	if displayName == "" {
		displayName = "Docente"
	}

	escapedInst := html.EscapeString(institutionName)
	escapedName := html.EscapeString(displayName)
	escapedURL := html.EscapeString(loginURL)

	textBody = fmt.Sprintf(`Hola %s,

Se ha habilitado tu acceso docente en la plataforma de laboratorios virtuales de %s.

Para ingresar:
1. Dirígete a: %s
2. Selecciona "Iniciar sesión con Google" utilizando tu cuenta de correo institucional.
3. Tu rol docente se activará de forma automática.

Si no reconoces esta acción, por favor contacta al administrador del sistema.

Saludos cordiales,
Equipo de Soporte SOLV
`, displayName, institutionName, loginURL)

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Bienvenido/a a SOLV</title>
  <style>
    body { margin: 0; padding: 0; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1e293b; }
    .wrapper { width: 100%%; max-width: 580px; margin: 32px auto; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; }
    .header { background-color: #1e293b; padding: 24px 32px; text-align: left; }
    .header h1 { margin: 0; font-size: 18px; color: #ffffff; font-weight: 600; }
    .header p { margin: 4px 0 0 0; font-size: 13px; color: #94a3b8; }
    .content { padding: 32px; line-height: 1.6; font-size: 15px; }
    .greeting { font-size: 17px; font-weight: 600; margin-bottom: 16px; color: #0f172a; }
    .steps { background-color: #f1f5f9; border-left: 3px solid #3b82f6; padding: 16px; margin: 20px 0; border-radius: 0 4px 4px 0; }
    .steps ol { margin: 0; padding-left: 20px; }
    .steps li { margin-bottom: 6px; font-size: 14px; }
    .btn-container { text-align: center; margin: 28px 0; }
    .btn { display: inline-block; background-color: #2563eb; color: #ffffff !important; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 600; font-size: 14px; }
    .footer { padding: 20px 32px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 12px; color: #64748b; text-align: center; }
  </style>
</head>
<body>
  <div class="wrapper">
    <div class="header">
      <h1>%s</h1>
      <p>Sistema de Orquestación de Laboratorios Virtuales</p>
    </div>
    <div class="content">
      <div class="greeting">Estimado/a %s,</div>
      <p>Te informamos que se ha habilitado tu cuenta con perfil de <strong>Docente</strong> en la plataforma académica.</p>
      
      <div class="steps">
        <p style="margin: 0 0 8px 0; font-weight: 600;">Instrucciones de ingreso:</p>
        <ol>
          <li>Ingresa a la plataforma pulsando el botón a continuación.</li>
          <li>Selecciona <strong>Iniciar sesión con Google</strong> usando tu cuenta institucional.</li>
          <li>El sistema reconocerá automáticamente tu rol docente.</li>
        </ol>
      </div>

      <div class="btn-container">
        <a href="%s" class="btn" target="_blank" rel="noopener noreferrer">Acceder a la Plataforma</a>
      </div>

      <p style="font-size: 13px; color: #64748b;">Si el botón no funciona, copia y pega el siguiente enlace en tu navegador:<br><a href="%s" style="color: #2563eb; word-break: break-all;">%s</a></p>
    </div>
    <div class="footer">
      Este es un mensaje institucional automático generado por SOLV para %s.
    </div>
  </div>
</body>
</html>`, escapedInst, escapedName, escapedURL, escapedURL, escapedURL, escapedInst)

	return htmlBody, textBody
}
