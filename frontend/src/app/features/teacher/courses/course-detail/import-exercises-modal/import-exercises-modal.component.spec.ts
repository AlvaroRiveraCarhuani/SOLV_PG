import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ImportExercisesModalComponent } from './import-exercises-modal.component';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { of } from 'rxjs';
import { ExerciseImportResponse } from '../../../models/teacher.models';

describe('ImportExercisesModalComponent', () => {
  let component: ImportExercisesModalComponent;
  let fixture: ComponentFixture<ImportExercisesModalComponent>;
  let mockTeacherCourseService: jasmine.SpyObj<TeacherCourseService>;

  const mockPreviewResponse: ExerciseImportResponse = {
    mode: 'dry_run',
    exercises_count: 1,
    can_import: true,
    total_valid: 1,
    total_invalid: 0,
    exercises: [
      {
        index: 0,
        title: 'Ejercicio de prueba',
        valid: true,
        errors: []
      }
    ]
  };

  beforeEach(async () => {
    mockTeacherCourseService = jasmine.createSpyObj('TeacherCourseService', ['importExercises']);
    mockTeacherCourseService.importExercises.and.returnValue(of(mockPreviewResponse));

    await TestBed.configureTestingModule({
      imports: [ImportExercisesModalComponent],
      providers: [
        { provide: TeacherCourseService, useValue: mockTeacherCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ImportExercisesModalComponent);
    component = fixture.componentInstance;
    component.subjectId = 'course-123';
    fixture.detectChanges();
  });

  it('should create component', () => {
    expect(component).toBeTruthy();
  });

  it('should analyze file when valid file is selected', () => {
    const file = new File(['[{"title": "Ex"}]'], 'exercises.json', { type: 'application/json' });
    const event = { target: { files: [file] } } as unknown as Event;

    component.onFileSelected(event);

    expect(component.selectedFile()).toBe(file);
    expect(mockTeacherCourseService.importExercises).toHaveBeenCalledWith('course-123', file, true);
    expect(component.importPreview()).toEqual(mockPreviewResponse);
  });

  it('should show error when unsupported file extension is selected', () => {
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' });
    const event = { target: { files: [file] } } as unknown as Event;

    component.onFileSelected(event);

    expect(component.selectedFile()).toBeNull();
    expect(component.errorMessage()).toContain('Solo se permiten archivos en formato JSON');
  });

  it('should confirm import when confirmImport is called', () => {
    const file = new File(['[{"title": "Ex"}]'], 'exercises.json', { type: 'application/json' });
    component.selectedFile.set(file);
    component.importPreview.set(mockPreviewResponse);

    const actualResponse: ExerciseImportResponse = {
      mode: 'import',
      can_import: true,
      imported_count: 1
    };
    mockTeacherCourseService.importExercises.and.returnValue(of(actualResponse));

    spyOn(component.imported, 'emit');
    spyOn(component.closed, 'emit');

    component.confirmImport();

    expect(mockTeacherCourseService.importExercises).toHaveBeenCalledWith('course-123', file, false);
    expect(component.imported.emit).toHaveBeenCalledWith(1);
    expect(component.closed.emit).toHaveBeenCalled();
  });
});
