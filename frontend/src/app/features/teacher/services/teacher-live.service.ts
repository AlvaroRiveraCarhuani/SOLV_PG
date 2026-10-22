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
      map(res => {
        const sessions = res.data || [];
        // Process metrics for each session
        sessions.forEach(session => {
          if (session.metrics_available && session.workspace_id) {
            const memHistory = this.memoryHistoryMap.get(session.workspace_id) || [];
            if (session.memory_used_mb !== undefined) {
              const newMemHistory = [...memHistory, session.memory_used_mb].slice(-MAX_HISTORY_SAMPLES);
              this.memoryHistoryMap.set(session.workspace_id, newMemHistory);
              session.memory_history = newMemHistory;
            }
            if (session.cpu_percent !== undefined) {
              const cpuHistory = this.cpuHistoryMap.get(session.workspace_id) || [];
              const newCpuHistory = [...cpuHistory, session.cpu_percent].slice(-MAX_HISTORY_SAMPLES);
              this.cpuHistoryMap.set(session.workspace_id, newCpuHistory);
              session.cpu_history = newCpuHistory;
            }
          }
        });
        return sessions;
      }),
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

  executeTutorCommand(containerId: string, command: string): Observable<TutorCommandResponse> {
    return this.http.post<ApiResponse<TutorCommandResponse>>(
      `/api/v1/teacher/live-sessions/${containerId}/exec`,
      { command }
    ).pipe(
      map(res => res.data)
    );
  }
}
