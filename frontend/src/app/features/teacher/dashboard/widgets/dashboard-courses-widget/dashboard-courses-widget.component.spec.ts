import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { DashboardCoursesWidgetComponent } from './dashboard-courses-widget.component';
import { CourseColorService } from '@core/services/course-color.service';

describe('DashboardCoursesWidgetComponent', () => {
  let component: DashboardCoursesWidgetComponent;
  let fixture: ComponentFixture<DashboardCoursesWidgetComponent>;
  let colorService: CourseColorService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardCoursesWidgetComponent],
      providers: [
        provideRouter([]),
        CourseColorService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardCoursesWidgetComponent);
    component = fixture.componentInstance;
    colorService = TestBed.inject(CourseColorService);
    fixture.componentRef.setInput('courses', [
      { id: 'c1', name: 'Algoritmos y Estructuras', code: 'CS101', students_count: 30, active_now: 8, pending_review: 4, at_risk: 1 }
    ]);
    fixture.detectChanges();
  });

  it('should render course cards and retrieve dynamic color accent', () => {
    expect(component).toBeTruthy();
    expect(component.courses().length).toBe(1);

    const color = component.getCourseColor('c1', 'CS101');
    expect(color).toBeTruthy();

    const style = component.getCourseThemeStyle('c1', 'CS101');
    expect(style.accentBg).toContain('rgba');
  });

  it('should toggle color picker popover', () => {
    const mockEvent = new Event('click');
    component.toggleCourseColorPicker('c1', mockEvent);
    expect(component.activeColorPickerCourseId()).toBe('c1');

    component.toggleCourseColorPicker('c1', mockEvent);
    expect(component.activeColorPickerCourseId()).toBeNull();
  });
});
