import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { DashboardAttentionWidgetComponent } from './dashboard-attention-widget.component';

describe('DashboardAttentionWidgetComponent', () => {
  let component: DashboardAttentionWidgetComponent;
  let fixture: ComponentFixture<DashboardAttentionWidgetComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardAttentionWidgetComponent],
      providers: [provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardAttentionWidgetComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('attention', {
      critical: [{
        type: 'oom_killed',
        student_id: 's1',
        student_name: 'Carlos Alumno',
        workspace_id: 'ws1',
        subject_id: 'sub1',
        occurred_at: new Date().toISOString()
      }],
      warning: [{
        type: 'ast_blocked',
        student_id: 's2',
        student_name: 'Maria Estudiante',
        exercise_id: 'ex1',
        rule_violated: 'system() call prohibited',
        occurred_at: new Date().toISOString()
      }],
      standard: []
    });
    fixture.detectChanges();
  });

  it('should compute alert counts correctly', () => {
    expect(component).toBeTruthy();
    expect(component.criticalCount()).toBe(1);
    expect(component.warningCount()).toBe(1);
    expect(component.standardCount()).toBe(0);
    expect(component.totalAlertsCount()).toBe(2);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Exceso de Memoria (OOM)');
    expect(compiled.textContent).toContain('Carlos Alumno');
  });
});
