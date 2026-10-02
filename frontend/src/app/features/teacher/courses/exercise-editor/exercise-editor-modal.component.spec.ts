import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ExerciseEditorModalComponent } from './exercise-editor-modal.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { of } from 'rxjs';

describe('ExerciseEditorModalComponent Wizard & Governance', () => {
  let component: ExerciseEditorModalComponent;
  let fixture: ComponentFixture<ExerciseEditorModalComponent>;
  let mockCourseService: Partial<TeacherCourseService>;

  beforeEach(async () => {
    mockCourseService = {
      createExercise: vi.fn().mockReturnValue(of({ id: 'ex-123' })),
      updateExercise: vi.fn().mockReturnValue(of(undefined)),
      publishExercise: vi.fn().mockReturnValue(of(undefined)),
      bulkUploadTestCases: vi.fn().mockReturnValue(of({ imported_count: 2 }))
    };

    await TestBed.configureTestingModule({
      imports: [
        ExerciseEditorModalComponent,
        HttpClientTestingModule
      ],
      providers: [
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ExerciseEditorModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('subjectId', 'sub-001');
    fixture.componentRef.setInput('subjectName', 'Programación II');
    fixture.detectChanges();
  });

  it('should initialize on step 1 with default ALGORITMO modality', () => {
    expect(component.currentStep()).toBe(1);
    expect(component.labType()).toBe('ALGORITMO');
    expect(component.language()).toBe('python');
    expect(component.allowBroadcast()).toBe(true);
  });

  it('should require title before advancing to step 2', () => {
    component.title.set('');
    component.goToStep(2);
    expect(component.currentStep()).toBe(1);
    expect(component.formError()).toBe('El título del laboratorio es obligatorio para continuar.');
  });

  it('should navigate through 3 steps when title is valid', () => {
    component.title.set('Árboles AVL');
    component.goToStep(2);
    expect(component.currentStep()).toBe(2);

    component.goToStep(3);
    expect(component.currentStep()).toBe(3);
  });

  it('should render markdown preview for pedagogical description', () => {
    component.description.set('### Instrucciones\nImplementar rotación simple.');
    const preview = component.renderedDescription();
    expect(preview).toContain('<h3>Instrucciones</h3>');
    expect(preview).toContain('Implementar rotación simple.');
  });

  it('should open template request modal (ADR-030) from workspace step', () => {
    component.setLabType('IDE_PERSISTENTE');
    expect(component.showTemplateRequestModal()).toBe(false);

    component.openTemplateRequestModal();
    expect(component.showTemplateRequestModal()).toBe(true);

    component.closeTemplateRequestModal();
    expect(component.showTemplateRequestModal()).toBe(false);
  });

  it('should submit algorithm lab with test cases and publish', () => {
    component.title.set('Grafos Dijkstra');
    component.testCases.set([
      { input: '4 4', expected_output: '10', is_hidden: false }
    ]);

    component.submit(true);

    expect(mockCourseService.createExercise).toHaveBeenCalledWith(expect.objectContaining({
      subject_id: 'sub-001',
      title: 'Grafos Dijkstra',
      type: 'algorithm',
      language: 'python'
    }));
    expect(mockCourseService.publishExercise).toHaveBeenCalledWith('ex-123');
  });

  it('should submit persistent workspace lab using approved template RAM', () => {
    component.setLabType('IDE_PERSISTENTE');
    component.title.set('Proyecto Data Science con Pandas');
    component.templateId.set('tpl-python-ds');

    component.submit(false);

    expect(mockCourseService.createExercise).toHaveBeenCalledWith(expect.objectContaining({
      subject_id: 'sub-001',
      title: 'Proyecto Data Science con Pandas',
      type: 'workspace',
      template_id: 'tpl-python-ds',
      memory_limit_mb: 1024
    }));
  });
});
