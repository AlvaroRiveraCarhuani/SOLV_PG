import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { RecommendationsPanelComponent } from './recommendations-panel.component';
import { StudentService, StudentRecommendationsDTO } from '@core/services/student.service';

describe('RecommendationsPanelComponent', () => {
  let component: RecommendationsPanelComponent;
  let fixture: ComponentFixture<RecommendationsPanelComponent>;
  let mockStudentService: {
    getCourseRecommendations: ReturnType<typeof vi.fn>;
  };

  const sampleRecommendations: StudentRecommendationsDTO = {
    has_enough_data: true,
    weak_tags: [{ tag: 'recursion', success_rate: 0.33, attempts: 9 }],
    recommendations: [
      {
        exercise_id: 'ex-101',
        title: 'Torres de Hanoi básico',
        difficulty: 'easy',
        matched_tag: 'recursion',
        reason: 'Practica recursión con un ejercicio más sencillo'
      }
    ]
  };

  beforeEach(async () => {
    mockStudentService = {
      getCourseRecommendations: vi.fn().mockResolvedValue(sampleRecommendations)
    };

    await TestBed.configureTestingModule({
      imports: [RecommendationsPanelComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(RecommendationsPanelComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('courseId', 'course-123');
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create and load recommendations on init', () => {
    expect(component).toBeTruthy();
    expect(mockStudentService.getCourseRecommendations).toHaveBeenCalledWith('course-123');
    expect(component.hasEnoughData()).toBe(true);
    expect(component.recommendations().length).toBe(1);
    expect(component.recommendations()[0].title).toBe('Torres de Hanoi básico');
  });

  it('should toggle collapse state when toggleCollapse is called', () => {
    expect(component.isCollapsed()).toBe(false);
    component.toggleCollapse();
    expect(component.isCollapsed()).toBe(true);
    component.toggleCollapse();
    expect(component.isCollapsed()).toBe(false);
  });

  it('should render not enough data state when has_enough_data is false', async () => {
    mockStudentService.getCourseRecommendations.mockResolvedValueOnce({
      has_enough_data: false,
      weak_tags: [],
      recommendations: [],
      message: 'Completa algunos ejercicios para recibir sugerencias personalizadas'
    });

    await component.loadRecommendations();
    fixture.detectChanges();

    expect(component.hasEnoughData()).toBe(false);
    expect(component.message()).toContain('Completa algunos ejercicios');
  });

  it('should render all caught up state when recommendations list is empty', async () => {
    mockStudentService.getCourseRecommendations.mockResolvedValueOnce({
      has_enough_data: true,
      weak_tags: [],
      recommendations: [],
      message: '¡Vas al día! No hay refuerzos sugeridos.'
    });

    await component.loadRecommendations();
    fixture.detectChanges();

    expect(component.hasEnoughData()).toBe(true);
    expect(component.hasRecommendations()).toBe(false);
    expect(component.message()).toContain('¡Vas al día!');
  });
});
