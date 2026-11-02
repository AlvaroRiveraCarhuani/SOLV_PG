import { KeystrokeEvent, KeystrokeEventType } from '../domain/keystroke';

const MAX_EVENT_CONTENT = 4000;

/**
 * Un cambio con texto igual o mayor al umbral es pegado, aunque el editor lo
 * reporte como reemplazo. Un salto de línea insertado de golpe también se marca
 * como pegado cuando el bloque es grande.
 */
export function classifyChange(
  insertedText: string,
  replacedLength: number,
  pasteThreshold: number
): KeystrokeEventType {
  if (insertedText.length === 0) {
    return 'delete';
  }
  if (insertedText.length >= pasteThreshold) {
    return 'paste';
  }
  if (replacedLength > 0 && insertedText.includes('\n') && insertedText.length >= 10) {
    return 'paste';
  }
  return 'insert';
}

export function buildKeystrokeEvent(
  insertedText: string,
  replacedLength: number,
  position: number,
  timestampMs: number,
  pasteThreshold: number
): KeystrokeEvent {
  const eventType = classifyChange(insertedText, replacedLength, pasteThreshold);
  return {
    timestamp_ms: timestampMs,
    event_type: eventType,
    position,
    content: insertedText.slice(0, MAX_EVENT_CONTENT),
    paste_source_detected: eventType === 'paste'
  };
}
