import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, map } from 'rxjs';
import {
  TeacherSubmissionReviewDTO,
  OverrideRequestDTO,
  AddCommentRequestDTO,
  SubmissionComment,
  EphemeralRunRequestDTO,
  EphemeralRunResult,
  SubmissionTimeline,
  SubmissionKeystrokeReport,
  ReviewEnvironmentAccess
} from '../models/teacher.models';

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TeacherGradingService {
  private http = inject(HttpClient);

  readonly currentReview = signal<TeacherSubmissionReviewDTO | null>(null);
  readonly comments = signal<SubmissionComment[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly isRunningEphemeral = signal<boolean>(false);
  readonly ephemeralResult = signal<EphemeralRunResult | null>(null);
  readonly isStartingReviewEnvironment = signal<boolean>(false);

  getSubmissionReview(submissionId: string): Observable<TeacherSubmissionReviewDTO> {
    this.isLoading.set(true);
    return this.http.get<ApiResponse<TeacherSubmissionReviewDTO>>(
      `/api/v1/teacher/submissions/${submissionId}/review`
    ).pipe(
      map(res => res.data),
      tap(review => {
        this.currentReview.set(review);
        this.comments.set(review.comments || []);
        this.isLoading.set(false);
      })
    );
  }

  getSubmissionTimeline(submissionId: string): Observable<SubmissionTimeline> {
    return this.http.get<ApiResponse<SubmissionTimeline>>(
      `/api/v1/teacher/submissions/${submissionId}/timeline`
    ).pipe(
      map(res => res.data)
    );
  }

  getKeystrokeEvents(submissionId: string): Observable<SubmissionKeystrokeReport> {
    return this.http.get<ApiResponse<SubmissionKeystrokeReport>>(
      `/api/v1/teacher/submissions/${submissionId}/keystroke-events`
    ).pipe(
      map(res => res.data)
    );
  }

  /**
   * Corrige la nota de una entrega sin tocar el veredicto. Es el camino de los
   * proyectos de IDE, donde el veredicto no existe: la nota es del docente.
   */
  updateReviewScore(
    submissionId: string,
    dto: { status: string; score?: number; feedback_summary?: string }
  ): Observable<void> {
    return this.http.patch<ApiResponse<{ cell: unknown }>>(
      `/api/v1/teacher/submissions/${submissionId}/review`,
      dto
    ).pipe(
      map(() => void 0),
      tap(() => {
        const curr = this.currentReview();
        if (curr && dto.score !== undefined) {
          this.currentReview.set({ ...curr, score: dto.score });
        }
      })
    );
  }

  overrideSubmission(submissionId: string, dto: OverrideRequestDTO): Observable<void> {
    return this.http.post<ApiResponse<void>>(
      `/api/v1/submissions/${submissionId}/override`,
      dto
    ).pipe(
      map(() => void 0),
      tap(() => {
        const curr = this.currentReview();
        if (curr) {
          this.currentReview.set({
            ...curr,
            verdict: dto.verdict,
            manual_override: true,
            override_reason: dto.override_reason,
            score: dto.score !== undefined ? dto.score : curr.score
          });
        }
      })
    );
  }

  addComment(submissionId: string, dto: AddCommentRequestDTO): Observable<SubmissionComment> {
    return this.http.post<ApiResponse<SubmissionComment>>(
      `/api/v1/teacher/submissions/${submissionId}/comments`,
      dto
    ).pipe(
      map(res => res.data),
      tap(newComment => {
        this.comments.update(list => [...list, newComment]);
        const curr = this.currentReview();
        if (curr) {
          this.currentReview.set({
            ...curr,
            comments: [...(curr.comments || []), newComment]
          });
        }
      })
    );
  }

  getComments(submissionId: string): Observable<SubmissionComment[]> {
    return this.http.get<ApiResponse<SubmissionComment[]>>(
      `/api/v1/teacher/submissions/${submissionId}/comments`
    ).pipe(
      map(res => res.data || []),
      tap(comments => this.comments.set(comments))
    );
  }

  runEphemeral(submissionId: string, dto: EphemeralRunRequestDTO): Observable<EphemeralRunResult> {
    this.isRunningEphemeral.set(true);
    return this.http.post<ApiResponse<EphemeralRunResult>>(
      `/api/v1/teacher/submissions/${submissionId}/run-ephemeral`,
      dto
    ).pipe(
      map(res => res.data),
      tap({
        next: result => {
          this.ephemeralResult.set(result);
          this.isRunningEphemeral.set(false);
        },
        error: () => {
          this.isRunningEphemeral.set(false);
        }
      })
    );
  }


  /**
   * Abre (o reabre) el preview de la entrega IDE. El backend devuelve una URL
   * pública y el instante en que el QoS la apaga por inactividad.
   */
  startReviewEnvironment(submissionId: string): Observable<ReviewEnvironmentAccess> {
    this.isStartingReviewEnvironment.set(true);
    return this.http.post<ApiResponse<ReviewEnvironmentAccess>>(
      `/api/v1/teacher/submissions/${submissionId}/test-environment`,
      {}
    ).pipe(
      map(res => res.data),
      tap({
        next: () => this.isStartingReviewEnvironment.set(false),
        error: () => this.isStartingReviewEnvironment.set(false)
      })
    );
  }
}
