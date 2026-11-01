export type WorkspaceStatus =
  | 'running'
  | 'pending'
  | 'hibernated'
  | 'failed'
  | 'oom_killed'
  | 'terminated'
  | 'frozen';

export interface Workspace {
  id: string;
  workspace_id?: string;
  student_id: string;
  subject_id?: string;
  tenant_id?: string;
  template_id?: string;
  status: WorkspaceStatus;
  subdomain?: string;
  url?: string;
  access_url?: string;
  type?: string;
  memory_limit_mb: number;
  ram_limit_mb?: number;
  cpu_limit?: number;
  oom_strikes?: number;
  last_activity_at?: string;
  last_heartbeat_at?: string;
  /** Hora del cierre del examen. Con valor, el volumen quedó en solo lectura. */
  frozen_at?: string | null;
  created_at: string;
}

export type WorkspaceInstance = Workspace;
