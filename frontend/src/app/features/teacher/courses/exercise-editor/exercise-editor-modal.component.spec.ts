import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ExerciseEditorModalComponent } from './exercise-editor-modal.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { of } from 'rxjs';

describe('ExerciseEditorModalComponent', () => {
  let component: ExerciseEditorModalComponent;
  let fixture: ComponentFixture<ExerciseEditorModalComponent>;
  let mockCourseService: Partial<TeacherCourseService>;

  beforeEach(async () => {
    mockCourseService = {
      createExercise: vi.fn().mockReturnValue(of({ id: 'ex-123' })),
      updateExercise: vi.fn().mockReturnValue(of(undefined)),
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
    fixture.detectChanges();
  });

  it('should initialize with default ALGORITMO modality', () => {
    expect(component.labType()).toBe('ALGORITMO');
    expect(component.language()).toBe('python');
    expect(component.memoryLimitMb()).toBe(256);
  });

  it('should switch modality to IDE_PERSISTENTE and configure workspace', () => {
    component.setLabType('IDE_PERSISTENTE');
    expect(component.labType()).toBe('IDE_PERSISTENTE');

    component.setWorkspaceRam(2048);
    expect(component.workspaceRamMb()).toBe(2048);
  });

  it('should auto-fill boilerplate when changing programming language in ALGORITMO mode', () => {
    component.setLanguage('cpp');
    expect(component.language()).toBe('cpp');
    expect(component.boilerplate()).toContain('#include <iostream>');
  });

  it('should validate required title before submission', () => {
    component.title.set('');
    component.submit();
    expect(component.formError()).toBe('El título del laboratorio es obligatorio.');
    expect(mockCourseService.createExercise).not.toHaveBeenCalled();
  });

  it('should submit algorithm lab with test cases', () => {
    component.title.set('Grafos Dijkstra');
    component.testCases.set([
      { input: '4 4', expected_output: '10', is_hidden: false }
    ]);

    component.submit();

    expect(mockCourseService.createExercise).toHaveBeenCalledWith(expect.objectContaining({
      subject_id: 'sub-001',
      title: 'Grafos Dijkstra',
      type: 'algorithm',
      language: 'python'
    }));
  });

  it('should submit persistent workspace lab', () => {
    component.setLabType('IDE_PERSISTENTE');
    component.title.set('Proyecto Data Science con Pandas');
    component.templateId.set('tpl-python-ds');
    component.workspaceRamMb.set(1024);

    component.submit();

    expect(mockCourseService.createExercise).toHaveBeenCalledWith(expect.objectContaining({
      subject_id: 'sub-001',
      title: 'Proyecto Data Science con Pandas',
      type: 'workspace',
      template_id: 'tpl-python-ds',
      memory_limit_mb: 1024
    }));
  });
});
