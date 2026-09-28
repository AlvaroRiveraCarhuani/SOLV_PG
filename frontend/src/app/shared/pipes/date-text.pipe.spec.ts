import { DateTextPipe, formatSolvDate, parseDateValue } from './date-text.pipe';

describe('DateTextPipe (canon de fechas SOLV)', () => {
  const pipe = new DateTextPipe();

  describe('formatos canónicos deterministas', () => {
    it('datetime default: dd/MM/yyyy HH:mm', () => {
      expect(pipe.transform('2026-09-15T14:05:00')).toBe('15/09/2026 14:05');
    });

    it('datetime-sec: dd/MM/yyyy HH:mm:ss', () => {
      expect(pipe.transform('2026-01-03T09:41:07', 'datetime-sec')).toBe('03/01/2026 09:41:07');
    });

    it('compact-datetime: dd/MM HH:mm:ss (tabla de auditoría)', () => {
      expect(pipe.transform('2026-09-15T08:07:09', 'compact-datetime')).toBe('15/09 08:07:09');
    });

    it('date: dd/MM/yyyy', () => {
      expect(pipe.transform('2026-09-15T00:00:00Z', 'date')).toBe('15/09/2026');
    });

    it('time: HH:mm', () => {
      expect(pipe.transform('2026-09-15T08:07:00', 'time')).toBe('08:07');
    });

    it('daymonth / daymonthyear en español fijo (independiente del locale del runtime)', () => {
      expect(pipe.transform('2026-02-15', 'daymonthyear')).toBe('15 feb 2026');
      expect(pipe.transform('2026-02-15', 'daymonth')).toBe('15 feb');
    });

    it('long-datetime: EEEE dd de MMM, HH:mm:ss', () => {
      // 2026-09-15 es martes
      expect(pipe.transform('2026-09-15T14:05:09', 'long-datetime')).toBe('martes 15 de sep, 14:05:09');
    });

    it('fecha pura YYYY-MM-DD se interpreta en hora local (sin corrimiento UTC)', () => {
      expect(pipe.transform('2026-09-15', 'date')).toBe('15/09/2026');
    });

    it('cero-padding consistente', () => {
      expect(pipe.transform('2026-03-05T04:02:03', 'datetime-sec')).toBe('05/03/2026 04:02:03');
    });

    it('medianoche UTC no corre el día en zonas UTC-4 (parte de fecha literal)', () => {
      expect(pipe.transform('2026-09-15T00:00:00Z', 'date')).toBe('15/09/2026');
      expect(pipe.transform('2026-09-15T00:00:00Z', 'daymonthyear')).toBe('15 sep 2026');
    });
  });

  describe('valores vacíos y no parseables', () => {
    it('null/undefined devuelven el parámetro empty', () => {
      expect(pipe.transform(null)).toBe('');
      expect(pipe.transform(undefined)).toBe('');
      expect(pipe.transform(null, 'date', 'Sin registros')).toBe('Sin registros');
    });

    it('string vacío devuelve el parámetro empty', () => {
      expect(pipe.transform('', 'date', '—')).toBe('—');
    });

    it('no parseable pasa sin cambios (ej. tiempos relativos "Hace 2d")', () => {
      expect(pipe.transform('Hace 2d')).toBe('Hace 2d');
      expect(pipe.transform('Reciente')).toBe('Reciente');
    });

    it('fecha inválida (rollover) devuelve null desde formatSolvDate', () => {
      expect(formatSolvDate('2026-02-30', 'date')).toBeNull();
      expect(parseDateValue('2026-02-30')).toBeNull();
    });
  });
});
