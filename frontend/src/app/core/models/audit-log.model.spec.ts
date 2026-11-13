import { describe, it, expect } from 'vitest';
import { auditActionLabel, enrichAction } from './audit-log.model';

describe('Etiquetas del registro de auditoría', () => {
  it('traduce las acciones históricas de avance de clase', () => {
    expect(auditActionLabel('CLASS_ADVANCE_APPLIED')).toBe('Avance de clase aplicado');
    expect(auditActionLabel('CLASS_ADVANCE_EMITTED')).toBe('Avance de clase emitido');
    expect(auditActionLabel('CLASS_ADVANCE_DISMISSED')).toBe('Avance de clase descartado');
  });

  it('conserva el identificador técnico y sólo traduce la etiqueta', () => {
    const enriched = enrichAction('CLASS_ADVANCE_EMITTED');
    expect(enriched.label).toBe('Avance de clase emitido');
    expect(enriched.kind).toBe('other');
  });

  it('traduce las acciones de emergencia con el mismo catálogo', () => {
    expect(auditActionLabel('EMERGENCY_HIBERNATE_ALL')).toBe('Emergencia: hibernación total');
    expect(auditActionLabel('EMERGENCY_TERMINATE_ALL')).toBe('Emergencia: terminación de todos los entornos');
  });

  it('mantiene el respaldo para verbos HTTP y eventos sin catálogo', () => {
    expect(enrichAction('POST /api/admin/templates').label).toBe('Creación');
    expect(auditActionLabel('FUTURE_EVENT_WITHOUT_LABEL')).toBe('Future Event Without Label');
  });
});
