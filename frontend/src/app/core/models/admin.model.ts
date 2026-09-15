export interface HostHardwareMetrics {
  ram_used_bytes: number;
  ram_total_bytes: number;
  ram_percent: number;
  cpu_cores: number;
  cpu_percent: number;
  disk_used_bytes: number;
  disk_total_bytes: number;
  disk_percent: number;
  containers_active: number;
  containers_hibernated: number;
  containers_max: number;
}

export interface DockerContainerSummary {
  id: string;
  student_name: string;
  student_email: string;
  course_name: string;
  image_tag: string;
  memory_used_mb: number;
  memory_limit_mb: number;
  ttl_remaining_seconds: number;
  status: 'running' | 'hibernated' | 'failed';
  started_at: string;
}

export interface CourseLoadSummary {
  id: string;
  course_name: string;
  teacher_name: string;
  active_students: number;
  hibernated_students: number;
  ram_used_mb: number;
}

export interface TechnicalIncident {
  id: string;
  type: 'oom_killed' | 'start_failure' | 'qos_warning';
  workspace_id: string;
  student_name: string;
  course_name: string;
  description: string;
  memory_limit_mb?: number;
  timestamp: string;
}

export interface HostSystemHealth {
  status: 'healthy' | 'degraded' | 'critical';
  docker_version: string;
  uptime_seconds: number;
  metrics: HostHardwareMetrics;
  containers: DockerContainerSummary[];
  courses_load: CourseLoadSummary[];
  incidents: TechnicalIncident[];
}

export type TeacherOrigin = 'manual' | 'gclassroom';
export type TeacherStatus = 'active' | 'pending' | 'expired';
export type TeacherRoleType = 'titular' | 'auxiliar';

export interface TeacherItem {
  id: string;
  full_name: string;
  email: string;
  origin: TeacherOrigin;
  status: TeacherStatus;
  role_type: TeacherRoleType;
  invited_at: string;
  last_login?: string;
  active_courses?: number;
}

export interface TeacherInvitationPayload {
  email: string;
  role: 'teacher';
  role_type: TeacherRoleType;
  send_email?: boolean;
}
