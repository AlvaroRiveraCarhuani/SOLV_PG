import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvStepExecutionComponent } from './step-execution.component';

describe('SolvStepExecutionComponent', () => {
  let component: SolvStepExecutionComponent;
  let fixture: ComponentFixture<SolvStepExecutionComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvStepExecutionComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepExecutionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.detectChanges();
  });

  it('debe crearse correctamente y mostrar textarea de setupScript para IDE_PERSISTENTE', () => {
    expect(component).toBeTruthy();
    const textarea = fixture.nativeElement.querySelector('textarea');
    expect(textarea).toBeTruthy();
    expect(fixture.nativeElement.querySelector('input[type="number"]')).toBeNull();
  });

  it('debe emitir setupScriptChange cuando se modifica el script de inicialización', () => {
    let emitted = '';
    component.setupScriptChange.subscribe(v => emitted = v);

    const textarea = fixture.nativeElement.querySelector('textarea');
    textarea.value = '#!/bin/bash\necho "init"';
    textarea.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(emitted).toBe('#!/bin/bash\necho "init"');
  });

  it('debe mostrar campos de entrypoint, timeout y stdin para JUEZ_EFIMERO', () => {
    fixture.componentRef.setInput('targetEnvironment', 'JUEZ_EFIMERO');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('textarea')).toBeNull();
    const inputs = fixture.nativeElement.querySelectorAll('input');
    expect(inputs.length).toBe(3); // entrypoint, timeoutMS, sampleInput
  });

  it('debe emitir entrypointChange al modificar el comando de juez', () => {
    fixture.componentRef.setInput('targetEnvironment', 'JUEZ_EFIMERO');
    fixture.detectChanges();

    let emitted = '';
    component.entrypointChange.subscribe(v => emitted = v);

    const entrypointInput = fixture.nativeElement.querySelector('input[type="text"]');
    entrypointInput.value = 'gcc solution.c -o solution';
    entrypointInput.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(emitted).toBe('gcc solution.c -o solution');
  });

  it('debe emitir helpRequested al hacer clic en el botón de ayuda', () => {
    let emitted = false;
    component.helpRequested.subscribe(() => emitted = true);

    const helpBtn = fixture.nativeElement.querySelector('.btn-step-help');
    expect(helpBtn).toBeTruthy();
    helpBtn.click();

    expect(emitted).toBe(true);
  });

  it('debe alternar showContractPopover al pulsar el botón de contrato', () => {
    expect(component.showContractPopover()).toBe(false);
    component.toggleContractPopover();
    expect(component.showContractPopover()).toBe(true);
    component.closeContractPopover();
    expect(component.showContractPopover()).toBe(false);
  });

  it('debe emitir setupScript al seleccionar un ejemplo de script para IDE', () => {
    let emitted = '';
    component.setupScriptChange.subscribe(v => emitted = v);

    component.applyScriptExample('export APP_ENV=development');
    expect(emitted).toBe('export APP_ENV=development');
  });

  it('debe emitir entrypoint al seleccionar un ejemplo de compilación para Juez', () => {
    let emitted = '';
    component.entrypointChange.subscribe(v => emitted = v);

    component.applyEntrypointExample('python3 solution.py');
    expect(emitted).toBe('python3 solution.py');
  });
});
