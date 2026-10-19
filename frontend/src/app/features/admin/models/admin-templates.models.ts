export type TargetEnvironment = 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO';

export interface ImageSuggestion {
  repoTag: string;
  isLocal: boolean;
  isOfficial?: boolean;
  sizeMB?: number;
  usageCount?: number;
  description?: string;
}

export interface ServiceRequirement {
  category: string;
  engine: string;
  version?: string;
}

export interface ServicesConfig {
  services: ServiceRequirement[];
}

export interface TemplateResourceProfile {
  min_mb: number;
  high_mb: number;
  max_mb: number;
}

export interface TemplateDraftFormData {
  name?: string;
  dockerImage?: string;
  targetEnvironment?: TargetEnvironment | string;
  baseRamMB?: number;
  setupScript?: string;
  entrypoint?: string;
  timeoutMS?: number;
  sampleInput?: string;
  description?: string;
  toolsDeclared?: string;
  selectedServices?: ServiceRequirement[];
  selectedCategoryId?: string | null;
  selectedModelId?: string | null;
  [key: string]: unknown;
}

export interface TemplateDraft {
  id: string;
  user_id: string;
  form_data: TemplateDraftFormData;
  template_id?: string;
  updated_at: string;
}

export interface SaveDraftDTO {
  form_data: TemplateDraftFormData;
  template_id?: string;
}

export interface AvailableSatelliteService {
  category: string;
  engine: string;
  label: string;
  version?: string;
  description: string;
  envVar: string;
  isAvailable?: boolean;
  base_ram_mb?: number;
}

export interface HostCapacityInfo {
  total_ram_mb: number;
  available_ram_mb: number;
  used_ram_mb: number;
  cpu_cores: number;
}

export interface SatelliteServiceCapability {
  category: string;
  engine: string;
  label: string;
  version?: string;
  description: string;
  env_var: string;
  is_available: boolean;
  base_ram_mb?: number;
}

export interface RamPresetSuggestion {
  mb: number;
  label: string;
  desc: string;
}

export interface RuntimeCapabilities {
  host_memory: HostCapacityInfo;
  satellite_services: SatelliteServiceCapability[];
  ide_presets: RamPresetSuggestion[];
  judge_presets: RamPresetSuggestion[];
  max_allowed_ram_mb: number;
  /** RAM mínima del editor (OpenVSCode Server). Viene del backend; valor base por defecto: 210 MB. */
  editor_base_mb?: number;
  /** RAM mínima reservada por el motor del Juez. Viene del backend; valor base por defecto: 32 MB. */
  runtime_base_mb?: number;
}

export interface AdminTemplateItem {
  id: string;
  tenant_id?: string;
  name: string;
  docker_image: string;
  base_ram_mb: number;
  status: 'pending' | 'approved' | 'rejected' | 'paused' | 'PENDIENTE_AUDITORIA' | 'APROBADA' | 'RECHAZADA';
  rejection_reason?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  requested_by?: string;
  requested_by_name?: string;
  description?: string;
  target_environment?: string;
  entrypoint?: string;
  timeout_ms?: number;
  services_config?: ServicesConfig;
  resource_profile?: TemplateResourceProfile;
  setup_script?: string;
  tools_declared?: string[];
  smoke_test_status?: string;
  smoke_test_output?: string;
  security_audit_status?: string;
  cve_critical_count?: number;
  cve_high_count?: number;
  security_audited_at?: string;
  eol_status?: 'supported' | 'warning' | 'eol';
  eol_date?: string;
  eol_message?: string;
  eol_checked_at?: string;
  category_id?: string;
  category_name?: string;
  model_id?: string;
  sample_input?: string;
  stack_config?: StackConfig;
  created_at: string;
}

export type ComponentLayer = 'runtime' | 'addon' | 'database';
export type InstallMode = 'none' | 'pip' | 'npm' | 'apt' | 'custom';

export interface TemplateComponent {
  id: string;
  tenant_id?: string;
  slug: string;
  layer: ComponentLayer;
  sub_category: string;
  name: string;
  description: string;
  icon_key: string;
  base_image: string;
  compatible_runtimes: string[];
  install_mode: InstallMode;
  package_names: string[];
  build_layer_snippet: string;
  setup_snippet: string;
  judge_entrypoint: string;
  tools_provided: string[];
  service_engine: string;
  ram_delta_mb: number;
  is_system: boolean;
  is_active: boolean;
  sort_order: number;
}

export type TechCatalogItem = TemplateComponent;
export type TemplateComponentItem = TemplateComponent;

