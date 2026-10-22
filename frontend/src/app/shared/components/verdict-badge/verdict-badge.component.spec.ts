import { describe, it, expect } from 'vitest';
import { getVerdictHumanLabel, getVerdictBadgeClass } from '../../../core/models/verdict.model';

describe('VerdictBadge mapping logic', () => {
  it('debe mapear AC a etiqueta humana Correcto y clase verdict-ac', () => {
    expect(getVerdictHumanLabel('AC')).toBe('Correcto');
    expect(getVerdictBadgeClass('AC')).toBe('verdict-ac');
  });

  it('debe mapear WA a Respuesta incorrecta y clase verdict-wa', () => {
    expect(getVerdictHumanLabel('WA')).toBe('Respuesta incorrecta');
    expect(getVerdictBadgeClass('WA')).toBe('verdict-wa');
  });

  it('debe mapear TLE a Tiempo excedido y clase verdict-tle', () => {
    expect(getVerdictHumanLabel('TLE')).toBe('Tiempo excedido');
    expect(getVerdictBadgeClass('TLE')).toBe('verdict-tle');
  });

  it('debe mapear RE a Error de ejecución y clase verdict-re', () => {
    expect(getVerdictHumanLabel('RE')).toBe('Error de ejecución');
    expect(getVerdictBadgeClass('RE')).toBe('verdict-re');
  });

  it('debe mapear AST_BLOCKED a Regla de código y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_BLOCKED')).toBe('Regla de código');
    expect(getVerdictBadgeClass('AST_BLOCKED')).toBe('verdict-ast');
  });

  it('debe mapear AST_VIOLATION a Regla de código y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_VIOLATION')).toBe('Regla de código');
    expect(getVerdictBadgeClass('AST_VIOLATION')).toBe('verdict-ast');
  });

  it('debe mapear PENDING a Evaluando y clase verdict-pending', () => {
    expect(getVerdictHumanLabel('PENDING')).toBe('Evaluando');
    expect(getVerdictBadgeClass('PENDING')).toBe('verdict-pending');
  });

  it('debe manejar entradas desconocidas o vacías con gracia', () => {
    expect(getVerdictHumanLabel('')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('')).toBe('verdict-unknown');
    expect(getVerdictHumanLabel('UNKNOWN_CODE')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('UNKNOWN_CODE')).toBe('verdict-unknown');
  });

  it('debe ser insensible a mayúsculas y espacios en blanco', () => {
    expect(getVerdictHumanLabel(' ac ')).toBe('Correcto');
    expect(getVerdictBadgeClass(' wa ')).toBe('verdict-wa');
  });
});
