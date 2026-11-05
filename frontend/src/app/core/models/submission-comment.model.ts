export interface SubmissionComment {
  id: string;
  tenant_id?: string;
  submission_id: string;
  author_id: string;
  author_name?: string;
  line_number: number;
  end_line_number?: number;
  file_path?: string;
  local_line_number?: number;
  comment: string;
  created_at: string;
}
