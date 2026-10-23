import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { firstValueFrom } from 'rxjs';
import { TeacherLiveService } from './teacher-live.service';
import { LiveWorkspaceSession } from '../models/teacher.models';

describe('TeacherLiveService', () => {
  let service: TeacherLiveService;
  let httpMock: HttpTestingController;

  const mockSessions: LiveWorkspaceSession[] = [
    {
      workspace_id: 'ws-1',
      container_id: 'cont-1',
      student_id: 'std-1',
      student_name: 'Ana García',
      student_email: 'ana@uab.edu.bo',
      subject_id: 'sub-1',
      subject_name: 'Estructuras de Datos',
      status: 'running',
      memory_used_mb: 256,
      memory_limit_mb: 512,
      cpu_percent: 12,
      metrics_available: true,
      activity_state: 'typing',
      oom_strikes: 0,
      last_heartbeat: new Date().toISOString(),
      is_attached: false
    }
  ];

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        TeacherLiveService,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    });

    service = TestBed.inject(TeacherLiveService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    service.ngOnDestroy();
    httpMock.verify();
  });

  it('loads live sessions and enriches with telemetry metrics', async () => {
    const sessionsPromise = firstValueFrom(service.loadLiveSessions());

    const req = httpMock.expectOne('/api/v1/teacher/live-sessions');
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockSessions });

    const sessions = await sessionsPromise;
    expect(sessions.length).toBe(1);
    expect(sessions[0].memory_used_mb).toBeGreaterThan(0);
    expect(sessions[0].memory_history).toBeDefined();
    expect(sessions[0].memory_history).toEqual([256]);
    expect(sessions[0].cpu_history).toEqual([12]);
    expect(sessions[0].activity_state).toBeDefined();
  });

  it('updates polling interval configuration', () => {
    service.setPollingInterval(5000);
    expect(service.pollingIntervalMs()).toBe(5000);
    expect(service.isPollingActive()).toBe(true);

    const req = httpMock.expectOne('/api/v1/teacher/live-sessions');
    req.flush({ data: mockSessions });

    service.setPollingInterval(0);
    expect(service.pollingIntervalMs()).toBe(0);
  });
});
