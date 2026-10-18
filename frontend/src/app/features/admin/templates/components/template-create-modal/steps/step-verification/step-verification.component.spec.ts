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
      error_code: 'tool_missing'
    } as any);
    fixture.detectChanges();

    let jumpedTo: string = '';
    component.jumpToSection.subscribe((step: any) => jumpedTo = step);

    const jumpButton = fixture.nativeElement.querySelector('.btn-step-jump-link');
    expect(jumpButton).toBeTruthy();
    jumpButton.click();
    expect(jumpedTo).toBe('image');

    // Ahora ante fallo de inicio/crash, debe apuntar a execution (Paso 4)
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-crash',
      status: 'failed',
      error_code: 'test_crash'
    } as any);
    fixture.detectChanges();

    const jumpButtonExecution = fixture.nativeElement.querySelector('.btn-step-jump-link');
    expect(jumpButtonExecution).toBeTruthy();
    jumpButtonExecution.click();
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

  it('debe reflejar estado de carga y deshabilitar el botón principal cuando isTesting es true, sin botón circular duplicado', () => {
    component.onButtonStateChange('running');
    fixture.detectChanges();

    expect(component.isTestRunning()).toBe(true);

    const primaryBtn = fixture.nativeElement.querySelector('.btn-integrity-primary');
    expect(primaryBtn.disabled).toBe(true);
    expect(primaryBtn.textContent).toContain('Probando...');

    const iconBtn = fixture.nativeElement.querySelector('.btn-integrity-icon');
    expect(iconBtn).toBeNull();

    component.onButtonStateChange('idle');
    fixture.detectChanges();
    expect(component.isTestRunning()).toBe(false);
    expect(primaryBtn.disabled).toBe(false);
    expect(primaryBtn.textContent).toContain('Probar entorno');
  });

  it('debe cambiar el texto a "Reintentar prueba" cuando la prueba anterior falló', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-fail',
      status: 'failed',
      error_code: 'registry_unreachable'
    } as any);
    fixture.detectChanges();

    expect(component.hasFailedTest()).toBe(true);
    const primaryBtn = fixture.nativeElement.querySelector('.btn-integrity-primary');
    expect(primaryBtn.textContent).toContain('Reintentar prueba');
  });

  it('debe emitir jobProgress cuando el runner secundario reporta avance', () => {
    let emittedJob: EnvTestJob | null = null;
    component.jobProgress.subscribe((job) => emittedJob = job);

    component.onJobProgress(mockJob);
    expect(emittedJob).toEqual(mockJob);
  });

  it('debe no tener badge de ID ficticio en la tarjeta de resumen técnico', () => {
    const idBadge = fixture.nativeElement.querySelector('.tech-summary-id');
    expect(idBadge).toBeNull();
  });

  it('debe habilitar bypass y emitir requestBypass cuando la imagen es local y hay desconexión OCI', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-err',
      status: 'failed',
      error_code: 'registry_unreachable',
      is_local_image: true
    } as any);
    fixture.detectChanges();

    expect(component.canBypass()).toBe(true);
    expect(component.diagnosticAction().jumpSection).toBe('image');
    expect(component.diagnosticAction().jumpLabel).toBe('Paso 3');

    let bypassEmitted = false;
    component.requestBypass.subscribe(() => bypassEmitted = true);

    const bypassBtn = fixture.nativeElement.querySelector('.btn-bypass-link');
    expect(bypassBtn).toBeTruthy();
    bypassBtn.click();
    expect(bypassEmitted).toBe(true);
  });

  it('debe PROHIBIR bypass cuando el error es not_found_remote (Error 404)', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-404',
      status: 'failed',
      error_code: 'not_found_remote',
      is_local_image: false
    } as any);
    fixture.detectChanges();

    expect(component.canBypass()).toBe(false);
    expect(fixture.nativeElement.querySelector('.btn-bypass-link')).toBeNull();
    expect(component.diagnosticAction().text).toContain('Error 404');
    expect(component.diagnosticAction().jumpSection).toBe('image');
  });

  it('debe PROHIBIR bypass cuando la imagen no reside físicamente en el disco local', () => {
    fixture.componentRef.setInput('activeEnvTestJob', {
      id: 'job-net-fail',
      status: 'failed',
      error_code: 'network_failure',
      is_local_image: false
    } as any);
    fixture.detectChanges();

    expect(component.canBypass()).toBe(false);
    expect(fixture.nativeElement.querySelector('.btn-bypass-link')).toBeNull();
  });
});
