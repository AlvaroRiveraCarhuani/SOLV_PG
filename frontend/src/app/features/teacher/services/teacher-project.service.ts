import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Observable, map } from 'rxjs';

export interface WorkspaceProjectDTO {
  id: string;
  exercise_id: string;
  current_published_version_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceProjectDraftDTO {
  id: string;
  project_id: string;
  base_version_id?: string | null;
  text_snapshot: string;
  content_sha256: string;
  revision: number;
  updated_at: string;
}

export interface WorkspaceProjectVersionDTO {
  id: string;
  version_number: number;
  text_snapshot: string;
  content_sha256: string;
  file_count: number;
  total_bytes: number;
  published_at: string;
}

export interface WorkspaceLegacyProjectDTO {
  id: string;
  text_snapshot: string;
  source: string;
  captured_at: string;
}

export interface WorkspaceProjectViewDTO {
  project: WorkspaceProjectDTO | null;
  draft?: WorkspaceProjectDraftDTO | null;
  versions: WorkspaceProjectVersionDTO[];
  legacy?: WorkspaceLegacyProjectDTO | null;
  publishing_enabled: boolean;
}

export interface WorkspaceProjectImportResultDTO {
  draft: WorkspaceProjectDraftDTO;
  file_count: number;
  total_bytes: number;
}

export interface WorkspaceProjectProposalFileDTO {
  path: string;
  content: string;
}

// Propuesta de la IA: solo archivos y un resumen. No trae identidad de
// proyecto, revisión ni versión porque no se guarda hasta que el docente la
// incorpora al editor y guarda el borrador.
export interface WorkspaceProjectProposalDTO {
  files: WorkspaceProjectProposalFileDTO[];
  summary: string;
  file_count: number;
  total_bytes: number;
}

export interface WorkspaceProjectGenerationRequestDTO {
  description: string;
  purpose?: string;
  technology?: string;
  requirements?: string;
}

interface ApiEnvelope<T> {
  data: T;
  error?: string;
  message?: string;
}

@Injectable({ providedIn: 'root' })
export class TeacherProjectService {
  private readonly http = inject(HttpClient);

  getProject(exerciseId: string): Observable<WorkspaceProjectViewDTO> {
    return this.http
      .get<ApiEnvelope<WorkspaceProjectViewDTO>>(`/api/v1/teacher/exercises/${exerciseId}/project`)
      .pipe(map((response) => response.data));
  }

  saveDraft(
    exerciseId: string,
    textSnapshot: string,
    expectedRevision: number,
  ): Observable<WorkspaceProjectDraftDTO> {
    return this.http
      .put<ApiEnvelope<WorkspaceProjectDraftDTO>>(
        `/api/v1/teacher/exercises/${exerciseId}/project/draft`,
        {
          text_snapshot: textSnapshot,
          expected_revision: expectedRevision,
        },
      )
      .pipe(map((response) => response.data));
  }

  importFolder(
    exerciseId: string,
    formData: FormData,
  ): Observable<WorkspaceProjectImportResultDTO> {
    return this.http
      .post<ApiEnvelope<WorkspaceProjectImportResultDTO>>(
        `/api/v1/teacher/exercises/${exerciseId}/project/import/folder`,
        formData,
      )
      .pipe(map((response) => response.data));
  }

  importZip(exerciseId: string, formData: FormData): Observable<WorkspaceProjectImportResultDTO> {
    return this.http
      .post<ApiEnvelope<WorkspaceProjectImportResultDTO>>(
        `/api/v1/teacher/exercises/${exerciseId}/project/import/zip`,
        formData,
      )
      .pipe(map((response) => response.data));
  }

  publish(exerciseId: string): Observable<WorkspaceProjectVersionDTO> {
    return this.http
      .post<ApiEnvelope<WorkspaceProjectVersionDTO>>(
        `/api/v1/teacher/exercises/${exerciseId}/project/publish`,
        {},
      )
      .pipe(map((response) => response.data));
  }

  // Pide una propuesta al proveedor configurado. La respuesta es contenido no
  // confiable: el componente la muestra para revisión y solo la incorpora si el
  // docente lo decide.
  generateProject(
    exerciseId: string,
    request: WorkspaceProjectGenerationRequestDTO,
  ): Observable<WorkspaceProjectProposalDTO> {
    return this.http
      .post<ApiEnvelope<WorkspaceProjectProposalDTO>>(
        `/api/v1/teacher/exercises/${exerciseId}/project/generate`,
        request,
      )
      .pipe(map((response) => response.data));
  }
}

export function projectErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof HttpErrorResponse) {
    const body = error.error as { error?: string; message?: string } | null;
    if (body?.message) return body.message;
    if (error.status === 409)
      return 'Otra persona guardó una revisión más nueva. Recarga el borrador antes de continuar.';
    if (error.status === 413) return 'La selección excede el tamaño permitido.';
    if (body?.error) return body.error;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}
