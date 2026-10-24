import { describe, it, expect } from 'vitest';
import { getVerdictHumanLabel, getVerdictBadgeClass, matchesVerdictFilter } from '../../../core/models/verdict.model';

describe('VerdictBadge mapping logic', () => {
  it('debe mapear AC a etiqueta humana Solución correcta y clase verdict-ac', () => {
    expect(getVerdictHumanLabel('AC')).toBe('Solución correcta');
    expect(getVerdictBadgeClass('AC')).toBe('verdict-ac');
  });

  it('debe mapear WA a Resultado incorrecto y clase verdict-wa', () => {
    expect(getVerdictHumanLabel('WA')).toBe('Resultado incorrecto');
    expect(getVerdictBadgeClass('WA')).toBe('verdict-wa');
  });

  it('debe mapear TLE a Tiempo límite excedido y clase verdict-tle', () => {
    expect(getVerdictHumanLabel('TLE')).toBe('Tiempo límite excedido');
    expect(getVerdictBadgeClass('TLE')).toBe('verdict-tle');
  });

  it('debe mapear MLE a Límite de memoria excedido y clase verdict-tle', () => {
    expect(getVerdictHumanLabel('MLE')).toBe('Límite de memoria excedido');
    expect(getVerdictBadgeClass('MLE')).toBe('verdict-tle');
  });

  it('debe mapear RE a Error durante la ejecución y clase verdict-re', () => {
    expect(getVerdictHumanLabel('RE')).toBe('Error durante la ejecución');
    expect(getVerdictBadgeClass('RE')).toBe('verdict-re');
  });

  it('debe mapear CE a Error de compilación y clase verdict-re', () => {
    expect(getVerdictHumanLabel('CE')).toBe('Error de compilación');
    expect(getVerdictBadgeClass('CE')).toBe('verdict-re');
  });

  it('debe mapear VE a Error en validación de prueba y clase verdict-re', () => {
    expect(getVerdictHumanLabel('VE')).toBe('Error en validación de prueba');
    expect(getVerdictBadgeClass('VE')).toBe('verdict-re');
  });

  it('debe mapear AST_BLOCKED a Restricción de código incumplida y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_BLOCKED')).toBe('Restricción de código incumplida');
    expect(getVerdictBadgeClass('AST_BLOCKED')).toBe('verdict-ast');
  });

  it('debe mapear AST_VIOLATION a Restricción de código incumplida y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_VIOLATION')).toBe('Restricción de código incumplida');
    expect(getVerdictBadgeClass('AST_VIOLATION')).toBe('verdict-ast');
  });

  it('debe mapear PENDING a Evaluando entrega y clase verdict-pending', () => {
    expect(getVerdictHumanLabel('PENDING')).toBe('Evaluando entrega');
    expect(getVerdictBadgeClass('PENDING')).toBe('verdict-pending');
  });

  it('debe manejar entradas desconocidas o vacías con gracia', () => {
    expect(getVerdictHumanLabel('')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('')).toBe('verdict-unknown');
    expect(getVerdictHumanLabel('UNKNOWN_CODE')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('UNKNOWN_CODE')).toBe('verdict-unknown');
  });

  it('debe ser insensible a mayúsculas y espacios en blanco', () => {
    expect(getVerdictHumanLabel(' ac ')).toBe('Solución correcta');
    expect(getVerdictBadgeClass(' wa ')).toBe('verdict-wa');
  });
});

describe('matchesVerdictFilter', () => {
  it('acepta cualquier veredicto con el filtro all o vacío', () => {
    expect(matchesVerdictFilter('AC', 'all')).toBe(true);
    expect(matchesVerdictFilter('WA', 'all')).toBe(true);
    expect(matchesVerdictFilter(undefined, 'all')).toBe(true);
    expect(matchesVerdictFilter('AST_VIOLATION', '')).toBe(true);
  });

  it('agrupa AST_BLOCKED y AST_VIOLATION bajo el mismo filtro de restricción', () => {
    expect(matchesVerdictFilter('AST_BLOCKED', 'AST_BLOCKED')).toBe(true);
    expect(matchesVerdictFilter('AST_VIOLATION', 'AST_BLOCKED')).toBe(true);
    expect(matchesVerdictFilter('AST_BLOCKED', 'AST_VIOLATION')).toBe(true);
    expect(matchesVerdictFilter('AST_VIOLATION', 'AST_VIOLATION')).toBe(true);
    expect(matchesVerdictFilter('WA', 'AST_BLOCKED')).toBe(false);
  });

  it('exige coincidencia exacta en el resto de veredictos, sin distinguir mayúsculas ni espacios', () => {
    expect(matchesVerdictFilter('AC', 'AC')).toBe(true);
    expect(matchesVerdictFilter(' ac ', ' ac ')).toBe(true);
    expect(matchesVerdictFilter('WA', 'AC')).toBe(false);
    expect(matchesVerdictFilter(undefined, 'AC')).toBe(false);
    expect(matchesVerdictFilter(null, 'WA')).toBe(false);
  });
});
