import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, tap } from 'rxjs';
import { LiveWorkspaceSession, TutorCommandResponse } from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TeacherLiveService {
  private http = inject(HttpClient);

  readonly liveSessions = signal<LiveWorkspaceSession[]>([]);
  readonly isLoading = signal<boolean>(false);

  loadLiveSessions(): Observable<LiveWorkspaceSession[]> {
    this.isLoading.set(true);
    return this.http.get<ApiResponse<LiveWorkspaceSession[]>>(
      '/api/v1/teacher/live-sessions'
    ).pipe(
      map(res => res.data || []),
      tap(sessions => {
        this.liveSessions.set(sessions);
        this.isLoading.set(false);
      })
    );
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
