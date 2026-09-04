export type WorkspaceStatus = 'running' | 'pending' | 'hibernated' | 'failed' | 'oom_killed' | 'terminated';

export interface Workspace {
  id: string;
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
  created_at: string;
}

export type WorkspaceInstance = Workspace;
