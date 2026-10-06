package domain

import (
	"context"
	"time"
)

type KeystrokeEventType string

const (
	KeystrokeEventInsert KeystrokeEventType = "insert"
	KeystrokeEventDelete KeystrokeEventType = "delete"
	KeystrokeEventPaste  KeystrokeEventType = "paste"
)

type SubmissionKeystrokeEvent struct {
	ID                  string             `json:"id" db:"id"`
	SubmissionID        string             `json:"submission_id" db:"submission_id"`
	TimestampMS         int64              `json:"timestamp_ms" db:"timestamp_ms"`
	EventType           KeystrokeEventType `json:"event_type" db:"event_type"`
	Position            int                `json:"position" db:"position"`
	Content             string             `json:"content" db:"content"`
	PasteSourceDetected bool               `json:"paste_source_detected" db:"paste_source_detected"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
}

type SubmissionKeystrokeReport struct {
	SubmissionID     string                     `json:"submission_id"`
	Events           []SubmissionKeystrokeEvent `json:"events"`
	TotalTimeMS      int64                      `json:"total_time_ms"`
	PasteCount       int                        `json:"paste_count"`
	PastePercentage  float64                    `json:"paste_percentage"`
	TotalCharsTyped  int                        `json:"total_chars_typed"`
	TotalCharsPasted int                        `json:"total_chars_pasted"`
}

type KeystrokeRepository interface {
	SaveBatch(ctx context.Context, submissionID string, events []SubmissionKeystrokeEvent) error
	GetBySubmission(ctx context.Context, submissionID string) ([]SubmissionKeystrokeEvent, error)
}
