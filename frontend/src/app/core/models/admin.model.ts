export interface HostHardwareMetrics {
  ram_used_bytes: number;
  ram_total_bytes: number;
  ram_percent: number;
  cpu_cores: number;
  cpu_percent: number;
  disk_used_bytes: number;
  disk_total_bytes: number;
  disk_percent: number;
  /** Ruta real medida por el backend (vía gopsutil), no un literal de UI. */
  disk_path?: string;
  containers_active: number;
  containers_hibernated: number;
  containers_max: number;
  host_safety_ceiling_pct?: number;
  estimated_max_containers?: number;
  /** Qué límite mandó en el cálculo: la política del tenant o la RAM disponible. */
  capacity_basis?: 'policy' | 'mem_available';
  max_containers_policy?: number;
  ram_per_lab_mb?: number;
}

export interface DockerContainerSummary {
  id: string;
  student_name: string;
  student_email: string;
  course_name: string;
  image_tag: string;
  memory_used_mb: number;
  /** false cuando no hay telemetría real del contenedor: la UI muestra «Sin métricas». */
  metrics_available?: boolean;
  memory_limit_mb: number;
  ttl_remaining_seconds: number;
  /** unknown = daemon no respondió durante la reconciliación; orphaned = contenedor ausente (ADR-049). */
  status: 'running' | 'hibernated' | 'failed' | 'unknown' | 'orphaned';
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

export interface LoadSnapshot {
  /** Timestamp real (RFC3339) del sample persistido por el worker; el gráfico calcula los deltas desde acá. */
  timestamp: string;
  ram_percent: number;
  ram_used_gb: number;
  cpu_percent: number;
  active_containers: number;
}

/** Daemon de Docker medido en el nodo (S6): nunca un literal fijo. */
export interface DockerDaemonInfo {
  server_version: string;
  storage_driver: string;
  healthy: boolean;
}

export interface HostSystemHealth {
  status: 'operational' | 'degraded' | 'critical';
  docker: DockerDaemonInfo;
  /** Línea formateada para el encabezado, derivada de docker (S7: sin literales). */
  docker_version?: string;
  /** Códigos estables del backend: docker_unreachable, orphaned_workspaces, ... */
  health_reasons?: string[];
  uptime_seconds: number;
  metrics: HostHardwareMetrics;
  containers: DockerContainerSummary[];
  courses_load: CourseLoadSummary[];
  incidents: TechnicalIncident[];
  load_history?: LoadSnapshot[];
}

export interface WorkspaceLogsResponse {
  workspace_id: string;
  container_id: string;
  student_id: string;
  status: string;
  logs: string;
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

export interface DirectTeacherPayload {
  email: string;
  first_name?: string;
  last_name?: string;
  role_type?: TeacherRoleType;
  send_notification?: boolean;
}

export interface ParsedRosterResult {
  total_detected: number;
  valid_emails: string[];
  duplicate_emails: string[];
  invalid_emails: string[];
}

export interface BatchDirectTeacherPayload {
  emails: string[];
  send_notification?: boolean;
}

export interface BatchDirectTeacherResponse {
  message: string;
  created_count: number;
}
