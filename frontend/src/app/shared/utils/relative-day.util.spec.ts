import { describe, expect, it } from 'vitest';
import { relativeDayLabel } from './relative-day.util';

describe('relative-day.util', () => {
  const now = new Date(2026, 10, 22, 10, 0, 0);

  it('marca Mañana cuando el día mostrado es el siguiente aunque falten menos de 24 h', () => {
    expect(relativeDayLabel('2026-11-23T00:00:00Z', now)).toEqual({
      label: 'Mañana',
      tone: 'tomorrow'
    });
  });

  it('marca Hoy solo cuando la fecha literal coincide con el día local', () => {
    expect(relativeDayLabel('2026-11-22T23:30:00Z', now)).toEqual({
      label: 'Hoy',
      tone: 'today'
    });
  });

  it('marca Vencida y cuenta días calendario hacia atrás', () => {
    expect(relativeDayLabel('2026-11-21T12:00:00Z', now).tone).toBe('overdue');
    expect(relativeDayLabel('2026-11-21T12:00:00Z', now).label).toBe('Vencida');
  });

  it('cuenta los días calendario restantes sin redondear por horas', () => {
    expect(relativeDayLabel('2026-11-25T23:59:00Z', now).label).toBe('En 3 días');
    expect(relativeDayLabel('2026-12-02T00:00:00Z', now).label).toBe('En 10 días');
  });

  it('devuelve etiqueta vacía para valores ausentes o inválidos', () => {
    expect(relativeDayLabel(undefined, now)).toEqual({ label: '', tone: 'unknown' });
    expect(relativeDayLabel('sin-fecha', now).tone).toBe('unknown');
  });
});
