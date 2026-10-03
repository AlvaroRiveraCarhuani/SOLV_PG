import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DashboardTelemetryWidgetComponent } from './dashboard-telemetry-widget.component';
import { LiveWorkspaceSession } from '../../../models/teacher.models';

describe('DashboardTelemetryWidgetComponent', () => {
  let component: DashboardTelemetryWidgetComponent;
  let fixture: ComponentFixture<DashboardTelemetryWidgetComponent>;

  const mockSessions: LiveWorkspaceSession[] = [
    {
      workspace_id: 'ws-101',
      container_id: 'cont-101',
      student_id: 'std-101',
      student_name: 'Lucía Fernández',
      student_email: 'lucia@uab.edu.bo',
      subject_id: 'sub-1',
      subject_name: 'Sistemas Operativos',
      status: 'running',
      memory_used_mb: 256,
      memory_limit_mb: 512,
      memory_history: [120, 180, 256],
      cpu_percent: 15,
      cpu_history: [5, 10, 15],
      activity_state: 'typing',
      wpm: 42,
      oom_strikes: 0,
      last_heartbeat: new Date().toISOString(),
      is_attached: false
    }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardTelemetryWidgetComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardTelemetryWidgetComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('sessions', mockSessions);
    fixture.detectChanges();
  });

  it('should render live telemetry and calculate metrics', () => {
    expect(component).toBeTruthy();
    expect(component.sessions().length).toBe(1);
    expect(component.getMemoryPercent(mockSessions[0])).toBe(50);
    expect(component.getStudentInitials('Lucía Fernández')).toBe('LF');
  });

  it('filters live sessions by search term', () => {
    component.liveSearchTerm.set('Sistemas');
    expect(component.filteredSessions().length).toBe(1);

    component.liveSearchTerm.set('Inexistente');
    expect(component.filteredSessions().length).toBe(0);
  });

  it('emits polling rate change and terminal actions', () => {
    let emittedRate = 0;
    component.pollingRateChange.subscribe(rate => emittedRate = rate);
    component.onSetPollingRate(5000);
    expect(emittedRate).toBe(5000);

    let terminalSession: LiveWorkspaceSession | null = null;
    component.openTerminal.subscribe(s => terminalSession = s);
    component.openTerminal.emit(mockSessions[0]);
    expect(terminalSession).toEqual(mockSessions[0]);
  });
});
