export type EnvironmentStatus =
  | 'running'
  | 'pending'
  | 'hibernated'
  | 'failed'
  | 'oom_killed'
  | 'terminated'
  | 'frozen';

/**
 * Estado que ve el alumno, calculado por el servidor: el entorno se está
 * armando, ya se puede trabajar, el examen terminó o el arranque falló.
 */
export type EnvironmentState =
  | 'preparing'
  | 'ready'
  | 'closed'
  | 'failed'
  | 'disk_warning'
  | 'disk_full';

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
  /** URL de la aplicación del alumno publicada en su propio puerto. Solo existe
   * cuando la práctica declara app_port. */
  app_url?: string;
  type?: string;
  memory_limit_mb: number;
  /** Perfil que declaró la plantilla antes del préstamo de memoria del nodo. */
  memory_profile_mb?: number;
  /** Cuota absoluta de CPU aplicada al contenedor, en milésimas. */
  cpu_millis?: number;
  ram_limit_mb?: number;
  cpu_limit?: number;
  oom_strikes?: number;
  last_activity_at?: string;
  last_heartbeat_at?: string;
  /** Hora del cierre del examen. Con valor, el volumen quedó en solo lectura. */
  frozen_at?: string | null;
  /** Estado calculado por el servidor; el cliente no lo inventa. */
  state?: EnvironmentState;
  /** Última medición del volumen y cuota del nodo. Solo informativo: la cuota
   * es blanda y el servidor decide si frena la entrega. */
  disk_used_mb?: number;
  disk_limit_mb?: number;
  disk_state?: 'ok' | 'warning' | 'full';
  created_at: string;
}

export type PracticeEnvironment = Environment;