export interface CreateTemplateComponentDTO {
  name: string;
  slug?: string;
  layer: ComponentLayer;
  sub_category?: string;
  description?: string;
  icon_key?: string;
  base_image?: string;
  compatible_runtimes?: string[];
  install_mode?: InstallMode;
  package_names?: string[];
  build_layer_snippet?: string;
  setup_snippet?: string;
  judge_entrypoint?: string;
  tools_provided?: string[];
  service_engine?: string;
  ram_delta_mb?: number;
}

export type CreateTechCatalogItemDTO = CreateTemplateComponentDTO;

export interface StackConfig {
  mode?: 'visual' | 'code';
  runtime_id?: string;
  web_platform_id?: string;
  addon_ids?: string[];
  database_id?: string;
  workspace_folders?: string[];
}

export interface ComposedTemplatePlan {
  suggested_name: string;
  resolved_image: string;
  base_image: string;
  needs_local_build: boolean;
  signature_hash: string;
  dockerfile_content: string;
  setup_script: string;
  tools_declared: string[];
  default_entrypoint: string;
  recommended_ram_mb: number;
  services_config: ServicesConfig;
}

export interface ReviewTemplateDTO {
  status: 'approved' | 'rejected' | 'paused' | 'suspended' | 'pending_audit';
  rejection_reason?: string;
  base_ram_mb?: number;
}

export interface CapabilitiesProbeResult {
  success: boolean;
  detected_runtimes: string[];
  output?: string;
}

export interface TemplateCacheStatus {
  template_id: string;
  template_name: string;
  docker_image: string;
  is_cached: boolean;
  size_bytes: number;
  status: 'cached' | 'missing' | 'warming' | 'failed';
  error_message?: string;
}

export interface WarmAllSummary {
  total_checked: number;
  already_warm: number;
  warmed_count: number;
  failed_count: number;
}

export interface ImageVerificationResult {
  image_ref: string;
  is_local: boolean;
  exists: boolean;
  is_official: boolean;
  origin_type: 'official' | 'verified_registry' | 'community';
  origin_warning?: string;
  architecture_compatible: boolean;
  host_arch: string;
  supported_platforms: string[];
  size_bytes: number;
  size_formatted: string;
  estimated_uncompressed_mb: number;
  host_disk_free_gb: number;
  host_disk_total_gb: number;
  storage_status: 'OK' | 'WARNING' | 'CRITICAL_BLOCKED';
  storage_message: string;
  capabilities_probe?: CapabilitiesProbeResult;
  buildx_suggestion?: string;
  error_message?: string;
  cached: boolean;
  verified_at: string;
}

export interface LocalImageItem {
  repo_tag: string;
  size_mb: number;
  created_at: string;
  is_official: boolean;
  has_latest_tag: boolean;
  usage_count?: number;
}

export interface LocalImagesResult {
  images: LocalImageItem[];
  usage_map: Record<string, number>;
}

export interface CreateOfficialTemplateDTO {
  name: string;
  docker_image: string;
  base_ram_mb: number;
  description?: string;
  target_environment?: string;
  entrypoint?: string;
  timeout_ms?: number;
  services_config?: ServicesConfig;
  resource_profile?: TemplateResourceProfile;
  setup_script?: string;
  tools_declared?: string[];
  category_id?: string;
  category_name?: string;
  model_id?: string;
  sample_input?: string;
  stack_config?: StackConfig;
}

export interface TemplateCategory {
  id: string;
  tenant_id?: string;
  name: string;
  description?: string;
  is_active?: boolean;
  sort_order?: number;
  model_count?: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryDTO {
  name: string;
  description?: string;
}

export interface UpdateCategoryDTO {
  name: string;
  description?: string;
}

export interface ReorderCategoryItem {
  id: string;
  sort_order: number;
}

export interface TemplateModelItem {
  id: string;
  tenant_id?: string;
  category_id?: string | null;
  category_name?: string | null;
  name: string;
  title?: string;
  description?: string;
  docker_image: string;
  target_environment: string;
  base_ram_mb: number;
  entrypoint?: string;
  timeout_ms?: number;
  sample_input?: string;
  usage_count: number;
  is_active?: boolean;
  sort_order?: number;
  source_template_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface UpdateTemplateModelDTO {
  title: string;
  description?: string;
  category_id: string;
}

export interface PromoteTemplateToModelDTO {
  name: string;
  category_id?: string;
  description?: string;
}

export interface ApiResponse<T> {
  code: number;
  data: T;
  message: string;
}
