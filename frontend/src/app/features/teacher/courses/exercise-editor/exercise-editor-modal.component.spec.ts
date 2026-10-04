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

  it('should initialize on step 1 without preselected modality and purpose by default', () => {
    expect(component.currentStep()).toBe(1);
    expect(component.labType()).toBeNull();
    expect(component.language()).toBe('python');
    expect(component.pedagogicalPurpose()).toBeNull();
    expect(component.allowBroadcast()).toBe(false);
  });

  it('should toggle pedagogical purpose to EXAM and disable broadcast', () => {
    component.setPedagogicalPurpose('EXAM');
    expect(component.pedagogicalPurpose()).toBe('EXAM');
    expect(component.allowBroadcast()).toBe(false);

    component.setPedagogicalPurpose('PRACTICE');
    expect(component.pedagogicalPurpose()).toBe('PRACTICE');
    expect(component.allowBroadcast()).toBe(true);
  });

  it('should require title, labType and purpose before advancing to step 2', () => {
    component.title.set('');
    component.goToStep(2);
    expect(component.currentStep()).toBe(1);
    expect(component.titleError()).toBe('El título del laboratorio es obligatorio para continuar.');
    expect(component.labTypeError()).toBe('Debe seleccionar la modalidad del laboratorio.');
    expect(component.purposeError()).toBe('Debe seleccionar el propósito pedagógico.');

    component.onTitleChange('Árboles Binarios');
    expect(component.titleError()).toBeNull();

    component.setLabType('ALGORITMO');
    expect(component.labTypeError()).toBeNull();

    component.setPedagogicalPurpose('PRACTICE');
    expect(component.purposeError()).toBeNull();
  });

  it('should toggle no due date correctly', () => {
    component.dueDate.set('2026-10-15T18:00');
    component.hasDueDate.set(true);

    component.toggleNoDueDate(true);
    expect(component.hasDueDate()).toBe(false);
    expect(component.dueDate()).toBe('');

    component.toggleNoDueDate(false);
    expect(component.hasDueDate()).toBe(true);
  });

  it('should navigate through 3 steps when title, labType and purpose are valid', () => {
    component.title.set('Árboles AVL');
    component.setLabType('ALGORITMO');
    component.setPedagogicalPurpose('PRACTICE');
    component.goToStep(2);
    expect(component.currentStep()).toBe(2);

    component.goToStep(3);
    expect(component.currentStep()).toBe(3);
  });

  it('should store pedagogical description in markdown', () => {
    component.description.set('### Instrucciones\nImplementar rotación simple.');
    expect(component.description()).toBe('### Instrucciones\nImplementar rotación simple.');
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
    component.setLabType('ALGORITMO');
    component.setPedagogicalPurpose('PRACTICE');
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
    component.setPedagogicalPurpose('PRACTICE');
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

  it('should compute hasDatabaseSatellite dynamically based on selected template', () => {
    component.setLabType('IDE_PERSISTENTE');
    component.templateId.set('tpl-python-ds');
    expect(component.hasDatabaseSatellite()).toBe(false);

    component.templateId.set('tpl-postgres-db');
    expect(component.hasDatabaseSatellite()).toBe(true);
  });

  it('should initialize with locked IDE modality when initialTemplate is passed', () => {
    const testFixture = TestBed.createComponent(ExerciseEditorModalComponent);
    const testComp = testFixture.componentInstance;
    testFixture.componentRef.setInput('subjectId', 'sub-001');
    testFixture.componentRef.setInput('initialTemplate', {
      id: 'tpl-custom-ide',
      name: 'Entorno C# .NET',
      environment_type: 'IDE_PERSISTENTE'
    });
    testFixture.detectChanges();

    expect(testComp.labType()).toBe('IDE_PERSISTENTE');
    expect(testComp.templateId()).toBe('tpl-custom-ide');
    expect(testComp.title()).toBe('Laboratorio: Entorno C# .NET');
  });
});
