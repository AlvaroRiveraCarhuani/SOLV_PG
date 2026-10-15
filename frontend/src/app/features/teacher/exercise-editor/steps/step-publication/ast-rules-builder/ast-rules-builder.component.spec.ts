import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ASTRulesBuilderComponent } from './ast-rules-builder.component';
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

  it('debe alternar la política pedagógica de ordenamiento nativo', () => {
    const sortPolicy = component.policies().find(p => p.key === 'sort');
    expect(sortPolicy).toBeDefined();

    sortPolicy?.toggle(true);
    expect(store.astRules().block_native_sort).toBe(true);

    sortPolicy?.toggle(false);
    expect(store.astRules().block_native_sort).toBe(false);
  });

  it('debe permitir agregar y eliminar reglas específicas de restricción', () => {
    component.newRule.set({
      language: 'python',
      type: 'function',
      pattern: 'pop',
      message: 'No usar pop'
    });
    component.addSpecificRule();

    let rules = component.customSpecificRules();
    expect(rules.length).toBe(1);
    expect(rules[0].pattern).toBe('pop');

    component.removeSpecificRule(rules[0]);
    rules = component.customSpecificRules();
    expect(rules.length).toBe(0);
  });
});
