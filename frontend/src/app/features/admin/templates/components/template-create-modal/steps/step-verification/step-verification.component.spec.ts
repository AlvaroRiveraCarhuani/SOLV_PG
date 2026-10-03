import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StepVerificationComponent } from './step-verification.component';
import { EnvTestJob } from '../../../../../services/env-test-job.service';

describe('StepVerificationComponent', () => {
  let component: StepVerificationComponent;
  let fixture: ComponentFixture<StepVerificationComponent>;

  const mockJob: EnvTestJob = {
    id: 'job-123',
    image: 'python:3.12-slim-bookworm',
    tools: ['python3', 'pip'],
    status: 'success',
    progress: {
      bytes_done: 100,
      bytes_total: 100,
      layer_current: 1,
      layers_total: 1,
      percent: 100
    },
    result: {
      tools: [
        { name: 'python3', present: true, version: 'Python 3.12.2' },
        { name: 'pip', present: true, version: 'pip 24.0' }
      ],
      exit_code: 0,
      duration_ms: 1250
    },
    digest_unverified: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StepVerificationComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StepVerificationComponent);
    component = fixture.componentInstance;
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: () => Promise.resolve() },
      configurable: true
    });

    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.componentRef.setInput('toolsList', ['python3', 'pip']);
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.componentRef.setInput('baseRamMB', 1024);
    fixture.detectChanges();
  });

  it('debe crearse correctamente con la tarjeta de integridad y resumen técnico', () => {
    expect(component).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.integrity-card')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.technical-summary-card')).toBeTruthy();
  });

  it('debe mostrar la alerta de prueba obsoleta si isEnvTestStale es true', () => {
    fixture.componentRef.setInput('isEnvTestStale', true);
    fixture.detectChanges();

    const staleAlert = fixture.nativeElement.querySelector('.stale-test-alert');
    expect(staleAlert).toBeTruthy();
    expect(staleAlert.textContent).toContain('Prueba obsoleta');

    let retryClicked = false;
    component.retryTest.subscribe(() => retryClicked = true);

    const retryBtn = staleAlert.querySelector('.btn-retry-stale');
    retryBtn.click();
    expect(retryClicked).toBe(true);
  });

  it('debe abrir modal de terminal de auditoría al presionar Ver logs de auditoría', () => {
    fixture.componentRef.setInput('activeEnvTestJob', mockJob);
    fixture.detectChanges();

    const toggleBtn = fixture.nativeElement.querySelector('.btn-integrity-secondary');
    expect(toggleBtn).toBeTruthy();

    toggleBtn.click();
    fixture.detectChanges();

    expect(component.isFullLogsModalOpen()).toBe(true);
    const modal = fixture.nativeElement.querySelector('.audit-logs-modal-dialog');
    expect(modal).toBeTruthy();
    expect(modal.textContent).toContain('Registro de ejecución');
    expect(modal.textContent).toContain('APROBADO');
  });

  it('debe cerrar el modal de logs al pulsar Escape sin cerrar el asistente padre', () => {
    component.openFullLogsModal();
    expect(component.isFullLogsModalOpen()).toBe(true);

    component.onEscape();
    expect(component.isFullLogsModalOpen()).toBe(false);
  });

  it('debe filtrar líneas del log en tiempo real según el término de búsqueda', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-err',
      image: 'custom:v1',
      tools: [],
      status: 'failed',
      error_message: 'Fallo durante la inicialización'
    } as any);
    fixture.detectChanges();

    component.openFullLogsModal();
    fixture.detectChanges();

    const totalBefore = component.filteredParsedLogLines().length;
    expect(totalBefore).toBeGreaterThan(0);

    component.searchQuery.set('error');
    fixture.detectChanges();

    const filtered = component.filteredParsedLogLines();
    expect(filtered.length).toBeGreaterThan(0);
    expect(filtered.some(l => l.message.toLowerCase().includes('error') || l.level === 'ERROR')).toBe(true);
  });

  it('debe emitir jumpToSection al hacer clic en los enlaces de salto directo del diagnóstico', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-err',
      image: 'python:3.12-slim-bookworm',
      tools: ['pytest'],
      status: 'failed',
      error_message: 'MISSING:pytest'
    } as any);
    fixture.detectChanges();

    let jumpedTo: string = '';
    component.jumpToSection.subscribe((step: any) => jumpedTo = step);

    const jumpButtons = fixture.nativeElement.querySelectorAll('.btn-step-jump-link');
    expect(jumpButtons.length).toBe(2);

    // Clic en Paso 3
    jumpButtons[0].click();
    expect(jumpedTo).toBe('image');

    // Clic en Paso 4
    jumpButtons[1].click();
    expect(jumpedTo).toBe('execution');
  });

  it('debe copiar el tag de la imagen con feedback visual', async () => {
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.detectChanges();
    component.copyImageTag();
    await Promise.resolve();
    await Promise.resolve();
    expect(component.copyImageSuccess()).toBe(true);
  });

  it('debe emitir helpRequested al presionar el botón de ayuda contextual', () => {
    let emitted = false;
    component.helpRequested.subscribe(() => emitted = true);

    const helpBtn = fixture.nativeElement.querySelector('.btn-step-help');
    expect(helpBtn).toBeTruthy();
    helpBtn.click();

    expect(emitted).toBe(true);
  });
});
