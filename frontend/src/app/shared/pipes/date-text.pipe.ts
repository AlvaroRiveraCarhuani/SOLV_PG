import { Pipe, PipeTransform } from '@angular/core';

/**
 * Canon de fechas de SOLV (ver solv-design-system):
 * - ÚNICA vía de formateo: nada de toLocaleDateString/Intl/`| date:` en vistas.
 * - Salida determinista en español fijo: no depende del locale del runtime.
 * - La FUENTE MONO la aplica el contenedor (var(--font-mono) en SCSS o
 *   utilidad global .font-mono en el elemento). Una celda de fecha sin mono
 *   es bug de UI, igual que un hex suelto.
 */
export type DateTextStyle =
  | 'datetime'          // dd/MM/yyyy HH:mm
  | 'datetime-sec'      // dd/MM/yyyy HH:mm:ss
  | 'compact-datetime'  // dd/MM HH:mm:ss
  | 'date'              // dd/MM/yyyy
  | 'time'              // HH:mm
  | 'daymonth'          // dd MMM
  | 'daymonthyear'      // dd MMM yyyy
  | 'long-datetime';    // EEEE dd 'de' MMM, HH:mm:ss

const MONTHS_ES_SHORT = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'] as const;
const WEEKDAYS_ES_LONG = ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'] as const;

const DATE_PREFIX_RE = /^(\d{4})-(\d{2})-(\d{2})/;

function pad2(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

interface YmdParts { y: number; m: number; day: number }

/**
 * Extrae la parte de fecha de un string ISO (con o sin hora) validando
 * rollover. Se usa para estilos de fecha pura: el día mostrado es el literal
 * del string, sin conversión de zona (determinista en cualquier TZ).
 */
function ymdFromString(raw: string): YmdParts | null {
  const m = DATE_PREFIX_RE.exec(raw);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const day = Number(m[3]);
  const d = new Date(y, mo - 1, day);
  if (d.getFullYear() !== y || d.getMonth() !== mo - 1 || d.getDate() !== day) return null;
  return { y, m: mo, day };
}

/**
 * Convierte un valor de fecha (ISO string, Date o epoch) a Date local.
 * Fecha pura YYYY-MM-DD se interpreta en hora local para evitar corrimiento
 * UTC; devuelve null si el valor no parsea o es una fecha inválida.
 */
export function parseDateValue(value: unknown): Date | null {
  if (value instanceof Date) return isNaN(value.getTime()) ? null : value;
  if (typeof value === 'number' && Number.isFinite(value)) return new Date(value);
  if (typeof value !== 'string') return null;
  const raw = value.trim();
  if (raw === '') return null;
  const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(raw);
  if (dateOnly) {
    const d = new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]));
    // Rechaza rollover: 2026-02-30 no es una fecha válida
    if (
      d.getFullYear() !== Number(dateOnly[1]) ||
      d.getMonth() !== Number(dateOnly[2]) - 1 ||
      d.getDate() !== Number(dateOnly[3])
    ) {
      return null;
    }
    return d;
  }
  const d = new Date(raw);
  return isNaN(d.getTime()) ? null : d;
}

/** Formateador canónico determinista. null si el valor no parsea. */
export function formatSolvDate(value: unknown, style: DateTextStyle = 'datetime'): string | null {
  const d = parseDateValue(value);
  if (!d) return null;

  const DATE_ONLY_STYLES: readonly DateTextStyle[] = ['date', 'daymonth', 'daymonthyear'];
  const isDateOnly = DATE_ONLY_STYLES.includes(style);

  // Estilos de fecha pura: el día es el literal del string ISO (sin TZ shift).
  // Evita el off-by-one de medianoche UTC en zonas UTC-4 como America/La_Paz.
  let dayNum = d.getDate();
  let monthNum = d.getMonth() + 1;
  let yearNum = d.getFullYear();
  let weekdayNum = d.getDay();
  if (isDateOnly && typeof value === 'string') {
    const parts = ymdFromString(value.trim());
    if (parts) {
      dayNum = parts.day;
      monthNum = parts.m;
      yearNum = parts.y;
      weekdayNum = new Date(parts.y, parts.m - 1, parts.day).getDay();
    }
  }

  const dd = pad2(dayNum);
  const MM = pad2(monthNum);
  const yyyy = yearNum;
  const HH = pad2(d.getHours());
  const mm = pad2(d.getMinutes());
  const ss = pad2(d.getSeconds());
  const mon = MONTHS_ES_SHORT[monthNum - 1];
  switch (style) {
    case 'datetime-sec':
      return `${dd}/${MM}/${yyyy} ${HH}:${mm}:${ss}`;
    case 'compact-datetime':
      return `${dd}/${MM} ${HH}:${mm}:${ss}`;
    case 'date':
      return `${dd}/${MM}/${yyyy}`;
    case 'time':
      return `${HH}:${mm}`;
    case 'daymonth':
      return `${dd} ${mon}`;
    case 'daymonthyear':
      return `${dd} ${mon} ${yyyy}`;
    case 'long-datetime':
      return `${WEEKDAYS_ES_LONG[weekdayNum]} ${dd} de ${mon}, ${HH}:${mm}:${ss}`;
    case 'datetime':
    default:
      return `${dd}/${MM}/${yyyy} ${HH}:${mm}`;
  }
}

/**
 * Pipe único de fechas. Valores vacíos devuelven el parámetro `empty`
 * (default ''); valores no parseables pasan sin cambios (ej. 'Hace 2d').
 */
@Pipe({ name: 'dateText' })
export class DateTextPipe implements PipeTransform {
  transform(value: unknown, style: DateTextStyle = 'datetime', empty = ''): string {
    const formatted = formatSolvDate(value, style);
    if (formatted !== null) return formatted;
    if (value === null || value === undefined) return empty;
    const raw = String(value).trim();
    return raw === '' ? empty : raw;
  }
}
