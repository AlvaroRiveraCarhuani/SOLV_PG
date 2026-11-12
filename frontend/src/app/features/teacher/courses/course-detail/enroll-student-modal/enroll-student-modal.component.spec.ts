import { ComponentFixture, TestBed } from '@angular/core/testing';
import { EnrollStudentModalComponent } from './enroll-student-modal.component';
import { TeacherCourseService, AvailableStudent } from '../../../services/teacher-course.service';
import { of, throwError } from 'rxjs';

describe('EnrollStudentModalComponent', () => {
  let component: EnrollStudentModalComponent;
  let fixture: ComponentFixture<EnrollStudentModalComponent>;
  let mockTeacherCourseService: {
    searchAvailableStudents: ReturnType<typeof vi.fn>;
    enrollStudent: ReturnType<typeof vi.fn>;
  };

  const mockStudents: AvailableStudent[] = [
    {
      id: 'student-1',
      first_name: 'Alan',
      last_name: 'Turing',
      email: 'alan.turing@institucion.edu'
    },
    {
      id: 'student-2',
      first_name: 'Ada',
      last_name: 'Lovelace',
      email: 'ada.lovelace@institucion.edu'
    }
  ];

  beforeEach(async () => {
    vi.useFakeTimers();
    mockTeacherCourseService = {
      searchAvailableStudents: vi.fn().mockReturnValue(of(mockStudents)),
      enrollStudent: vi.fn().mockReturnValue(of({}))
    };

    await TestBed.configureTestingModule({
      imports: [EnrollStudentModalComponent],
      providers: [
        { provide: TeacherCourseService, useValue: mockTeacherCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EnrollStudentModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('subjectId', 'course-123');
    fixture.detectChanges();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should create component', () => {
    expect(component).toBeTruthy();
  });

  it('should search students when query has at least 2 characters', () => {
    component.onSearchChange('al');
    vi.advanceTimersByTime(350);

    expect(mockTeacherCourseService.searchAvailableStudents).toHaveBeenCalledWith('course-123', 'al');
    expect(component.students().length).toBe(2);
    expect(component.hasSearched()).toBe(true);
  });

  it('should clear results when query has less than 2 characters', () => {
    component.onSearchChange('a');
    vi.advanceTimersByTime(350);

    expect(mockTeacherCourseService.searchAvailableStudents).not.toHaveBeenCalled();
    expect(component.students().length).toBe(0);
    expect(component.hasSearched()).toBe(false);
  });

  it('should allow selecting a student', () => {
    component.selectStudent(mockStudents[0]);
    expect(component.selectedStudent()).toEqual(mockStudents[0]);
  });

  it('should enroll selected student and emit events on success', () => {
    const enrolledSpy = vi.fn();
    const closedSpy = vi.fn();

    component.enrolled.subscribe(enrolledSpy);
    component.closed.subscribe(closedSpy);

    component.selectStudent(mockStudents[0]);
    component.confirmEnroll();

    expect(mockTeacherCourseService.enrollStudent).toHaveBeenCalledWith('course-123', 'student-1');
    expect(enrolledSpy).toHaveBeenCalledWith(mockStudents[0]);
    expect(closedSpy).toHaveBeenCalled();
  });

  it('should handle error when enrollment fails', () => {
    mockTeacherCourseService.enrollStudent.mockReturnValue(
      throwError(() => ({ error: { error: 'Error al matricular' } }))
    );

    component.selectStudent(mockStudents[0]);
    component.confirmEnroll();

    expect(component.errorMessage()).toBe('Error al matricular');
    expect(component.isEnrolling()).toBe(false);
  });
});
