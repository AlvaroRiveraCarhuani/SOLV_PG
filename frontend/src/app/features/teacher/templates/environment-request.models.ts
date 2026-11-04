export type EnvironmentRequestStatus =
  | 'EN_REVISION'
  | 'APROBADA'
  | 'NECESITA_CAMBIO'
  | 'NO_VIABLE';

/** Uso declarado por el docente: describe la clase, no la máquina. */
export type EnvironmentRequestUse = 'practice' | 'grade';

export interface EnvironmentRequestComponent {
  id: string;
  slug: string;
  name: string;
  layer: 'runtime' | 'addon' | 'database' | string;
  icon_key?: string;
  base_image?: string;
}

export interface EnvironmentRequestPreviewComponent {
  id: string;
  slug: string;
  name: string;
  layer: string;
  icon_key?: string;
  base_image?: string;
  missing: boolean;
}

export interface EnvironmentMissingComponent {
  name: string;
  layer: string;
  category?: string;
  reason?: string;
}

export interface EnvironmentRequestSubject {
  subject_id?: string;
  subject_name: string;
  requested_by?: string;
  requester_name?: string;
  added_at: string;
}

export interface EnvironmentRequest {
  id: string;
  tenant_id: string;
  requested_by: string;
  requester_name: string;
  title: string;
  description: string;
  teacher_text: string;
  uses: EnvironmentRequestUse[];
  target_environment: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS' | string;
  components: EnvironmentRequestComponent[];
  missing_components: EnvironmentMissingComponent[];
  subjects: EnvironmentRequestSubject[];
  stack_signature: string;
  status: EnvironmentRequestStatus;
  review_note: string;
  reviewed_by?: string;
  reviewed_at?: string;
  resolved_template_id?: string;
  created_at: string;
  updated_at: string;
}

export interface EnvironmentRequestDuplicate {
  id: string;
  title: string;
  status: EnvironmentRequestStatus;
  subjects: string[];
  requested_by_me: boolean;
}

export interface EnvironmentRequestTemplateRef {
  id: string;
  name: string;
}

export interface EnvironmentRequestPreview {
  suggested_name: string;
  target_environment: string;
  recommended_ram_mb: number;
  signature: string;
  components: EnvironmentRequestPreviewComponent[];
  missing_components: EnvironmentMissingComponent[];
  duplicate_request?: EnvironmentRequestDuplicate;
  existing_template?: EnvironmentRequestTemplateRef;
}

export interface EnvironmentRequestStackConfig {
  mode?: string;
  runtime_id?: string;
  /** Las plantillas anteriores al constructor declaran lenguaje y versión en lugar de ids. */
  language?: string;
  version?: string;
  web_platform_id?: string;
  addon_ids?: string[];
  database_id?: string;
  environment_folders?: string[];
}

export interface CreateEnvironmentRequestPayload {
  title: string;
  description: string;
  teacher_text: string;
  uses: EnvironmentRequestUse[];
  stack_config: EnvironmentRequestStackConfig;
  missing_components?: EnvironmentMissingComponent[];
  subject_id?: string;
  subject_name?: string;
}

export interface EnvironmentRequestCreated {
  request: EnvironmentRequest;
  joined: boolean;
  message: string;
}

export interface ReviewEnvironmentRequestPayload {
  status: EnvironmentRequestStatus;
  review_note: string;
  resolved_template_id?: string;
}

/** Etiquetas en palabras de persona: los estados técnicos no salen de la API. */
export const ENVIRONMENT_REQUEST_STATUS_LABEL: Record<EnvironmentRequestStatus, string> = {
  EN_REVISION: 'En revisión',
  APROBADA: 'Aprobada',
  NECESITA_CAMBIO: 'Necesita un cambio',
  NO_VIABLE: 'No es viable'
};

export const ENVIRONMENT_REQUEST_STATUS_TONE: Record<EnvironmentRequestStatus, string> = {
  EN_REVISION: 'warning',
  APROBADA: 'success',
  NECESITA_CAMBIO: 'warning',
  NO_VIABLE: 'error'
};
