import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';

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

export interface AvailableSatelliteService {
  category: string;
  engine: string;
  label: string;
  version?: string;
  description: string;
  envVar: string;
}

export interface AdminTemplateItem {
  id: string;
  tenant_id?: string;
  name: string;
  docker_image: string;
  base_ram_mb: number;
  status: 'pending' | 'approved' | 'rejected' | 'paused';
  rejection_reason?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  requested_by?: string;
  requested_by_name?: string;
  description?: string;
  target_environment?: string;
  services_config?: ServicesConfig;
  resource_profile?: TemplateResourceProfile;
  setup_script?: string;
  created_at: string;
}

export interface ReviewTemplateDTO {
  status: 'approved' | 'rejected' | 'paused';
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
}

export interface CreateOfficialTemplateDTO {
  name: string;
  docker_image: string;
  base_ram_mb: number;
  description?: string;
  target_environment?: string;
  services_config?: ServicesConfig;
  resource_profile?: TemplateResourceProfile;
  setup_script?: string;
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

  getLocalImages(): Observable<LocalImageItem[]> {
    return this.http.get<ApiResponse<LocalImageItem[]>>(`${this.apiUrl}/local-images`).pipe(
      map(res => res.data || [])
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

  getAvailableSatelliteServices(): AvailableSatelliteService[] {
    return [
      {
        category: 'database',
        engine: 'postgres',
        label: 'PostgreSQL',
        version: '16',
        description: 'Base de datos relacional multi-tenant',
        envVar: 'DATABASE_URL'
      },
      {
        category: 'database',
        engine: 'mysql',
        label: 'MySQL',
        version: '8.0',
        description: 'Base de datos relacional estándar',
        envVar: 'DATABASE_URL'
      },
      {
        category: 'database',
        engine: 'mongodb',
        label: 'MongoDB',
        version: '7.0',
        description: 'Base de datos NoSQL documental',
        envVar: 'MONGODB_URI'
      },
      {
        category: 'cache',
        engine: 'redis',
        label: 'Redis',
        version: '7.2',
        description: 'Almacén en memoria y caché de alto rendimiento',
        envVar: 'REDIS_URL'
      }
    ];
  }
}
