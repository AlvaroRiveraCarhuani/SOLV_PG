import { describe, it, expect } from 'vitest';
import {
  auditActionLabel,
  auditResourceLabel,
  describeStatus,
  enrichAction
} from './audit-log.model';

describe('Etiquetas del registro de auditoría', () => {
  it('traduce las acciones históricas de avance de clase y congelamiento', () => {
    expect(auditActionLabel('CLASS_ADVANCE_APPLIED')).toBe('Avance de clase aplicado');
    expect(auditActionLabel('CLASS_ADVANCE_EMITTED')).toBe('Avance de clase emitido');
    expect(auditActionLabel('CLASS_ADVANCE_DISMISSED')).toBe('Avance de clase descartado');
    expect(auditActionLabel('CLASS_FROZEN')).toBe('Sesión de clase congelada');
    expect(auditActionLabel('CLASS_UNFROZEN')).toBe('Sesión de clase descongelada');
    expect(auditActionLabel('CLASS_ADVANCE_GRACE_SET')).toBe('Plazo de gracia de clase fijado');
    expect(auditActionLabel('CLASS_ADVANCE_GRACE_CLOSED')).toBe('Plazo de gracia de clase cerrado');
  });

  it('traduce eventos de plantillas, gobernanza y reconciliación', () => {
    expect(auditActionLabel('TEMPLATE_COMPONENT_CREATED')).toBe('Tecnología creada');
    expect(auditActionLabel('IMAGE_VERIFICATION_FORCE_BYPASS')).toBe('Omisión forzada de verificación de imagen');
    expect(auditActionLabel('ADMINISTRATOR_CREATED')).toBe('Administrador creado');
    expect(auditActionLabel('ADMINISTRATOR_REVOKED')).toBe('Administrador revocado');
    expect(auditActionLabel('ORPHAN_VOLUME_RELEASED')).toBe('Volumen huérfano liberado');
    expect(auditActionLabel('PRACTICE_ENVIRONMENT_RECONCILED_EXITED')).toBe('Entorno reconciliado como detenido');
    expect(auditActionLabel('SUBMISSION_REVIEW_UPDATED')).toBe('Revisión de entrega actualizada');
    expect(auditActionLabel('EXERCISE_GRADE_SELECTED')).toBe('Calificación de ejercicio seleccionada');
    expect(auditActionLabel('MAINTENANCE_ENABLED')).toBe('Mantenimiento activado');
  });

  it('traduce recursos en español con auditResourceLabel', () => {
    expect(auditResourceLabel('class_session')).toBe('Sesión de clase');
    expect(auditResourceLabel('template_component')).toBe('Tecnología');
    expect(auditResourceLabel('lab_template')).toBe('Plantilla');
    expect(auditResourceLabel('template_category')).toBe('Categoría');
    expect(auditResourceLabel('template_model')).toBe('Modelo');
    expect(auditResourceLabel('emergency_action')).toBe('Acción de emergencia');
    expect(auditResourceLabel('environment')).toBe('Entorno');
    expect(auditResourceLabel('user')).toBe('Usuario');
    expect(auditResourceLabel('docker_volume')).toBe('Volumen Docker');
    expect(auditResourceLabel('docker_image')).toBe('Imagen Docker');
    expect(auditResourceLabel('exercise')).toBe('Ejercicio');
    expect(auditResourceLabel('submission')).toBe('Entrega');
    expect(auditResourceLabel('maintenance')).toBe('Mantenimiento');
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

  it('aplica silencio visual en describeStatus para 2xx (neutral) y alerta en 4xx/5xx', () => {
    const s200 = describeStatus(200);
    expect(s200.label).toBe('200 Exitoso');
    expect(s200.kind).toBe('neutral');

    const s201 = describeStatus(201);
    expect(s201.label).toBe('201 Creado');
    expect(s201.kind).toBe('neutral');

    const s204 = describeStatus(204);
    expect(s204.label).toBe('204 Eliminado');
    expect(s204.kind).toBe('neutral');

    const s403 = describeStatus(403);
    expect(s403.label).toBe('403 Denegado');
    expect(s403.kind).toBe('warning');

    const s500 = describeStatus(500);
    expect(s500.label).toBe('500 Error interno');
    expect(s500.kind).toBe('error');

    const s502 = describeStatus(502);
    expect(s502.label).toBe('502 Error de pasarela');
    expect(s502.kind).toBe('error');
  });
});
