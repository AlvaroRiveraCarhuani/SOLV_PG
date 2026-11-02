/** Comentario tal como lo devuelve la API de SOLV. */
export interface SubmissionComment {
  id: string;
  submission_id: string;
  author_name?: string;
  line_number: number;
  end_line_number?: number;
  file_path?: string;
  local_line_number?: number;
  comment: string;
  created_at?: string;
}

/** Comentario ya resuelto a un rango local de un archivo abierto. */
export interface CommentAnchor {
  startLine: number;
  endLine: number;
  comment: SubmissionComment;
}
