export type EnvironmentStatus =
  | 'running'
  | 'pending'
  | 'hibernated'
  | 'failed'
  | 'oom_killed'
  | 'terminated'
  | 'frozen';

export interface Environment {
  id: string;
  environment_id?: string;
  student_id: string;
  subject_id?: string;
  /** Práctica a la que está acotado el entorno. Siempre presente en el modelo
   * por práctica: dos prácticas de la misma materia no comparten repositorio. */
  exercise_id?: string | null;
  tenant_id?: string;
  template_id?: string;
  status: EnvironmentStatus;
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

export type PracticeEnvironment = Environment;
