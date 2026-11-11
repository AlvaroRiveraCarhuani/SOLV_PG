import { describe, it, expect } from 'vitest';
import { TestBed } from '@angular/core/testing';
import {
  OUTCOME_COPY,
  getVerdictHumanLabel,
  getVerdictBadgeClass,
  getVerdictOutcomeDetail,
  matchesVerdictFilter,
  verdictOriginLabel,
  VERDICT_FILTER_OPTIONS,
  VERDICT_OVERRIDE_OPTIONS
} from '../../../core/models/verdict.model';
import { VerdictBadgeComponent } from './verdict-badge.component';

describe('VerdictBadge mapping logic', () => {
  it('debe mapear AC a Cumple y clase verdict-ac', () => {
    expect(getVerdictHumanLabel('AC')).toBe('Cumple');
    expect(getVerdictBadgeClass('AC')).toBe('verdict-ac');
  });

  it('debe mapear WA a No cumple y clase verdict-wa', () => {
    expect(getVerdictHumanLabel('WA')).toBe('No cumple');
    expect(getVerdictBadgeClass('WA')).toBe('verdict-wa');
  });

  it('debe mapear TLE a Tarda demasiado y clase verdict-tle', () => {
    expect(getVerdictHumanLabel('TLE')).toBe('Tarda demasiado');
    expect(getVerdictBadgeClass('TLE')).toBe('verdict-tle');
  });

  it('debe mapear MLE a Se quedó sin memoria y clase verdict-tle', () => {
    expect(getVerdictHumanLabel('MLE')).toBe('Se quedó sin memoria');
    expect(getVerdictBadgeClass('MLE')).toBe('verdict-tle');
  });

  it('debe mapear RE a Se cae al ejecutar y clase verdict-re', () => {
    expect(getVerdictHumanLabel('RE')).toBe('Se cae al ejecutar');
    expect(getVerdictBadgeClass('RE')).toBe('verdict-re');
  });

  it('debe mapear CE a No compila y clase verdict-re', () => {
    expect(getVerdictHumanLabel('CE')).toBe('No compila');
    expect(getVerdictBadgeClass('CE')).toBe('verdict-re');
  });

  it('debe mapear VE a Caso de prueba inválido y clase verdict-re', () => {
    expect(getVerdictHumanLabel('VE')).toBe('Caso de prueba inválido');
    expect(getVerdictBadgeClass('VE')).toBe('verdict-re');
  });

  it('debe mapear AST_BLOCKED a Usó algo no permitido y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_BLOCKED')).toBe('Usó algo no permitido');
    expect(getVerdictBadgeClass('AST_BLOCKED')).toBe('verdict-ast');
  });

  it('debe mapear AST_VIOLATION a Usó algo no permitido y clase verdict-ast', () => {
    expect(getVerdictHumanLabel('AST_VIOLATION')).toBe('Usó algo no permitido');
    expect(getVerdictBadgeClass('AST_VIOLATION')).toBe('verdict-ast');
  });

  it('debe mapear PENDING a Sin calificar y clase verdict-pending', () => {
    expect(getVerdictHumanLabel('PENDING')).toBe('Sin calificar');
    expect(getVerdictBadgeClass('PENDING')).toBe('verdict-pending');
  });

  it('debe manejar entradas desconocidas o vacías con gracia', () => {
    expect(getVerdictHumanLabel('')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('')).toBe('verdict-unknown');
    expect(getVerdictHumanLabel('UNKNOWN_CODE')).toBe('Sin evaluar');
    expect(getVerdictBadgeClass('UNKNOWN_CODE')).toBe('verdict-unknown');
  });

  it('debe ser insensible a mayúsculas y espacios en blanco', () => {
    expect(getVerdictHumanLabel(' ac ')).toBe('Cumple');
    expect(getVerdictBadgeClass(' wa ')).toBe('verdict-wa');
  });

  it('expone un detalle humano y distinto para cada resultado', () => {
    const details = Object.values(OUTCOME_COPY).map((outcome) => outcome.detail);
    expect(new Set(details).size).toBe(details.length);
    expect(getVerdictOutcomeDetail('CE')).toContain('compilar');
  });
});

describe('Vocabulario derivado de la tabla canónica', () => {
  it('los filtros toman sus etiquetas de OUTCOME_COPY', () => {
    const filterLabels = VERDICT_FILTER_OPTIONS.map((option) => option.label);
    expect(filterLabels[0]).toBe('Todos los resultados');
    expect(filterLabels).toContain(OUTCOME_COPY.WA.label);
    expect(filterLabels).toContain(OUTCOME_COPY.AST_BLOCKED.label);
    expect(filterLabels).not.toContain(OUTCOME_COPY.UNKNOWN.label);
  });

  it('los overrides toman sus etiquetas de OUTCOME_COPY', () => {
    expect(VERDICT_OVERRIDE_OPTIONS.map((option) => option.label)).toEqual([
      OUTCOME_COPY.AC.label,
      OUTCOME_COPY.WA.label,
      OUTCOME_COPY.RE.label,
      OUTCOME_COPY.TLE.label
    ]);
  });
});

describe('Procedencia del veredicto', () => {
  it('declara la procedencia cuando el veredicto no lo escribió el juez', () => {
    expect(verdictOriginLabel('rubric')).toBe('Calificado por rúbrica');
    expect(verdictOriginLabel('override')).toBe('Ajuste docente');
    expect(verdictOriginLabel('legacy')).toBe('Resultado heredado');
    expect(verdictOriginLabel(' LEGACY ')).toBe('Resultado heredado');
  });

  it('no declara procedencia para el juez ni para una entrega sin veredicto', () => {
    expect(verdictOriginLabel('judge')).toBeNull();
    expect(verdictOriginLabel('')).toBeNull();
    expect(verdictOriginLabel(null)).toBeNull();
    expect(verdictOriginLabel(undefined)).toBeNull();
  });

  it('conserva el resultado técnico y declara la procedencia heredada en el tooltip', () => {
    const fixture = TestBed.createComponent(VerdictBadgeComponent);
    fixture.componentRef.setInput('verdict', 'AC');
    fixture.componentRef.setInput('origin', 'legacy');
    fixture.detectChanges();

    const el = fixture.nativeElement as HTMLElement;
    const text = el.textContent ?? '';
    expect(text).toContain('Cumple');
    expect(text).not.toContain('Resultado heredado');
    const badge = el.querySelector('.verdict-badge');
    expect(badge?.getAttribute('title')).toContain('Resultado heredado');
  });

  it('no presenta un veredicto escrito por la rúbrica como resultado del juez', () => {
    const fixture = TestBed.createComponent(VerdictBadgeComponent);
    fixture.componentRef.setInput('verdict', 'AC');
    fixture.componentRef.setInput('origin', 'rubric');
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Calificado por rúbrica');
    expect(text).not.toContain('Cumple');
  });

  it('no atribuye procedencia cuando todavía no hay resultado', () => {
    const fixture = TestBed.createComponent(VerdictBadgeComponent);
    fixture.componentRef.setInput('verdict', 'PENDING');
    fixture.componentRef.setInput('origin', 'legacy');
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Sin calificar');
    expect(text).not.toContain('Resultado heredado');
  });

  it('usa el resultado canónico cuando el veredicto es del juez', () => {
    const fixture = TestBed.createComponent(VerdictBadgeComponent);
    fixture.componentRef.setInput('verdict', 'AC');
    fixture.componentRef.setInput('origin', 'judge');
    fixture.detectChanges();

    expect((fixture.nativeElement as HTMLElement).textContent).toContain('Cumple');
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
