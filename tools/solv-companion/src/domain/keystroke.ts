export type KeystrokeEventType = 'insert' | 'delete' | 'paste';

/** Evento compatible con POST /api/v1/submissions/{id}/keystroke-events. */
export interface KeystrokeEvent {
  timestamp_ms: number;
  event_type: KeystrokeEventType;
  position: number;
  content: string;
  paste_source_detected: boolean;
}
