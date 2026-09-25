import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Component } from '@angular/core';
import { signal } from '@angular/core';
import { KpiCardComponent, KpiGridComponent, KpiVariant } from './kpi-card.component';

@Component({
  standalone: true,
  imports: [KpiCardComponent, KpiGridComponent],
  template: `
    <kpi-grid [columns]="columns()">
      <kpi-card
        [label]="label()"
        [value]="value()"
        [detail]="detail()"
        [totalBadge]="totalBadge()"
        [variant]="variant()"
      >
        <span kpi-icon class="test-icon">icon</span>
      </kpi-card>
    </kpi-grid>
  `
})
class HostComponent {
  readonly label = signal('Cursando en Periodo');
  readonly value = signal<string | number>(42);
  readonly detail = signal<string | undefined>('Estudiantes con matrícula activa');
  readonly totalBadge = signal<string | undefined>('de 100 total');
  readonly variant = signal<KpiVariant>('primary');
  readonly columns = signal(4);
}

describe('KpiCardComponent and KpiGridComponent Unit Tests', () => {
  let fixture: ComponentFixture<HostComponent>;
  let host: HostComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(HostComponent);
    host = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse y renderizar label y value correctamente', () => {
    const labelEl = fixture.nativeElement.querySelector('.kpi-label');
    const valueEl = fixture.nativeElement.querySelector('.kpi-value');

    expect(labelEl.textContent).toContain('Cursando en Periodo');
    expect(valueEl.textContent).toContain('42');
  });

  it('debe renderizar totalBadge si se proporciona', () => {
    const badgeEl = fixture.nativeElement.querySelector('.kpi-total-badge');
    expect(badgeEl).toBeTruthy();
    expect(badgeEl.textContent).toContain('de 100 total');
  });

  it('debe renderizar detail si se proporciona', () => {
    const detailEl = fixture.nativeElement.querySelector('.kpi-detail');
    expect(detailEl).toBeTruthy();
    expect(detailEl.textContent).toContain('Estudiantes con matrícula activa');
  });

  it('debe resolver y asignar las variantes de estilo semánticas', () => {
    const iconWrapper = fixture.nativeElement.querySelector('.kpi-icon-wrapper');
    expect(iconWrapper.getAttribute('data-variant')).toBe('primary');

    host.variant.set('running');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('running');

    host.variant.set('green');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('running');

    host.variant.set('warning');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('warning');

    host.variant.set('amber');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('warning');

    host.variant.set('danger');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('danger');

    host.variant.set('red');
    fixture.detectChanges();
    expect(iconWrapper.getAttribute('data-variant')).toBe('danger');
  });

  it('debe proyectar el ícono dentro del wrapper', () => {
    const projectedIcon = fixture.nativeElement.querySelector('.kpi-icon-wrapper .test-icon');
    expect(projectedIcon).toBeTruthy();
    expect(projectedIcon.textContent).toBe('icon');
  });

  it('KpiGridComponent debe configurar la variable de columnas CSS', () => {
    const gridEl = fixture.nativeElement.querySelector('.kpi-grid');
    expect(gridEl.style.getPropertyValue('--kpi-grid-cols')).toBe('4');

    host.columns.set(3);
    fixture.detectChanges();
    expect(gridEl.style.getPropertyValue('--kpi-grid-cols')).toBe('3');
  });
});
