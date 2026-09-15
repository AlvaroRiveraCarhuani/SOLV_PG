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

export interface HostSystemHealth {
  status: 'healthy' | 'degraded' | 'critical';
  docker_version: string;
  uptime_seconds: number;
  metrics: HostHardwareMetrics;
  containers: DockerContainerSummary[];
}
