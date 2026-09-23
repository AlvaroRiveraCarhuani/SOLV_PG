import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map, catchError, of } from 'rxjs';

export interface ServiceRequirement {
  category: string; // "database", "cache", "message_broker", "storage", etc.
  engine: string;   // "postgres", "mysql", "mongodb", "redis", etc.
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

export interface TemplateDraft {
  id: string;
  user_id: string;
  form_data: any;
  template_id?: string;
  updated_at: string;
}

export interface SaveDraftDTO {
  form_data: any;
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
  /** RAM mínima del editor (OpenVSCode Server). Viene del backend; fallback: 210 MB. */
  editor_base_mb?: number;
  /** RAM mínima reservada por el runtime del Juez. Viene del backend; fallback: 32 MB. */
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
  created_at: string;
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

interface ApiResponse<T> {
  code: number;
  data: T;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class AdminTemplatesService {
  private http = inject(HttpClient);
  private apiUrl = '/api/v1/admin/templates';
  private categoriesUrl = '/api/v1/admin/template-categories';
  private modelsUrl = '/api/v1/admin/template-models';

  getTemplates(status?: string, search?: string): Observable<AdminTemplateItem[]> {
    let params = new HttpParams();
    if (status && status !== 'all') {
      params = params.set('status', status);
    }
    if (search && search.trim()) {
      params = params.set('search', search.trim());
    }

    return this.http.get<ApiResponse<AdminTemplateItem[]>>(this.apiUrl, { params }).pipe(
      map(res => res.data || [])
    );
  }

  reviewTemplate(id: string, dto: ReviewTemplateDTO): Observable<AdminTemplateItem> {
    return this.http.put<ApiResponse<AdminTemplateItem>>(`${this.apiUrl}/${id}/review`, dto).pipe(
      map(res => res.data)
    );
  }

  createOfficialTemplate(dto: CreateOfficialTemplateDTO): Observable<AdminTemplateItem> {
    return this.http.post<ApiResponse<AdminTemplateItem>>(this.apiUrl, dto).pipe(
      map(res => res.data)
    );
  }

  duplicateTemplate(id: string): Observable<AdminTemplateItem> {
    return this.http.post<ApiResponse<AdminTemplateItem>>(`${this.apiUrl}/${id}/duplicate`, {}).pipe(
      map(res => res.data)
    );
  }

  getLocalImages(): Observable<LocalImagesResult> {
    return this.http.get<ApiResponse<LocalImagesResult>>(`${this.apiUrl}/local-images`).pipe(
      map(res => res.data || { images: [], usage_map: {} })
    );
  }

  verifyImage(image: string, force = false): Observable<ImageVerificationResult> {
    return this.http.post<ApiResponse<ImageVerificationResult>>(`${this.apiUrl}/verify-image`, {
      image,
      force
    }).pipe(
      map(res => res.data)
    );
  }

  getRuntimeCapabilities(): Observable<RuntimeCapabilities> {
    return this.http.get<ApiResponse<RuntimeCapabilities>>(`${this.apiUrl}/capabilities`).pipe(
      map(res => res.data)
    );
  }

  getCategories(): Observable<TemplateCategory[]> {
    return this.http.get<ApiResponse<TemplateCategory[]>>(this.categoriesUrl).pipe(
      map(res => res.data || [])
    );
  }

  createCategory(dto: CreateCategoryDTO): Observable<TemplateCategory> {
    return this.http.post<ApiResponse<TemplateCategory>>(this.categoriesUrl, dto).pipe(
      map(res => res.data)
    );
  }

  updateCategory(id: string, dto: UpdateCategoryDTO): Observable<TemplateCategory> {
    return this.http.put<ApiResponse<TemplateCategory>>(`${this.categoriesUrl}/${id}`, dto).pipe(
      map(res => res.data)
    );
  }

  deleteCategory(id: string): Observable<void> {
    return this.http.delete<ApiResponse<void>>(`${this.categoriesUrl}/${id}`).pipe(
      map(() => void 0)
    );
  }

  reorderCategories(items: ReorderCategoryItem[]): Observable<void> {
    return this.http.put<ApiResponse<any>>(`${this.categoriesUrl}/reorder`, items).pipe(
      map(() => void 0)
    );
  }

  getModels(targetEnv?: string, categoryId?: string, includeInactive: boolean = false): Observable<TemplateModelItem[]> {
    let params = new HttpParams();
    if (targetEnv) {
      params = params.set('target_environment', targetEnv);
    }
    if (categoryId) {
      params = params.set('category_id', categoryId);
    }
    if (includeInactive) {
      params = params.set('include_inactive', 'true');
    }
    return this.http.get<ApiResponse<TemplateModelItem[]>>(this.modelsUrl, { params }).pipe(
      map(res => (res.data || []).map(m => ({
        ...m,
        name: m.name || m.title || ''
      })))
    );
  }

  updateModel(id: string, dto: UpdateTemplateModelDTO): Observable<TemplateModelItem> {
    return this.http.put<ApiResponse<TemplateModelItem>>(`${this.modelsUrl}/${id}`, dto).pipe(
      map(res => ({
        ...res.data,
        name: res.data.name || res.data.title || ''
      }))
    );
  }

  deactivateModel(id: string): Observable<void> {
    return this.http.post<ApiResponse<any>>(`${this.modelsUrl}/${id}/deactivate`, {}).pipe(
      map(() => void 0)
    );
  }

  reactivateModel(id: string): Observable<void> {
    return this.http.post<ApiResponse<any>>(`${this.modelsUrl}/${id}/reactivate`, {}).pipe(
      map(() => void 0)
    );
  }

  promoteToModel(templateId: string, dto: PromoteTemplateToModelDTO): Observable<TemplateModelItem> {
    return this.http.post<ApiResponse<TemplateModelItem>>(`${this.apiUrl}/${templateId}/promote-to-model`, dto).pipe(
      map(res => res.data)
    );
  }

  saveDraft(formData: any, templateId?: string): Observable<TemplateDraft> {
    const payload: SaveDraftDTO = { form_data: formData, template_id: templateId };
    return this.http.post<ApiResponse<TemplateDraft>>(`${this.apiUrl}/drafts`, payload).pipe(
      map(res => res.data)
    );
  }

  getDraft(): Observable<TemplateDraft | null> {
    return this.http.get<ApiResponse<TemplateDraft>>(`${this.apiUrl}/drafts`).pipe(
      map(res => res.data),
      catchError(err => {
        if (err.status === 404) {
          return of(null);
        }
        throw err;
      })
    );
  }

  deleteDraft(): Observable<void> {
    return this.http.delete<ApiResponse<any>>(`${this.apiUrl}/drafts`).pipe(
      map(() => void 0)
    );
  }

  getAvailableSatelliteServices(): AvailableSatelliteService[] {
    return [
      {
        category: 'database',
        engine: 'postgres',
        label: 'PostgreSQL',
        version: '16',
        description: 'Base de datos relacional PostgreSQL aislada por estudiante y materia',
        envVar: 'DATABASE_URL',
        isAvailable: true
      },
      {
        category: 'database',
        engine: 'mysql',
        label: 'MySQL',
        version: '8.4',
        description: 'Base de datos relacional MySQL para ejercicios de SQL',
        envVar: 'DATABASE_URL',
        isAvailable: true
      },
      {
        category: 'database',
        engine: 'mongodb',
        label: 'MongoDB',
        version: '7.0',
        description: 'Base de datos de documentos NoSQL para proyectos web',
        envVar: 'MONGODB_URI',
        isAvailable: false
      },
      {
        category: 'cache',
        engine: 'redis',
        label: 'Redis',
        version: '7.2',
        description: 'Almacén en memoria y caché clave-valor',
        envVar: 'REDIS_URL',
        isAvailable: false
      }
    ];
  }
}
