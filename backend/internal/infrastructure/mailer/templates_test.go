package mailer_test

import (
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/mailer"
)

func TestEmailBranding_AbsoluteLogoURL(t *testing.T) {
	tests := []struct {
		name     string
		branding domain.EmailBranding
		expected string
	}{
		{
			name: "relative path with leading slash",
			branding: domain.EmailBranding{
				BaseDomain: "solv.uab.edu.bo",
				LogoURL:    "/assets/uab-logo.png",
			},
			expected: "https://solv.uab.edu.bo/assets/uab-logo.png",
		},
		{
			name: "relative path without leading slash",
			branding: domain.EmailBranding{
				BaseDomain: "solv.uab.edu.bo",
				LogoURL:    "assets/uab-logo.png",
			},
			expected: "https://solv.uab.edu.bo/assets/uab-logo.png",
		},
		{
			name: "already absolute https URL",
			branding: domain.EmailBranding{
				BaseDomain: "solv.uab.edu.bo",
				LogoURL:    "https://cdn.uab.edu.bo/logo.svg",
			},
			expected: "https://cdn.uab.edu.bo/logo.svg",
		},
		{
			name: "empty logo URL",
			branding: domain.EmailBranding{
				BaseDomain: "solv.uab.edu.bo",
				LogoURL:    "",
			},
			expected: "",
		},
		{
			name: "empty base domain with relative logo",
			branding: domain.EmailBranding{
				BaseDomain: "",
				LogoURL:    "/assets/logo.png",
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.branding.AbsoluteLogoURL()
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestTeacherWelcomeEmail_NoSOLVBrandLeak(t *testing.T) {
	branding := domain.EmailBranding{
		InstitutionName: "Universidad Adventista de Bolivia",
		BaseDomain:      "solv.uab.edu.bo",
		LogoURL:         "/assets/uab-logo.png",
		SupportEmail:    "soporte@uab.edu.bo",
	}

	htmlBody, textBody := mailer.BuildTeacherWelcomeEmail(branding, "Carlos Perez", "https://solv.uab.edu.bo/login")

	// Verificar marca blanca institucional en badge
	if !strings.Contains(htmlBody, "Universidad Adventista de Bolivia") {
		t.Errorf("htmlBody should contain institution name")
	}
	if !strings.Contains(htmlBody, "class=\"badge\"") {
		t.Errorf("htmlBody should contain badge class")
	}
	if !strings.Contains(htmlBody, "soporte@uab.edu.bo") {
		t.Errorf("htmlBody should contain support email")
	}
	if !strings.Contains(textBody, "Carlos Perez") {
		t.Errorf("textBody should contain teacher name")
	}

	// Verificar ausencia de la marca interna SOLV
	if strings.Contains(htmlBody, "SOLV") {
		t.Errorf("htmlBody should not leak 'SOLV' internal brand, got: %s", htmlBody)
	}
	if strings.Contains(textBody, "SOLV") {
		t.Errorf("textBody should not leak 'SOLV' internal brand, got: %s", textBody)
	}

	// Verificar tono formal ("usted")
	if strings.Contains(htmlBody, "Te informamos") || strings.Contains(htmlBody, "tu cuenta") {
		t.Errorf("htmlBody should use formal tone ('usted'), found informal phrases")
	}
}

func TestGradeFeedbackEmail(t *testing.T) {
	branding := domain.EmailBranding{
		InstitutionName: "Universidad Adventista de Bolivia",
		BaseDomain:      "solv.uab.edu.bo",
		LogoURL:         "/assets/uab-logo.png",
	}

	htmlBody, textBody := mailer.BuildGradeFeedbackEmail(
		branding,
		"Ana Gomez",
		"Laboratorio 03: Arboles AVL",
		"Estructura de Datos",
		"https://solv.uab.edu.bo/student/courses/10/assignments/25",
	)

	// Verificar contenido clave
	if !strings.Contains(htmlBody, "Ana Gomez") {
		t.Errorf("htmlBody missing student name")
	}
	if !strings.Contains(htmlBody, "Laboratorio 03: Arboles AVL") {
		t.Errorf("htmlBody missing assignment title")
	}
	if !strings.Contains(htmlBody, "Estructura de Datos") {
		t.Errorf("htmlBody missing subject name")
	}
	if !strings.Contains(htmlBody, "https://solv.uab.edu.bo/student/courses/10/assignments/25") {
		t.Errorf("htmlBody missing review URL")
	}
	if !strings.Contains(textBody, "Laboratorio 03: Arboles AVL") {
		t.Errorf("textBody missing assignment title")
	}

	// Verificar ausencia de SOLV
	if strings.Contains(htmlBody, "SOLV") || strings.Contains(textBody, "SOLV") {
		t.Errorf("template leaked SOLV internal brand")
	}
}

func TestDeadlineReminderEmail_WithAndWithoutWorkspace(t *testing.T) {
	branding := domain.EmailBranding{
		InstitutionName: "Universidad Adventista de Bolivia",
		BaseDomain:      "solv.uab.edu.bo",
	}

	// Caso 1: Estudiante con workspace iniciado (borrador guardado)
	htmlWithWs, textWithWs := mailer.BuildDeadlineReminderEmail(
		branding,
		"Lucas Silva",
		"Laboratorio 05: Docker y Redes",
		"Sistemas Operativos II",
		"27 de Octubre, 23:59 (GMT-4)",
		true,
		"https://solv.uab.edu.bo/student/workspace/42",
	)

	if !strings.Contains(htmlWithWs, "Registramos avances guardados") {
		t.Errorf("htmlWithWs should mention saved progress")
	}
	if !strings.Contains(textWithWs, "Registramos avances guardados") {
		t.Errorf("textWithWs should mention saved progress")
	}
	if !strings.Contains(htmlWithWs, "27 de Octubre, 23:59 (GMT-4)") {
		t.Errorf("htmlWithWs should contain formatted deadline")
	}

	// Caso 2: Estudiante sin workspace iniciado
	htmlNoWs, textNoWs := mailer.BuildDeadlineReminderEmail(
		branding,
		"Lucas Silva",
		"Laboratorio 05: Docker y Redes",
		"Sistemas Operativos II",
		"27 de Octubre, 23:59 (GMT-4)",
		false,
		"https://solv.uab.edu.bo/student/workspace/42",
	)

	if !strings.Contains(htmlNoWs, "no registra actividad ni entrega enviada") {
		t.Errorf("htmlNoWs should notify about no activity")
	}
	if !strings.Contains(textNoWs, "no registra actividad ni entrega enviada") {
		t.Errorf("textNoWs should notify about no activity")
	}

	// Ninguno debe fugar la marca SOLV
	if strings.Contains(htmlWithWs, "SOLV") || strings.Contains(htmlNoWs, "SOLV") {
		t.Errorf("templates leaked SOLV brand")
	}
}

func TestTemplates_XSSPrevention(t *testing.T) {
	maliciousBranding := domain.EmailBranding{
		InstitutionName: `<script>alert('xss')</script>Universidad`,
		BaseDomain:      "solv.uab.edu.bo",
		LogoURL:         `"><script>alert('img')</script>`,
	}

	htmlBody, _ := mailer.BuildGradeFeedbackEmail(
		maliciousBranding,
		`<b onmouseover="alert('xss')">Hacker</b>`,
		`Lab <b>Bold</b>`,
		`Materia <script>`,
		`https://solv.uab.edu.bo/review?param=1&other=2`,
	)

	if strings.Contains(htmlBody, "<script>") || strings.Contains(htmlBody, "<script") {
		t.Errorf("htmlBody contains unescaped script tag: %s", htmlBody)
	}
	if strings.Contains(htmlBody, `"><script>`) {
		t.Errorf("htmlBody contains unescaped image script: %s", htmlBody)
	}
	if strings.Contains(htmlBody, "<b onmouseover") {
		t.Errorf("htmlBody contains unescaped HTML element tag with attributes: %s", htmlBody)
	}
}
