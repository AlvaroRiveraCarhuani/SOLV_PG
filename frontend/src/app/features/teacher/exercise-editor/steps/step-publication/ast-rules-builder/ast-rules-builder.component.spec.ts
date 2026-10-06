import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ASTRulesBuilderComponent, AST_PRESETS } from './ast-rules-builder.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

describe('ASTRulesBuilderComponent', () => {
  let component: ASTRulesBuilderComponent;
  let fixture: ComponentFixture<ASTRulesBuilderComponent>;
  let store: ExerciseEditorStore;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ASTRulesBuilderComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ASTRulesBuilderComponent);
    component = fixture.componentInstance;
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe inicializarse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('debe aplicar un preset rápido de reglas AST', () => {
    const preset = AST_PRESETS[0];
    component.applyPreset(preset);

    const rules = store.astRules().custom_rules || [];
    expect(rules.length).toBe(1);
    expect(rules[0].pattern).toBe(preset.rule.pattern);
  });

  it('debe permitir agregar y eliminar reglas personalizadas', () => {
    component.newRule.set({
      language: 'python',
      type: 'method',
      pattern: 'pop',
      message: 'No usar pop'
    });
    component.addRule();

    let rules = store.astRules().custom_rules || [];
    expect(rules.length).toBe(1);
    expect(rules[0].pattern).toBe('pop');

    component.removeRule(0);
    rules = store.astRules().custom_rules || [];
    expect(rules.length).toBe(0);
  });
});
