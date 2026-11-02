/** Entrega del alumno tal como la devuelve GET /api/v1/student/submissions. */
export interface StudentSubmission {
  id: string;
  exercise_id: string;
  verdict: string;
  review_status: string;
  score?: number | null;
  submitted_at: string;
  exercise_title?: string;
  exercise_type?: string;
  exercise_purpose?: string;
  subject_id?: string;
  subject_code?: string;
  subject_name?: string;
}
