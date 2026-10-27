package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/mailer"
)

// DeadlineReminderWorker escanea periódicamente ejercicios próximos a vencer dentro de
// la ventana de 12 horas y notifica a los estudiantes que aún no registraron entrega.
type DeadlineReminderWorker struct {
	repo       domain.DeadlineReminderRepository
	tenantRepo domain.TenantRepository
	mailer     domain.Mailer
	ticker     *time.Ticker
	done       chan struct{}
	mu         sync.Mutex
	running    bool
}

func NewDeadlineReminderWorker(
	repo domain.DeadlineReminderRepository,
	tenantRepo domain.TenantRepository,
	m domain.Mailer,
) *DeadlineReminderWorker {
	return &DeadlineReminderWorker{
		repo:       repo,
		tenantRepo: tenantRepo,
		mailer:     m,
		done:       make(chan struct{}),
	}
}

// ProcessReminders ejecuta una pasada de verificación y despacho para una fecha de referencia.
func (w *DeadlineReminderWorker) ProcessReminders(ctx context.Context, now time.Time) (int, error) {
	if w.repo == nil || w.mailer == nil {
		return 0, nil
	}

	// Ventana de urgencia: ejercicios cuyo plazo vence entre el momento actual y las próximas 12 horas
	windowStart := now
	windowEnd := now.Add(12 * time.Hour)

	candidates, err := w.repo.FindPendingReminders(ctx, windowStart, windowEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch reminder candidates: %w", err)
	}

	if len(candidates) == 0 {
		return 0, nil
	}

	brandingCache := make(map[string]domain.EmailBranding)
	sentCount := 0

	for _, c := range candidates {
		if strings.TrimSpace(c.StudentEmail) == "" {
			continue
		}

		branding, ok := brandingCache[c.TenantID]
		if !ok {
			branding = domain.EmailBranding{
				InstitutionName: "Plataforma Académica",
			}
			if w.tenantRepo != nil {
				if t, err := w.tenantRepo.GetByID(ctx, c.TenantID); err == nil && t != nil {
					branding = t.ResolveBranding()
				}
			}
			brandingCache[c.TenantID] = branding
		}

		baseDomain := strings.TrimSpace(branding.BaseDomain)
		if baseDomain == "" {
			baseDomain = "localhost:4200"
		}

		workspaceURL := fmt.Sprintf("https://%s/student/courses/%s/assignments/%s", baseDomain, c.SubjectID, c.ExerciseID)
		formattedDate := c.DueDate.Format("02/01/2006 15:04")

		htmlBody, textBody := mailer.BuildDeadlineReminderEmail(
			branding,
			c.StudentName,
			c.ExerciseTitle,
			c.SubjectName,
			formattedDate,
			c.HasStartedWorkspace,
			workspaceURL,
		)

		w.mailer.SendAsync(&domain.EmailMessage{
			To:       c.StudentEmail,
			Subject:  fmt.Sprintf("[%s] Recordatorio: Próximo vencimiento de %s — %s", branding.InstitutionName, c.ExerciseTitle, c.SubjectName),
			HTMLBody: htmlBody,
			TextBody: textBody,
		})

		// Registrar inmediatamente para garantizar deduplicación idempotente
		if err := w.repo.MarkReminderSent(ctx, c.TenantID, c.ExerciseID, c.StudentID); err != nil {
			log.Printf("[DeadlineReminderWorker] Warning: could not mark reminder as sent for student %s on exercise %s: %v", c.StudentID, c.ExerciseID, err)
		}

		sentCount++
	}

	return sentCount, nil
}

// Start inicia la rutina de escaneo periódico en segundo plano.
func (w *DeadlineReminderWorker) Start(ctx context.Context, interval time.Duration) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	w.ticker = time.NewTicker(interval)
	w.running = true
	w.mu.Unlock()

	go func() {
		// Pasada inicial al arrancar el servicio
		if _, err := w.ProcessReminders(ctx, time.Now()); err != nil {
			log.Printf("[DeadlineReminderWorker] Error in initial reminder run: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				w.Stop()
				return
			case <-w.done:
				return
			case now := <-w.ticker.C:
				if _, err := w.ProcessReminders(ctx, now); err != nil {
					log.Printf("[DeadlineReminderWorker] Error in scheduled run: %v", err)
				}
			}
		}
	}()
}

// Stop detiene el worker y libera los recursos del ticker.
func (w *DeadlineReminderWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.done)
	w.running = false
}
