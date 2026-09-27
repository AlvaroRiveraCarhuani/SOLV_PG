import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { PublishDialogComponent } from './publish-dialog.component';
import { EnvTestJob } from '../../../services/env-test-job.service';

const makeJob = (overrides: Partial<EnvTestJob> = {}): EnvTestJob => ({
  id: 'job-1',
  status: 'success',
  error_message: undefined,
  digest_unverified: false,
  result: { tools: ['python', 'pip'], services: [] },
  ...overrides
} as any);

describe('PublishDialogComponent', () => {
  let component: PublishDialogComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PublishDialogComponent]
    }).compileComponents();

    const fixture = TestBed.createComponent(PublishDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('imagen vacía genera check de error bloqueante', () => {
    // dockerImage default ''
    const checks = component.checks();
    const imgCheck = checks.find(c => c.id === 'img-empty');
    expect(imgCheck?.type).toBe('error');
    expect(component.hasBlockingErrors()).toBe(true);
  });

  it('imagen con tag :latest genera error bloqueante', () => {
    TestBed.runInInjectionContext(() => {});
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:latest');
    fixture.detectChanges();
    const latestCheck = c.checks().find(ch => ch.id === 'img-latest');
    expect(latestCheck?.type).toBe('error');
    expect(c.hasBlockingErrors()).toBe(true);
  });

  it('imagen válida sin tag latest genera check pass', () => {
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim');
    fixture.detectChanges();
    const imgCheck = c.checks().find(ch => ch.id === 'img-ok');
    expect(imgCheck?.type).toBe('pass');
  });

  it('sin envTestJob genera advertencia de entorno no probado', () => {
    // envTestJob default null
    const checks = component.checks();
    const envCheck = checks.find(c => c.id === 'env-not-tested');
    expect(envCheck?.type).toBe('warning');
  });

  it('envTestJob success sin digest_unverified genera check pass', () => {
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim');
    fixture.componentRef.setInput('envTestJob', makeJob());
    fixture.detectChanges();
    const envCheck = c.checks().find(ch => ch.id === 'env-verified');
    expect(envCheck?.type).toBe('pass');
  });

  it('envTestJob success con digest_unverified genera advertencia', () => {
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim');
    fixture.componentRef.setInput('envTestJob', makeJob({ digest_unverified: true }));
    fixture.detectChanges();
    const envCheck = c.checks().find(ch => ch.id === 'env-digest-unverified');
    expect(envCheck?.type).toBe('warning');
  });

  it('envTestJob failed genera error bloqueante', () => {
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim');
    fixture.componentRef.setInput('envTestJob', makeJob({ status: 'failed', error_message: 'Python not found' }));
    fixture.detectChanges();
    const envCheck = c.checks().find(ch => ch.id === 'env-failed');
    expect(envCheck?.type).toBe('error');
    expect(c.hasBlockingErrors()).toBe(true);
  });

  it('RAM mayor a 2048 MB genera advertencia de memoria alta', () => {
    const fixture = TestBed.createComponent(PublishDialogComponent);
    const c = fixture.componentInstance;
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim');
    fixture.componentRef.setInput('baseRamMB', 4096);
    fixture.componentRef.setInput('envTestJob', makeJob());
    fixture.detectChanges();
    const ramCheck = c.checks().find(ch => ch.id === 'ram-high');
    expect(ramCheck?.type).toBe('warning');
  });

  it('onConfirm no emite cuando hay errores bloqueantes', () => {
    let emitted = false;
    component.confirmed.subscribe(() => { emitted = true; });
    // default: imagen vacía = error bloqueante
    component.onConfirm();
    expect(emitted).toBe(false);
  });

  it('onClose emite evento closed', () => {
    let closed = false;
    component.closed.subscribe(() => { closed = true; });
    component.onClose();
    expect(closed).toBe(true);
  });

  it('toggleScriptModal invierte el estado del visor de script', () => {
    expect(component.showScriptModal()).toBe(false);
    component.toggleScriptModal();
    expect(component.showScriptModal()).toBe(true);
    component.toggleScriptModal();
    expect(component.showScriptModal()).toBe(false);
  });
});
