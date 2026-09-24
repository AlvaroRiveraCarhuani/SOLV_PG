import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { SolvStepVerificationComponent } from './step-verification.component';
import { EnvTestJob } from '../../../../../services/env-test-job.service';

describe('SolvStepVerificationComponent', () => {
  let component: SolvStepVerificationComponent;
  let fixture: ComponentFixture<SolvStepVerificationComponent>;

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
      imports: [SolvStepVerificationComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepVerificationComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.componentRef.setInput('toolsList', ['python3', 'pip']);
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.componentRef.setInput('baseRamMB', 1024);
    fixture.detectChanges();
  });

  it('debe crearse correctamente con el botón de prueba y resumen técnico', () => {
    expect(component).toBeTruthy();
    expect(fixture.nativeElement.querySelector('solv-env-test-button')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.verification-summary-card')).toBeTruthy();
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

  it('debe permitir expandir los logs de ejecución y mostrar el output formateado', () => {
    fixture.componentRef.setInput('activeEnvTestJob', mockJob);
    fixture.detectChanges();

    const toggleBtn = fixture.nativeElement.querySelector('.btn-toggle-logs');
    expect(toggleBtn).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.logs-terminal')).toBeNull();

    toggleBtn.click();
    fixture.detectChanges();

    const terminal = fixture.nativeElement.querySelector('.logs-terminal');
    expect(terminal).toBeTruthy();
    expect(terminal.textContent).toContain('SOLV SMOKE TEST RUNNER');
    expect(terminal.textContent).toContain('Python 3.12.2');
  });

  it('debe surfacear advertencias si la configuración es subóptima', () => {
    // IDE con RAM < 512 MB y sin herramientas
    fixture.componentRef.setInput('baseRamMB', 256);
    fixture.componentRef.setInput('toolsList', []);
    fixture.detectChanges();

    expect(component.suboptimalWarnings().length).toBeGreaterThan(0);
    const warningsBox = fixture.nativeElement.querySelector('.suboptimal-warnings-box');
    expect(warningsBox).toBeTruthy();
    expect(warningsBox.textContent).toContain('256 MB');
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
