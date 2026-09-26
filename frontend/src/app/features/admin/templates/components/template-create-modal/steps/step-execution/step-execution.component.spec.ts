import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StepExecutionComponent } from './step-execution.component';

describe('StepExecutionComponent', () => {
  let component: StepExecutionComponent;
  let fixture: ComponentFixture<StepExecutionComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StepExecutionComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(StepExecutionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.detectChanges();
  });

  it('debe crearse correctamente y mostrar textarea de setupScript para IDE_PERSISTENTE sin title nativos', () => {
    expect(component).toBeTruthy();
    const textarea = fixture.nativeElement.querySelector('textarea');
    expect(textarea).toBeTruthy();
    expect(textarea.getAttribute('spellcheck')).toBe('false');
    expect(fixture.nativeElement.querySelector('input[type="number"]')).toBeNull();

    // Sin atributos title nativos en botones o inputs
    const elementsWithTitle = fixture.nativeElement.querySelectorAll('[title]');
    expect(elementsWithTitle.length).toBe(0);
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

  it('debe insertar snippet con marcador formateado al hacer clic en chip de IDE', () => {
    let emitted = '';
    component.setupScriptChange.subscribe(v => emitted = v);

    const example = component.ideExamples[0];
    component.applyScriptExample(example);
    expect(emitted).toContain(`# --- ${example.label} ---`);
    expect(emitted).toContain(example.code);
  });

  it('no debe duplicar un snippet ya insertado y debe activar toastWarning', () => {
    const example = component.ideExamples[0];
    fixture.componentRef.setInput('setupScript', `# --- ${example.label} ---\n${example.code}`);
    fixture.detectChanges();

    let emitted = '';
    component.setupScriptChange.subscribe(v => emitted = v);

    component.applyScriptExample(example);
    expect(emitted).toBe('');
    expect(component.toastWarning()).toBe('La plantilla ya está insertada en el editor');
  });

  it('debe gestionar el estado de hoveredIdeExample y hoveredJudgeExample para previsualización', () => {
    expect(component.hoveredIdeExample()).toBeNull();
    component.showIdePreview(component.ideExamples[0]);
    expect(component.hoveredIdeExample()).toEqual(component.ideExamples[0]);
    component.hideIdePreview();
    expect(component.hoveredIdeExample()).toBeNull();

    expect(component.hoveredJudgeExample()).toBeNull();
    component.showJudgePreview(component.judgeExamples[0]);
    expect(component.hoveredJudgeExample()).toEqual(component.judgeExamples[0]);
    component.hideJudgePreview();
    expect(component.hoveredJudgeExample()).toBeNull();
  });

  it('debe emitir entrypoint al seleccionar un ejemplo de compilación para Juez', () => {
    let emitted = '';
    component.entrypointChange.subscribe(v => emitted = v);

    component.applyEntrypointExample('python3 solution.py');
    expect(emitted).toBe('python3 solution.py');
  });
});
