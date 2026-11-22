import { isoCalendarDate } from '@shared/pipes/date-text.pipe';

export type RelativeDayTone = 'overdue' | 'today' | 'tomorrow' | 'upcoming' | 'unknown';

export interface RelativeDayLabel {
  label: string;
  tone: RelativeDayTone;
}

const MS_PER_DAY = 86_400_000;

/**
 * Etiqueta relativa por día calendario para fechas límite.
 *
 * Compara el día literal del valor ISO con el día local de `now`, la misma
 * base que usa `dateText:'daymonth'` al pintar la fecha. No usa ventanas de
 * horas ni convierte el instante a la zona del navegador: un vencimiento del
 * 23 nov sigue siendo «Mañana» el 22 nov aunque su hora UTC caiga dentro de
 * las próximas 24 h.
 */
export function relativeDayLabel(value: unknown, now: Date = new Date()): RelativeDayLabel {
  const due = isoCalendarDate(value);
  if (!due) {
    return { label: '', tone: 'unknown' };
  }

  const dueDay = Date.UTC(due.y, due.m - 1, due.day);
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  const dayDiff = Math.round((dueDay - today) / MS_PER_DAY);

  if (dayDiff < 0) return { label: 'Vencida', tone: 'overdue' };
  if (dayDiff === 0) return { label: 'Hoy', tone: 'today' };
  if (dayDiff === 1) return { label: 'Mañana', tone: 'tomorrow' };
  return { label: `En ${dayDiff} días`, tone: 'upcoming' };
}
