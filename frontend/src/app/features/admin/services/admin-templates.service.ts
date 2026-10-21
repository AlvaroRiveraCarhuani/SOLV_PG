import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map, catchError, of } from 'rxjs';
import {
  AdminTemplateItem,
  ApiResponse,
  ComponentLayer,
  ComposedTemplatePlan,
  CreateCategoryDTO,
  CreateOfficialTemplateDTO,
  CreateTemplateComponentDTO,
  UpdateTemplateComponentDTO,
  ImageVerificationResult,
  LocalImagesResult,
  PromoteTemplateToModelDTO,
  ReorderCategoryItem,
  ReviewTemplateDTO,
  RuntimeCapabilities,
  SaveDraftDTO,
  StackConfig,
  TemplateCacheStatus,
  TemplateCategory,
  TemplateComponent,
  TemplateDraft,
  TemplateDraftFormData,
  TemplateModelItem,
  UpdateCategoryDTO,
  UpdateTemplateModelDTO,
  WarmAllSummary
} from '../models/admin-templates.models';

export * from '../models/admin-templates.models';

@Injectable({
  providedIn: 'root'
})
export class AdminTemplatesService {
  private http = inject(HttpClient);
  private apiUrl = '/api/v1/admin/templates';
  private categoriesUrl = '/api/v1/admin/template-categories';
  private modelsUrl = '/api/v1/admin/template-models';
  private componentsUrl = '/api/v1/admin/template-components';

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
    return this.http.put<ApiResponse<unknown>>(`${this.categoriesUrl}/reorder`, items).pipe(
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
    return this.http.post<ApiResponse<unknown>>(`${this.modelsUrl}/${id}/deactivate`, {}).pipe(
      map(() => void 0)
    );
  }

  reactivateModel(id: string): Observable<void> {
    return this.http.post<ApiResponse<unknown>>(`${this.modelsUrl}/${id}/reactivate`, {}).pipe(
      map(() => void 0)
    );
  }

  promoteToModel(templateId: string, dto: PromoteTemplateToModelDTO): Observable<TemplateModelItem> {
    return this.http.post<ApiResponse<TemplateModelItem>>(`${this.apiUrl}/${templateId}/promote-to-model`, dto).pipe(
      map(res => res.data)
    );
  }

  saveDraft(formData: TemplateDraftFormData, templateId?: string): Observable<TemplateDraft> {
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
    return this.http.delete<ApiResponse<unknown>>(`${this.apiUrl}/drafts`).pipe(
      map(() => void 0)
    );
  }

  getTemplatesCacheStatus(): Observable<TemplateCacheStatus[]> {
    return this.http.get<ApiResponse<TemplateCacheStatus[]>>(`${this.apiUrl}/cache-status`).pipe(
      map(res => res.data || [])
    );
  }

  warmTemplate(id: string): Observable<TemplateCacheStatus> {
    return this.http.post<ApiResponse<TemplateCacheStatus>>(`${this.apiUrl}/${id}/warm`, {}).pipe(
      map(res => res.data)
    );
  }

  warmAllTemplates(): Observable<WarmAllSummary> {
    return this.http.post<ApiResponse<WarmAllSummary>>(`${this.apiUrl}/warm-all`, {}).pipe(
      map(res => res.data)
    );
  }

  getTemplateById(id: string): Observable<AdminTemplateItem | null> {
    return this.getTemplates('all').pipe(
      map(list => list.find(t => t.id === id) || null)
    );
  }

  updateOfficialTemplate(id: string, dto: CreateOfficialTemplateDTO): Observable<AdminTemplateItem> {
    return this.http.put<ApiResponse<AdminTemplateItem>>(`${this.apiUrl}/${id}`, dto).pipe(
      map(res => res.data)
    );
  }

  composeTemplatePreview(targetEnvironment: string, stackConfig: StackConfig): Observable<ComposedTemplatePlan> {
    return this.http.post<ApiResponse<ComposedTemplatePlan>>(`${this.apiUrl}/compose-preview`, {
      target_environment: targetEnvironment,
      stack_config: stackConfig
    }).pipe(
      map(res => res.data)
    );
  }

  getTechCatalog(layer?: ComponentLayer, includeInactive = false): Observable<TemplateComponent[]> {
    let params = new HttpParams();
    if (layer) {
      params = params.set('layer', layer);
    }
    if (includeInactive) {
      params = params.set('include_inactive', 'true');
    }
    return this.http.get<ApiResponse<TemplateComponent[]>>(this.componentsUrl, { params }).pipe(
      map(res => res.data || [])
    );
  }

  createTechCatalogItem(dto: CreateTemplateComponentDTO): Observable<TemplateComponent> {
    return this.http.post<ApiResponse<TemplateComponent>>(this.componentsUrl, dto).pipe(
      map(res => res.data)
    );
  }

  updateTechCatalogItem(id: string, dto: UpdateTemplateComponentDTO): Observable<TemplateComponent> {
    return this.http.put<ApiResponse<TemplateComponent>>(`${this.componentsUrl}/${id}`, dto).pipe(
      map(res => res.data)
    );
  }

  deleteTechCatalogItem(id: string): Observable<void> {
    return this.http.delete<ApiResponse<void>>(`${this.componentsUrl}/${id}`).pipe(
      map(() => void 0)
    );
  }
}
