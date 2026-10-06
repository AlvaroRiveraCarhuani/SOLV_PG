-- +goose Up
-- Migration 00018: Tabla para auditoría forense de eventos de escritura y pegado (Time-Travel Replay)
CREATE TABLE IF NOT EXISTS submission_keystroke_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  timestamp_ms BIGINT NOT NULL,
  event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('insert', 'delete', 'paste')),
  position INT NOT NULL,
  content TEXT,
  paste_source_detected BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_keystroke_submission 
  ON submission_keystroke_events(submission_id, timestamp_ms);

-- +goose Down
DROP INDEX IF EXISTS idx_keystroke_submission;
DROP TABLE IF EXISTS submission_keystroke_events;
