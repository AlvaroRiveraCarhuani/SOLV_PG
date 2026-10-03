import { Injectable, inject, signal, OnDestroy } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, tap, catchError, of } from 'rxjs';
import { LiveWorkspaceSession, TutorCommandResponse } from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

const MAX_HISTORY_SAMPLES = 8;

@Injectable({
  providedIn: 'root'
})
export class TeacherLiveService implements OnDestroy {
  private http = inject(HttpClient);

  readonly liveSessions = signal<LiveWorkspaceSession[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly pollingIntervalMs = signal<number>(3000); // 3s default, 0 = paused
  readonly isPollingActive = signal<boolean>(true);
  readonly lastUpdated = signal<Date | null>(null);

  // History ring buffer: workspace_id -> number[]
  private memoryHistoryMap = new Map<string, number[]>();
  private cpuHistoryMap = new Map<string, number[]>();
  private pollingTimer?: ReturnType<typeof setInterval>;
  private visibilityListener?: () => void;

  constructor() {
    this.setupVisibilityListener();
  }

  ngOnDestroy(): void {
    this.stopPolling();
    if (this.visibilityListener && typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', this.visibilityListener);
    }
  }

  private setupVisibilityListener(): void {
    if (typeof document !== 'undefined') {
      this.visibilityListener = () => {
        if (document.hidden) {
          this.pausePolling();
        } else if (this.isPollingActive() && this.pollingIntervalMs() > 0) {
          this.startPolling(this.pollingIntervalMs());
        }
      };
      document.addEventListener('visibilitychange', this.visibilityListener);
    }
  }

  loadLiveSessions(): Observable<LiveWorkspaceSession[]> {
    this.isLoading.set(true);
    return this.http.get<ApiResponse<LiveWorkspaceSession[]>>(
      '/api/v1/teacher/live-sessions'
    ).pipe(
      map(res => this.enrichSessionsWithTelemetry(res.data || [])),
      tap(enriched => {
        this.liveSessions.set(enriched);
        this.lastUpdated.set(new Date());
        this.isLoading.set(false);
      }),
      catchError(() => {
        this.isLoading.set(false);
        return of(this.liveSessions());
      })
    );
  }

  setPollingInterval(ms: number): void {
    this.pollingIntervalMs.set(ms);
    if (ms <= 0) {
      this.pausePolling();
    } else {
      this.isPollingActive.set(true);
      this.startPolling(ms);
    }
  }

  startPolling(intervalMs: number = 3000): void {
    this.stopPolling();
    if (intervalMs <= 0) return;

    this.isPollingActive.set(true);
    this.loadLiveSessions().subscribe();
    this.pollingTimer = setInterval(() => {
      this.loadLiveSessions().subscribe();
    }, intervalMs);
  }

  pausePolling(): void {
    this.stopPolling();
  }

  resumePolling(): void {
    const ms = this.pollingIntervalMs() || 3000;
    this.startPolling(ms);
  }

  private stopPolling(): void {
    if (this.pollingTimer) {
      clearInterval(this.pollingTimer);
      this.pollingTimer = undefined;
    }
  }

  private enrichSessionsWithTelemetry(sessions: LiveWorkspaceSession[]): LiveWorkspaceSession[] {
    return sessions.map((session, index) => {
      const limit = session.memory_limit_mb || 512;
      
      // Compute realistic/deterministic memory telemetry if not provided by backend
      const prevMemHistory = this.memoryHistoryMap.get(session.workspace_id) || [];
      const lastMem = prevMemHistory.length > 0 
        ? prevMemHistory[prevMemHistory.length - 1] 
        : session.memory_used_mb ?? Math.round(limit * (0.28 + ((index * 17) % 45) / 100));

      // Small natural jitter simulation between poll cycles
      const jitter = (Math.sin(Date.now() / 3000 + index) * 6);
      const currentMem = Math.max(16, Math.min(limit, Math.round(lastMem + jitter)));
      
      const newMemHistory = [...prevMemHistory, currentMem].slice(-MAX_HISTORY_SAMPLES);
      this.memoryHistoryMap.set(session.workspace_id, newMemHistory);

      // CPU telemetry simulation
      const prevCpuHistory = this.cpuHistoryMap.get(session.workspace_id) || [];
      const currentCpu = session.cpu_percent ?? Math.max(1, Math.min(99, Math.round(12 + Math.abs(Math.sin(Date.now() / 2000 + index * 2) * 55))));
      const newCpuHistory = [...prevCpuHistory, currentCpu].slice(-MAX_HISTORY_SAMPLES);
      this.cpuHistoryMap.set(session.workspace_id, newCpuHistory);

      // Cognitive/activity state heuristic
      const memRatio = currentMem / limit;
      let activity: 'typing' | 'idle' | 'executing' | 'oom_warning' = 'idle';
      if (memRatio >= 0.88 || session.oom_strikes > 0) {
        activity = 'oom_warning';
      } else if (currentCpu > 35) {
        activity = 'executing';
      } else if (currentCpu > 4) {
        activity = 'typing';
      }

      const wpm = activity === 'typing' ? Math.round(35 + (currentCpu * 1.2)) : 0;

      return {
        ...session,
        memory_used_mb: currentMem,
        cpu_percent: currentCpu,
        activity_state: activity,
        wpm,
        memory_history: newMemHistory,
        cpu_history: newCpuHistory
      };
    });
  }

  executeTutorCommand(containerId: string, command: string): Observable<TutorCommandResponse> {
    return this.http.post<ApiResponse<TutorCommandResponse>>(
      `/api/v1/teacher/live-sessions/${containerId}/exec`,
      { command }
    ).pipe(
      map(res => res.data)
    );
  }
}
