import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CurricularMapComponent } from './curricular-map.component';
import { StudentService, CourseCurricularMap } from '@core/services/student.service';
import { provideRouter } from '@angular/router';

describe('CurricularMapComponent', () => {
  let component: CurricularMapComponent;
  let fixture: ComponentFixture<CurricularMapComponent>;
  let mockStudentService: { getCourseCurricularMap: ReturnType<typeof vi.fn> };

  const mockMapData: CourseCurricularMap = {
    course_id: 'course-1',
    modules: [
      {
        id: 'mod-1',
        title: 'Introducción',
        description: 'Módulo inicial',
        order_index: 1,
        pass_score: 60,
        state: 'completed',
        prerequisite_module_ids: [],
        exercises: [
          {
            id: 'ex-1',
            title: 'Hola Mundo',
            difficulty: 'easy',
            purpose: 'practice',
            best_score: 100,
            attempts: 1,
            submittable: true
          }
        ]
      },
      {
        id: 'mod-2',
        title: 'Estructuras de Datos',
        description: 'Módulo intermedio',
        order_index: 2,
        pass_score: 60,
        state: 'locked',
        lock_reason: 'Requiere completar Introducción',
        prerequisite_module_ids: ['mod-1'],
        exercises: [
          {
            id: 'ex-2',
            title: 'Examen de Algoritmia',
            difficulty: 'hard',
            purpose: 'exam',
            best_score: undefined,
            attempts: 0,
            submittable: true
          },
          {
            id: 'ex-3',
            title: 'Árboles Binarios',
            difficulty: 'medium',
            purpose: 'practice',
            best_score: undefined,
            attempts: 0,
            submittable: false
          }
        ]
      }
    ],
    unassigned_exercises: [
      {
        id: 'ex-gen-1',
        title: 'Práctica Libre',
        difficulty: 'easy',
        purpose: 'practice',
        best_score: 80,
        attempts: 2,
        submittable: true
      }
    ]
  };

  beforeEach(async () => {
    mockStudentService = {
      getCourseCurricularMap: vi.fn().mockResolvedValue(mockMapData)
    };

    await TestBed.configureTestingModule({
      imports: [CurricularMapComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CurricularMapComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('courseId', 'course-1');
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create and load curricular map data', () => {
    expect(component).toBeTruthy();
    expect(mockStudentService.getCourseCurricularMap).toHaveBeenCalledWith('course-1');
  });

  it('should calculate module progress correctly', () => {
    component.mapData.set(mockMapData);
    expect(component.totalModules()).toBe(2);
    expect(component.completedModules()).toBe(1);
    expect(component.overallProgressPercent()).toBe(50);
  });

  it('should render locked banner when module is locked', () => {
    component.mapData.set(mockMapData);
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    const lockedCard = compiled.querySelector('.module-card.is-locked');
    expect(lockedCard).toBeTruthy();
    expect(lockedCard?.textContent).toContain('Requiere completar Introducción');
  });

  it('should highlight exam exercises even if module is locked', () => {
    component.mapData.set(mockMapData);
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    const examBadge = compiled.querySelector('.exam-badge');
    expect(examBadge).toBeTruthy();
    expect(examBadge?.textContent).toContain('Examen');
  });
});
