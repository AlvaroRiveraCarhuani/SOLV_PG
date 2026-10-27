import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed, ComponentFixture } from '@angular/core/testing';
import { HardwareKpiComponent } from './hardware-kpi.component';
import { HostHardwareMetrics } from '@core/models/admin.model';

function makeMetrics(overrides: Partial<HostHardwareMetrics> = {}): HostHardwareMetrics {
  return {
    ram_used_bytes: 8 * 1024 * 1024 * 1024,
    ram_total_bytes: 16 * 1024 * 1024 * 1024,
    ram_percent: 50,
    cpu_cores: 8,
    cpu_percent: 12,
    disk_used_bytes: 184 * 1024 * 1024 * 1024,
    disk_total_bytes: 512 * 1024 * 1024 * 1024,
    disk_percent: 36,
    disk_path: '/var/lib/docker',
    containers_active: 3,
    containers_hibernated: 5,
    containers_max: 40,
    estimated_max_containers: 8,
    capacity_basis: 'mem_available',
    max_containers_policy: 40,
    ram_per_lab_mb: 512,
    ...overrides
  };
}

describe('HardwareKpiComponent (Signals)', () => {
  let component: HardwareKpiComponent;
  let fixture: ComponentFixture<HardwareKpiComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HardwareKpiComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(HardwareKpiComponent);
    fixture.componentRef.setInput('metrics', makeMetrics());
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('se crea y calcula RAM/CPU/Disco desde el input con Signals', () => {
    expect(component).toBeTruthy();
    expect(component.ramUsedGB()).toBe('8.0');
    expect(component.ramTotalGB()).toBe('16.0');
    expect(component.ramPercentComputed()).toBe(50);
    expect(component.diskUsedGB()).toBe('184');
    expect(component.diskTotalGB()).toBe('512');
    expect(component.concurrencyMax()).toBe(8);
  });

  it('muestra la ruta de disco medida por el backend en el footer', () => {
    const html = fixture.nativeElement.outerHTML || '';
    expect(html).toContain('/var/lib/docker');
  });

  it('la concurrencia usa running / estimated_max_containers, no containers_max', () => {
    expect(component.concurrencyPercent()).toBe(Math.round((3 / 8) * 100));
    expect(component.concurrencyMax()).not.toBe(component.metrics().containers_max);
  });

  it('explica el capacity_basis mem_available en el subtítulo', () => {
    expect(component.capacityBasisLabel()).toBe('Limitado por RAM disponible del host');
  });

  it('explica el capacity_basis policy', () => {
    TestBed.resetTestingModule();
    const fixture = TestBed.configureTestingModule({
      imports: [HardwareKpiComponent]
    }).createComponent(HardwareKpiComponent);
    fixture.componentRef.setInput('metrics', makeMetrics({ capacity_basis: 'policy' }));
    fixture.detectChanges();
    expect(fixture.componentInstance.capacityBasisLabel()).toBe('Techo de política (max_containers)');
  });

  it('el footer de hibernados usa el conteo real', () => {
    expect(component.metrics().containers_hibernated).toBe(5);
  });

  it('sin estimated_max_containers cae al techo de política', () => {
    TestBed.resetTestingModule();
    const fixture = TestBed.configureTestingModule({
      imports: [HardwareKpiComponent]
    }).createComponent(HardwareKpiComponent);
    fixture.componentRef.setInput('metrics', makeMetrics({ estimated_max_containers: undefined, containers_max: 40 }));
    fixture.detectChanges();
    expect(fixture.componentInstance.concurrencyMax()).toBe(40);
  });
});