import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi, describe, beforeEach, afterEach, it, expect } from 'vitest';
import { ComplexityBenchmarkComponent } from './complexity-benchmark.component';
import { BenchmarkReport } from '../../../features/teacher/models/teacher.models';

describe('ComplexityBenchmarkComponent', () => {
  let component: ComplexityBenchmarkComponent;
  let fixture: ComponentFixture<ComplexityBenchmarkComponent>;

  const mockReport: BenchmarkReport = {
    submission_id: 'sub-101',
    exercise_id: 'ex-202',
    student_name: 'Estudiante Prueba',
    expected_time_complexity: 'O(N log N)',
    detected_time_complexity: 'O(N^2)',
    expected_space_complexity: 'O(1)',
    detected_space_complexity: 'O(1)',
    r_squared: 0.985,
    is_optimal: false,
    summary: 'Se detectó una complejidad cuadrática O(N²) debido a bucles anidados.',
    analyzed_at: '2026-10-03T14:30:00Z',
    samples: [
      { input_size: 10, execution_time_ms: 0.12, memory_used_kb: 120, status: 'pass' },
      { input_size: 100, execution_time_ms: 1.45, memory_used_kb: 140, status: 'pass' },
      { input_size: 1000, execution_time_ms: 45.2, memory_used_kb: 210, status: 'pass' },
      { input_size: 10000, execution_time_ms: 1250.0, memory_used_kb: 512, status: 'pass' }
    ]
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ComplexityBenchmarkComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(ComplexityBenchmarkComponent);
    component = fixture.componentInstance;
    component.report = mockReport;
    component.submissionId = 'sub-101';
    fixture.detectChanges();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('debe crearse y computar los puntos del gráfico SVG correctamente', () => {
    expect(component).toBeTruthy();
    expect(component.hasData()).toBe(true);
    expect(component.isOptimal()).toBe(false);
    expect(component.chartPoints().length).toBe(4);
    expect(component.empiricalPath()).toContain('M ');
  });

  it('debe alternar métricas entre tiempo y memoria', () => {
    expect(component.activeMetric()).toBe('time');
    component.activeMetric.set('memory');
    expect(component.activeMetric()).toBe('memory');

    const pts = component.chartPoints();
    expect(pts[0].label).toContain('KB');
  });

  it('debe alternar escala entre lineal y logarítmica', () => {
    expect(component.scaleType()).toBe('linear');
    component.scaleType.set('log');
    expect(component.scaleType()).toBe('log');

    const pts = component.chartPoints();
    expect(pts.length).toBe(4);
  });

  it('debe emitir evento runBenchmark al ejecutar preset', () => {
    const runSpy = vi.spyOn(component.runBenchmark, 'emit');
    component.onExecuteBenchmark('stress');

    expect(component.selectedPreset()).toBe('stress');
    expect(runSpy).toHaveBeenCalledWith('stress');
  });

  it('debe alternar curvas teóricas de referencia', () => {
    expect(component.showTheoreticalBounds()).toBe(true);
    expect(component.theoreticalCurves().length).toBe(2);

    component.showTheoreticalBounds.set(false);
    expect(component.theoreticalCurves().length).toBe(0);
  });

  it('debe exportar datos de muestras a archivo CSV', () => {
    if (typeof URL.createObjectURL !== 'function') {
      URL.createObjectURL = vi.fn().mockReturnValue('blob:csv');
      URL.revokeObjectURL = vi.fn();
    } else {
      vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:csv');
      vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
    }

    const clickMock = vi.fn();
    const origCreateElement = document.createElement.bind(document);
    vi.spyOn(document, 'createElement').mockImplementation((tagName: string, options?: ElementCreationOptions) => {
      if (tagName.toLowerCase() === 'a') {
        const el = origCreateElement('a');
        el.click = clickMock;
        return el;
      }
      return origCreateElement(tagName, options);
    });

    component.exportDataCsv();
    expect(clickMock).toHaveBeenCalled();
  });

  it('debe emitir cierre cuando se invoca onClose', () => {
    const closeSpy = vi.spyOn(component.close, 'emit');
    component.onClose();
    expect(closeSpy).toHaveBeenCalled();
  });
});
