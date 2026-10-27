package mailer

import (
	"context"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestSMTPMailer_MockMode(t *testing.T) {
	cfg := Config{
		Host: "mock",
		Port: 587,
		From: "soporte@uab.edu.bo",
	}

	m := NewSMTPMailer(cfg, 10, 1)
	defer func() { _ = m.Close() }()

	msg := &domain.EmailMessage{
		To:       "profesor.prueba@uab.edu.bo",
		Subject:  "Bienvenido a SOLV",
		TextBody: "Cuerpo de prueba",
		HTMLBody: "<h1>Cuerpo HTML</h1>",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.Send(ctx, msg); err != nil {
		t.Fatalf("Expected mock send to succeed, got %v", err)
	}
}

func TestSMTPMailer_AsyncQueue(t *testing.T) {
	cfg := Config{
		Host: "", // Vacío -> Modo mock
		Port: 587,
	}

	m := NewSMTPMailer(cfg, 50, 2)
	defer func() { _ = m.Close() }()

	for i := 0; i < 5; i++ {
		m.SendAsync(&domain.EmailMessage{
			To:       "docente@uab.edu.bo",
			Subject:  "Notificación de acceso",
			TextBody: "Acceso activado",
		})
	}

	// Permitir que el worker procese
	time.Sleep(50 * time.Millisecond)
}

func TestSMTPMailer_BuildMIME(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
		From: "no-reply@uab.edu.bo",
	}
	m := NewSMTPMailer(cfg, 1, 1)
	defer func() { _ = m.Close() }()

	msg := &domain.EmailMessage{
		To:       "test@uab.edu.bo",
		Subject:  "Asunto con acento áéíóú",
		TextBody: "Texto plano",
		HTMLBody: "<p>HTML plano</p>",
	}

	raw := string(m.buildMIMEMessage(msg))
	if !strings.Contains(raw, "To: test@uab.edu.bo") {
		t.Errorf("MIME missing recipient: %s", raw)
	}
	if !strings.Contains(raw, "From: no-reply@uab.edu.bo") {
		t.Errorf("MIME missing sender: %s", raw)
	}
	if !strings.Contains(raw, "multipart/alternative") {
		t.Errorf("MIME missing multipart header: %s", raw)
	}
}

func TestBuildTeacherWelcomeEmail(t *testing.T) {
	branding := domain.EmailBranding{
		InstitutionName: "Universidad Adventista de Bolivia",
		BaseDomain:      "solv.uab.edu.bo",
		LogoURL:         "/assets/uab-logo.png",
	}
	htmlBody, textBody := BuildTeacherWelcomeEmail(branding, "Elena Docente", "https://solv.uab.edu.bo/login")

	if !strings.Contains(htmlBody, "Universidad Adventista de Bolivia") {
		t.Errorf("HTML template missing institution name")
	}
	if !strings.Contains(htmlBody, "Elena Docente") {
		t.Errorf("HTML template missing teacher name")
	}
	if !strings.Contains(htmlBody, "https://solv.uab.edu.bo/login") {
		t.Errorf("HTML template missing login link")
	}
	if !strings.Contains(textBody, "https://solv.uab.edu.bo/login") {
		t.Errorf("Text template missing login link")
	}
}
