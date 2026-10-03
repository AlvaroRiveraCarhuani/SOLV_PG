import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DashboardKpiWidgetComponent } from './dashboard-kpi-widget.component';

describe('DashboardKpiWidgetComponent', () => {
  let component: DashboardKpiWidgetComponent;
  let fixture: ComponentFixture<DashboardKpiWidgetComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardKpiWidgetComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardKpiWidgetComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('totalStudents', 45);
    fixture.componentRef.setInput('activeNow', 12);
    fixture.componentRef.setInput('pendingReviews', 3);
    fixture.componentRef.setInput('atRisk', 1);
    fixture.detectChanges();
  });

  it('should render KPI values properly', () => {
    expect(component).toBeTruthy();
    expect(component.totalStudents()).toBe(45);
    expect(component.activeNow()).toBe(12);
    expect(component.pendingReviews()).toBe(3);
    expect(component.atRisk()).toBe(1);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('45');
    expect(compiled.textContent).toContain('12');
  });
});
