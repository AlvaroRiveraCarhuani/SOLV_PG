package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

// Config define los parámetros de conexión al servidor SMTP
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	UseTLS   bool
}

// SMTPMailer implementa domain.Mailer con soporte para SMTP real y fallback mock
type SMTPMailer struct {
	cfg       Config
	isMock    bool
	queue     chan *domain.EmailMessage
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// NewSMTPMailer crea una nueva instancia del servicio de correo con workers asíncronos
func NewSMTPMailer(cfg Config, bufferSize int, workerCount int) *SMTPMailer {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	if workerCount <= 0 {
		workerCount = 2
	}

	isMock := cfg.Host == "" || cfg.Host == "mock" || cfg.Host == "stdout"
	ctx, cancel := context.WithCancel(context.Background())

	m := &SMTPMailer{
		cfg:    cfg,
		isMock: isMock,
		queue:  make(chan *domain.EmailMessage, bufferSize),
		ctx:    ctx,
		cancel: cancel,
	}

	for i := 0; i < workerCount; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}

	if isMock {
		log.Println("[SMTPMailer] Modo mock activo: los correos se imprimirán en el registro del sistema")
	} else {
		log.Printf("[SMTPMailer] Configurado con host %s:%d (desde: %s)", cfg.Host, cfg.Port, cfg.From)
	}

	return m
}

// NewSMTPMailerFromEnv inicializa el servicio leyendo variables de entorno del sistema
func NewSMTPMailerFromEnv() *SMTPMailer {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	useTLS := strings.EqualFold(os.Getenv("SMTP_TLS"), "true") || port == 465

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = os.Getenv("SMTP_USER")
	}
	if from == "" {
		from = "no-reply@uab.edu.bo"
	}

	cfg := Config{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     port,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		From:     from,
		UseTLS:   useTLS,
	}

	return NewSMTPMailer(cfg, 512, 4)
}

func (m *SMTPMailer) worker(workerID int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			// Drenar canal residual
			for len(m.queue) > 0 {
				msg := <-m.queue
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = m.Send(ctx, msg)
				cancel()
			}
			return
		case msg, ok := <-m.queue:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := m.Send(ctx, msg); err != nil {
				log.Printf("[SMTPWorker-%d] Error despachando correo a %s: %v", workerID, msg.To, err)
			}
			cancel()
		}
	}
}

// SendAsync encola un correo de forma asíncrona sin bloquear la petición HTTP
func (m *SMTPMailer) SendAsync(msg *domain.EmailMessage) {
	if msg == nil || strings.TrimSpace(msg.To) == "" {
		return
	}
	select {
	case m.queue <- msg:
	default:
		log.Printf("[SMTPMailer] Cola de correos llena; descartando mensaje para %s", msg.To)
	}
}

// Send despacha un mensaje de correo de forma síncrona
func (m *SMTPMailer) Send(ctx context.Context, msg *domain.EmailMessage) error {
	if msg == nil {
		return fmt.Errorf("mensaje nulo")
	}
	targetEmail := strings.TrimSpace(msg.To)
	if targetEmail == "" {
		return fmt.Errorf("destinatario vacío")
	}

	// 1. Modo Mock para desarrollo local
	if m.isMock {
		log.Printf("[SMTP Mock] Para: %s | Asunto: %s | Cuerpo: %s...",
			targetEmail,
			msg.Subject,
			truncate(msg.TextBody, 60),
		)
		return nil
	}

	// 2. Construcción MIME
	rawMsg := m.buildMIMEMessage(msg)
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	// 3. Envío con soporte para TLS directo (puerto 465) o STARTTLS (puerto 587)
	if m.cfg.UseTLS && m.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("error al conectar TLS con servidor SMTP: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			return fmt.Errorf("error creando cliente SMTP TLS: %w", err)
		}
		defer client.Close()

		if m.cfg.User != "" && m.cfg.Password != "" {
			auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("fallo de autenticación SMTP TLS: %w", err)
			}
		}

		if err := client.Mail(m.cfg.From); err != nil {
			return err
		}
		if err := client.Rcpt(targetEmail); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(rawMsg); err != nil {
			return err
		}
		return w.Close()
	}

	// Envío estándar STARTTLS
	var auth smtp.Auth
	if m.cfg.User != "" && m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.cfg.From, []string{targetEmail}, rawMsg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("error despachando correo vía SMTP (%s): %w", addr, err)
		}
		return nil
	}
}

func (m *SMTPMailer) buildMIMEMessage(msg *domain.EmailMessage) []byte {
	boundary := fmt.Sprintf("solv_boundary_%d", time.Now().UnixNano())
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(msg.Subject)))

	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("From: %s\r\n", m.cfg.From))
	b.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

	// Parte Texto Plano
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.TextBody)))
	b.WriteString("\r\n\r\n")

	// Parte HTML
	if msg.HTMLBody != "" {
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		b.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.HTMLBody)))
		b.WriteString("\r\n\r\n")
	}

	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return b.Bytes()
}

// Close finaliza los workers y drena la cola
func (m *SMTPMailer) Close() error {
	m.closeOnce.Do(func() {
		m.cancel()
		close(m.queue)
		m.wg.Wait()
	})
	return nil
}

func truncate(s string, maxLen int) string {
	clean := strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "")
	if len(clean) > maxLen {
		return clean[:maxLen] + "..."
	}
	return clean
}
