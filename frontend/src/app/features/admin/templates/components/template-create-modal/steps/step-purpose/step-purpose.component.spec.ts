import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvStepPurposeComponent } from './step-purpose.component';
import { TargetEnvironment } from '../../../../../services/admin-templates.service';

describe('SolvStepPurposeComponent', () => {
  let component: SolvStepPurposeComponent;
  let fixture: ComponentFixture<SolvStepPurposeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvStepPurposeComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepPurposeComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe iniciar con IDE_PERSISTENTE por defecto', () => {
    expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');
    const ideCard = fixture.nativeElement.querySelector('#purpose-card-ide');
    expect(ideCard.classList.contains('selected')).toBe(true);
    expect(ideCard.getAttribute('aria-checked')).toBe('true');
  });

  it('debe emitir targetEnvironmentChange al hacer clic en Juez Virtual', () => {
    let selected: TargetEnvironment | null = null;
    component.targetEnvironmentChange.subscribe((env) => (selected = env));

    const judgeCard = fixture.nativeElement.querySelector('#purpose-card-judge');
    judgeCard.click();

    expect(selected).toBe('JUEZ_EFIMERO');
  });

  it('debe alternar selección con flechas de teclado y avanzar con Enter', () => {
    let selected: TargetEnvironment | null = null;
    let advanced = false;
    component.targetEnvironmentChange.subscribe((env) => (selected = env));
    component.advance.subscribe(() => (advanced = true));

    const ideCard = fixture.nativeElement.querySelector('#purpose-card-ide');
    ideCard.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
    expect(selected).toBe('JUEZ_EFIMERO');

    ideCard.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    expect(advanced).toBe(true);
  });
});
