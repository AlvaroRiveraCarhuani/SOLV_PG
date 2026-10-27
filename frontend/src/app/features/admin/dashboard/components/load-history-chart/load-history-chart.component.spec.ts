import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { LoadHistoryChartComponent } from './load-history-chart.component';
import { LoadSnapshot } from '@core/models/admin.model';

const iso = (minutesAgo: number): string => {
  const d = new Date(Date.now() - minutesAgo * 60000);
  return d.toISOString();
};

function snap(ts: string, ram = 40, cpu = 20): LoadSnapshot {
  return { timestamp: ts, ram_percent: ram, ram_used_gb: 4.0, cpu_percent: cpu, active_containers: 2 };
}

function mount(history: LoadSnapshot[], threshold = 85): LoadHistoryChartComponent {
  const fixture = TestBed.createComponent(LoadHistoryChartComponent);
  fixture.componentRef.setInput('history', history);
  fixture.componentRef.setInput('threshold', threshold);
  fixture.detectChanges();
  return fixture.componentInstance;
}

describe('LoadHistoryChartComponent (límites y umbral dinámico)', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LoadHistoryChartComponent]
    }).compileComponents();
  });

  it('muestra el estado vacío con menos de 2 puntos reales', () => {
    const c0 = mount([]);
    expect(c0.hasEnoughData()).toBe(false);

    const c1 = mount([snap(iso(1))]);
    expect(c1.hasEnoughData()).toBe(false);

    const c2 = mount([snap(iso(2)), snap(iso(1))]);
    expect(c2.hasEnoughData()).toBe(true);
  });

  it('el umbral de la línea sale del input (85 → Y=24; 90 → Y=16)', () => {
    const c85 = mount([snap(iso(2)), snap(iso(1))], 85);
    expect(c85.thresholdY()).toBeCloseTo(160 - (85 / 100) * 160, 5);

    const c90 = mount([snap(iso(2)), snap(iso(1))], 90);
    expect(c90.thresholdY()).toBeCloseTo(160 - (90 / 100) * 160, 5);
  });

  it('clampa los porcentajes fuera de rango a 0..100 (límites del gráfico)', () => {
    const c = mount([
      snap(iso(2), 130, -15),
      snap(iso(1), 40, 20)
    ]);
    const pts = c.points();
    // RAM 130% queda clavada al techo (Y=0); CPU -15% no cae bajo el baseline (Y=160).
    expect(pts[0].ramY).toBe(0);
    expect(pts[0].cpuY).toBe(160);
    expect(pts[1].ramY).toBeCloseTo(160 - (40 / 100) * 160, 5);
  });

  it('el eje X es proporcional al span real de timestamps', () => {
    // Samples separados 60 minutos entre sí (último hace 0 min, primero hace 60).
    const c = mount([
      snap(iso(60)),
      snap(iso(0))
    ]);
    const labels = c.xAxisLabels();
    expect(labels[0]).toBe('-60 min');
    expect(labels[1]).toBe('-45 min');
    expect(labels[2]).toBe('-30 min');
    expect(labels[3]).toBe('-15 min');
    expect(labels[4]).toBe('Ahora');
  });

  it('el delta del tooltip sale del timestamp real, no de un índice', () => {
    const c = mount([snap(iso(5)), snap(iso(0))]);
    expect(c.minutesAgo(iso(5))).toBe(5);
    expect(c.minutesAgo(iso(0))).toBe(0);
    expect(c.minutesAgo('no-es-una-fecha')).toBe(0);
  });

  it('formatea el timestamp con el formateador canónico (HH:mm)', () => {
    const c = mount([snap(iso(2)), snap(iso(1))]);
    const formatted = c.formatTime(new Date().toISOString());
    expect(formatted).toMatch(/^\d{2}:\d{2}$/);
  });
});