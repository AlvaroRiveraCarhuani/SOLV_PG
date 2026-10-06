import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StudentPreviewModalComponent } from './student-preview-modal.component';
import { ExerciseEditorStore } from '../exercise-editor.store';

describe('StudentPreviewModalComponent', () => {
  let component: StudentPreviewModalComponent;
  let fixture: ComponentFixture<StudentPreviewModalComponent>;
  let store: ExerciseEditorStore;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StudentPreviewModalComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentPreviewModalComponent);
    component = fixture.componentInstance;
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('debe filtrar solo los casos con visibilidad example', () => {
    store.setCases([
      { input: '1', expected_output: '1', visibility: 'example', weight: 1.0 },
      { input: '2', expected_output: '2', visibility: 'public', weight: 1.0 },
      { input: '3', expected_output: '3', visibility: 'hidden', weight: 1.0 },
      { input: '4', expected_output: '4', visibility: 'example', weight: 1.0 }
    ]);

    const examples = component.exampleCases();
    expect(examples.length).toBe(2);
    expect(examples[0].input).toBe('1');
    expect(examples[1].input).toBe('4');
  });

  it('debe actualizar el boilerplate al cambiar de lenguaje', () => {
    store.metadata.set({
      title: 'Prueba',
      difficulty: 'easy',
      tags: [],
      modality: 'judge',
      purpose: 'class',
      per_student_seed: false,
      language: 'python',
      allowed_languages: ['python', 'cpp', 'java'],
      time_limit_ms: 1000,
      memory_limit_mb: 128,
      due_date: ''
    });

    store.setBoilerplateForLanguage('python', 'def main(): pass');
    store.setBoilerplateForLanguage('cpp', 'int main() { return 0; }');

    component.onLanguageChange('python');
    expect(component.currentBoilerplate()).toBe('def main(): pass');

    component.onLanguageChange('cpp');
    expect(component.currentBoilerplate()).toBe('int main() { return 0; }');
  });

  it('debe emitir evento de cierre al hacer click en cerrar', () => {
    let closed = false;
    component.close.subscribe(() => {
      closed = true;
    });

    component.onClose();
    expect(closed).toBe(true);
  });
});
