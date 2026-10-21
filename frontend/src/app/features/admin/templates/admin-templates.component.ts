import { Component, OnInit, signal, computed, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ActivatedRoute, Router } from '@angular/router';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { 
  AdminTemplatesService, 
  AdminTemplateItem, 
  CreateOfficialTemplateDTO,
  ReviewTemplateDTO, 
  TemplateCacheStatus
} from '../services/admin-templates.service';
import { TemplateReviewModalComponent } from './components/template-review-modal/template-review-modal.component';
import { TemplateRejectModalComponent } from './components/template-reject-modal/template-reject-modal.component';
import { TemplateEditModalComponent } from './components/template-edit-modal/template-edit-modal.component';
import { TemplateStatusModalComponent } from './components/template-status-modal/template-status-modal.component';
import { TemplatePromoteModalComponent } from './components/template-promote-modal/template-promote-modal.component';
import { 
  LucideLayers, 
  LucideBox, 
  LucideRefreshCw, 
  LucidePlus, 
  LucideClock, 
  LucideCheckCircle2, 
  LucidePause, 
  LucidePlay, 
  LucideAlertCircle,
  LucidePencil,
  LucideHardDrive,
  LucideCopy,
  LucideCheck,
  LucideChevronDown,
  LucideBadgeCheck,
  LucideMoreVertical,
  LucideDownloadCloud
} from '@lucide/angular';
import { CURATED_TECH_ICONS, TechIconKey, TechLogoComponent } from './components/tech-logo/tech-logo.component';
import { SearchBarComponent } from '../../../shared/components/search-bar/search-bar.component';
import { KpiCardComponent, KpiGridComponent } from '../../../shared/components/kpi-card/kpi-card.component';
import { PaginationBarComponent } from '../../../shared/components/pagination-bar/pagination-bar.component';
import { ViewSwitcherComponent } from '../../../shared/components/view-switcher/view-switcher.component';
import { StatusTabsComponent, StatusTabItem } from '../../../shared/components/status-tabs/status-tabs.component';

export interface TechMeta {
  id: string;
  name: string;
  badgeClass: string;
  iconType: TechIconKey;
}

export function detectTemplateTech(item: Pick<AdminTemplateItem, 'name' | 'docker_image' | 'stack_config'>): TechMeta {
  const runtimeId = (item.stack_config?.runtime_id || '').toLowerCase().trim();
  const webId = (item.stack_config?.web_platform_id || '').toLowerCase().trim();

  const matchCuratedId = (id: string): TechMeta | null => {
    if (!id) return null;
    const direct = CURATED_TECH_ICONS.find(c => c.key.toLowerCase() === id && c.key !== 'docker');
    if (direct) {
      return { id: direct.key, name: direct.label, badgeClass: `tech-${direct.key}`, iconType: direct.key as TechIconKey };
    }
    for (const c of CURATED_TECH_ICONS) {
      if (c.key === 'docker') continue;
      const key = c.key.toLowerCase();
      const re = new RegExp(`(^|[^a-z0-9])${key}([^a-z0-9]|$)`, 'i');
      if (re.test(id)) {
        return { id: c.key, name: c.label, badgeClass: `tech-${c.key}`, iconType: c.key as TechIconKey };
      }
    }
    return null;
  };

  if (runtimeId) {
    const matched = matchCuratedId(runtimeId);
    if (matched) return matched;
  }

  if (webId) {
    const matched = matchCuratedId(webId);
    if (matched) return matched;
  }

  const name = item.name || '';
  const image = item.docker_image || '';
  const text = `${name} ${image}`.toLowerCase();

  if (text.includes('python') || text.includes('django') || text.includes('flask') || text.includes('fastapi') || text.includes('pandas') || text.includes('numpy') || text.includes('streamlit') || text.includes(' ds')) {
    return { id: 'python', name: 'Python', badgeClass: 'tech-python', iconType: 'python' };
  }
  if (text.includes('rust') || text.includes('cargo') || text.includes('actix') || text.includes('tokio')) {
    return { id: 'rust', name: 'Rust', badgeClass: 'tech-rust', iconType: 'rust' };
  }
  if (text.includes('typescript') || text.includes('ts-node')) {
    return { id: 'typescript', name: 'TypeScript', badgeClass: 'tech-typescript', iconType: 'typescript' };
  }
  if (text.includes('angular')) {
    return { id: 'angular', name: 'Angular', badgeClass: 'tech-angular', iconType: 'angular' };
  }
  if (text.includes('react') || text.includes('next')) {
    return { id: 'react', name: 'React', badgeClass: 'tech-react', iconType: 'react' };
  }
  if (text.includes('vue') || text.includes('vite')) {
    return { id: 'vue', name: 'Vue', badgeClass: 'tech-vue', iconType: 'vue' };
  }
  if (text.includes('node') || text.includes('javascript') || text.includes('express') || text.includes('nest') || text.includes('bun') || text.includes('deno')) {
    return { id: 'node', name: 'Node.js', badgeClass: 'tech-node', iconType: 'node' };
  }
  if (text.includes('php') || text.includes('laravel')) {
    return { id: 'php', name: 'PHP', badgeClass: 'tech-php', iconType: 'php' };
  }
  if (text.includes('dotnet') || text.includes('c#') || text.includes('.net') || text.includes('csharp')) {
    return { id: 'dotnet', name: '.NET / C#', badgeClass: 'tech-dotnet', iconType: 'dotnet' };
  }
  if (text.includes('golang') || text.includes('go:') || text.includes('go-') || text.includes('/go') || text.includes('gin') || text.includes('fiber') || /\bgo\b/.test(text)) {
    return { id: 'go', name: 'Go', badgeClass: 'tech-go', iconType: 'go' };
  }
  if (text.includes('java') || text.includes('spring') || text.includes('kotlin') || text.includes('maven') || text.includes('gradle')) {
    return { id: 'java', name: 'Java', badgeClass: 'tech-java', iconType: 'java' };
  }
  if (text.includes('c++') || text.includes('cpp') || text.includes('clang') || text.includes('gcc') || text.includes('g++') || text.includes('cmake')) {
    return { id: 'cpp', name: 'C / C++', badgeClass: 'tech-cpp', iconType: 'cpp' };
  }
  if (text.includes('postgres') || text.includes('mysql') || text.includes('sql') || text.includes('redis') || text.includes('mongo') || text.includes('mariadb')) {
    return { id: 'db', name: 'Bases de Datos', badgeClass: 'tech-db', iconType: 'db' };
  }

  for (const curated of CURATED_TECH_ICONS) {
    if (curated.key === 'docker' || curated.key === 'go') continue;
    if (text.includes(curated.key)) {
      return { id: curated.key, name: curated.label, badgeClass: `tech-${curated.key}`, iconType: curated.key as TechIconKey };
    }
  }

  return { id: 'docker', name: 'Docker Base', badgeClass: 'tech-docker', iconType: 'docker' };
}

export function detectTech(name: string, image: string): TechMeta {
  return detectTemplateTech({ name, docker_image: image });
}

export type TabType = 'pending' | 'catalog' | 'rejected';
export type ViewMode = 'cards' | 'table';

export interface ToastNotification {
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
}

@Component({
  selector: 'admin-templates',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    DateTextPipe,
    TemplateReviewModalComponent,
    TemplateRejectModalComponent,
    TemplateEditModalComponent,
    TemplateStatusModalComponent,
    TemplatePromoteModalComponent,
    TechLogoComponent,
    SearchBarComponent,
    KpiCardComponent,
    KpiGridComponent,
    PaginationBarComponent,
    ViewSwitcherComponent,
    StatusTabsComponent,
    LucideLayers,
    LucideBox,
    LucideRefreshCw,
    LucidePlus,
    LucideClock,
    LucideCheckCircle2,
    LucidePause,
    LucidePlay,
    LucideAlertCircle,
    LucidePencil,
    LucideHardDrive,
    LucideCopy,
    LucideCheck,
    LucideChevronDown,
    LucideBadgeCheck,
    LucideMoreVertical,
    LucideDownloadCloud
  ],
  templateUrl: './admin-templates.component.html',
  styleUrls: ['./admin-templates.component.scss']
})
export class AdminTemplatesComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // States
  activeTab = signal<TabType>('pending');
  viewMode = signal<ViewMode>('cards');
  searchTerm = signal<string>('');
  selectedTech = signal<string>('all');
  selectedPurpose = signal<'ALL' | 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS'>('ALL');
  copiedTag = signal<string | null>(null);
  isLoading = signal<boolean>(false);
  openMenuId = signal<string | null>(null);

  isPausedStatus(status?: string): boolean {
    return status === 'paused' || status === 'suspended' || status === 'SUSPENDIDA';
  }

  // Cache warming state
  cacheStatuses = signal<Map<string, TemplateCacheStatus>>(new Map());
  isWarmingAll = signal<boolean>(false);
  warmingTemplateIds = signal<Set<string>>(new Set());

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.row-menu-anchor')) {
      this.openMenuId.set(null);
    }
  }

  toggleRowMenu(id: string, event: MouseEvent): void {
    event.stopPropagation();
    this.openMenuId.update((current) => (current === id ? null : id));
  }

  // Paginación Catálogo
  catalogPage = signal<number>(1);
  pageSize = signal<number>(8);

  // Paginación Rechazadas
  rejectedPage = signal<number>(1);
  rejectedPageSize = signal<number>(8);

  // Data
  allTemplates = signal<AdminTemplateItem[]>([]);

  // Modals state
  selectedTemplateForReview = signal<AdminTemplateItem | null>(null);
  selectedTemplateForReject = signal<AdminTemplateItem | null>(null);
  selectedTemplateForEdit = signal<AdminTemplateItem | null>(null);
  templateToToggleStatus = signal<AdminTemplateItem | null>(null);

  // Toast feedback
  toast = signal<ToastNotification | null>(null);

  // Plantillas base de la pestaña activa (para calcular chips contextuales)
  currentTabTemplates = computed(() => {
    const tab = this.activeTab();
    const all = this.allTemplates();
    if (tab === 'pending') {
      return all.filter(t => t.status === 'pending' || t.status === 'PENDIENTE_AUDITORIA');
    }
    if (tab === 'catalog') {
      return all.filter(t => t.status === 'approved' || t.status === 'APROBADA' || this.isPausedStatus(t.status));
    }
    return all.filter(t => t.status === 'rejected' || t.status === 'RECHAZADA');
  });

  // Tecnologías disponibles dinámicas (calculadas reactivamente según la pestaña activa)
  availableTechs = computed(() => {
    const counts: Record<string, { meta: TechMeta; count: number }> = {};
    for (const t of this.currentTabTemplates()) {
      const meta = detectTemplateTech(t);
      if (!counts[meta.id]) {
        counts[meta.id] = { meta, count: 0 };
      }
      counts[meta.id].count++;
    }
    return Object.values(counts).sort((a, b) => b.count - a.count);
  });

  // Top 4 tecnologías más usadas + extras para agrupar en dropdown
  topTechs = computed(() => {
    return this.availableTechs().slice(0, 4);
  });

  extraTechs = computed(() => {
    return this.availableTechs().slice(4);
  });

  isExtraSelected = computed(() => {
    const selected = this.selectedTech();
    if (selected === 'all') return false;
    return this.extraTechs().some(t => t.meta.id === selected);
  });

  // Computed lists con filtro de búsqueda, propósito y tecnología
  pendingTemplates = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const tech = this.selectedTech();
    const purpose = this.selectedPurpose();
    return this.allTemplates().filter(t => {
      const isPending = t.status === 'pending' || t.status === 'PENDIENTE_AUDITORIA';
      if (!isPending) return false;
      if (purpose !== 'ALL') {
        const env = t.target_environment || 'IDE_PERSISTENTE';
        if (purpose === 'IDE_PERSISTENTE' && env !== 'IDE_PERSISTENTE') return false;
        if (purpose === 'JUEZ_EFIMERO' && env !== 'JUEZ_EFIMERO' && env !== 'JUEZ_VIRTUAL') return false;
        if (purpose === 'AMBOS' && env !== 'AMBOS') return false;
      }
      if (tech !== 'all') {
        const itemTech = detectTemplateTech(t);
        if (itemTech.id !== tech) return false;
      }
      if (!term) return true;
      return (
        t.name.toLowerCase().includes(term) ||
        t.docker_image.toLowerCase().includes(term) ||
        (t.requested_by_name && t.requested_by_name.toLowerCase().includes(term)) ||
        (t.description && t.description.toLowerCase().includes(term))
      );
    });
  });

  catalogTemplates = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const tech = this.selectedTech();
    const purpose = this.selectedPurpose();
    return this.allTemplates().filter(t => {
      const isCatalog = t.status === 'approved' || t.status === 'APROBADA' || this.isPausedStatus(t.status);
      if (!isCatalog) return false;
      if (purpose !== 'ALL') {
        const env = t.target_environment || 'IDE_PERSISTENTE';
        if (purpose === 'IDE_PERSISTENTE' && env !== 'IDE_PERSISTENTE') return false;
        if (purpose === 'JUEZ_EFIMERO' && env !== 'JUEZ_EFIMERO' && env !== 'JUEZ_VIRTUAL') return false;
        if (purpose === 'AMBOS' && env !== 'AMBOS') return false;
      }
      if (tech !== 'all') {
        const itemTech = detectTemplateTech(t);
        if (itemTech.id !== tech) return false;
      }
      if (!term) return true;
      return (
        t.name.toLowerCase().includes(term) ||
        t.docker_image.toLowerCase().includes(term) ||
        (t.description && t.description.toLowerCase().includes(term))
      );
    });
  });

  totalCatalogPages = computed(() => {
    const total = this.catalogTemplates().length;
    return Math.max(1, Math.ceil(total / this.pageSize()));
  });

  paginatedCatalogTemplates = computed(() => {
    const list = this.catalogTemplates();
    const page = this.catalogPage();
    const size = this.pageSize();
    const start = (page - 1) * size;
    return list.slice(start, start + size);
  });

  rejectedTemplates = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const tech = this.selectedTech();
    const purpose = this.selectedPurpose();
    return this.allTemplates().filter(t => {
      const isRejected = t.status === 'rejected' || t.status === 'RECHAZADA';
      if (!isRejected) return false;
      if (purpose !== 'ALL') {
        const env = t.target_environment || 'IDE_PERSISTENTE';
        if (purpose === 'IDE_PERSISTENTE' && env !== 'IDE_PERSISTENTE') return false;
        if (purpose === 'JUEZ_EFIMERO' && env !== 'JUEZ_EFIMERO' && env !== 'JUEZ_VIRTUAL') return false;
        if (purpose === 'AMBOS' && env !== 'AMBOS') return false;
      }
      if (tech !== 'all') {
        const itemTech = detectTemplateTech(t);
        if (itemTech.id !== tech) return false;
      }
      if (!term) return true;
      return (
        t.name.toLowerCase().includes(term) ||
        t.docker_image.toLowerCase().includes(term) ||
        (t.requested_by_name && t.requested_by_name.toLowerCase().includes(term)) ||
        (t.rejection_reason && t.rejection_reason.toLowerCase().includes(term)) ||
        (t.description && t.description.toLowerCase().includes(term))
      );
    });
  });

  totalRejectedPages = computed(() => {
    const total = this.rejectedTemplates().length;
    return Math.max(1, Math.ceil(total / this.rejectedPageSize()));
  });

  paginatedRejectedTemplates = computed(() => {
    const list = this.rejectedTemplates();
    const page = this.rejectedPage();
    const size = this.rejectedPageSize();
    const start = (page - 1) * size;
    return list.slice(start, start + size);
  });

  pendingCount = computed(() => {
    return this.allTemplates().filter(t => t.status === 'pending' || t.status === 'PENDIENTE_AUDITORIA').length;
  });

  catalogCount = computed(() => {
    return this.allTemplates().filter(t => t.status === 'approved' || t.status === 'APROBADA' || this.isPausedStatus(t.status)).length;
  });

  activeCount = computed(() => {
    return this.allTemplates().filter(t => t.status === 'approved' || t.status === 'APROBADA').length;
  });

  pausedCount = computed(() => {
    return this.allTemplates().filter(t => this.isPausedStatus(t.status)).length;
  });

  rejectedCount = computed(() => {
    return this.allTemplates().filter(t => t.status === 'rejected' || t.status === 'RECHAZADA').length;
  });

  templateTabs = computed<StatusTabItem[]>(() => [
    {
      id: 'pending',
      label: 'Solicitudes Pendientes',
      count: this.pendingCount(),
      badgeVariant: this.pendingCount() > 0 ? 'warning' : 'default'
    },
    {
      id: 'catalog',
      label: 'Catálogo Activo',
      count: this.catalogCount(),
      badgeVariant: 'neutral'
    },
    {
      id: 'rejected',
      label: 'Rechazadas',
      count: this.rejectedCount(),
      badgeVariant: this.rejectedCount() > 0 ? 'error' : 'default'
    }
  ]);

  onTabChange(tabId: string): void {
    this.setTab(tabId as TabType);
  }

  averageRam = computed(() => {
    const activeCatalog = this.allTemplates().filter(t => t.status === 'approved' || t.status === 'APROBADA');
    if (activeCatalog.length === 0) return 512;
    const total = activeCatalog.reduce((acc, t) => acc + (t.base_ram_mb || 512), 0);
    return Math.round(total / activeCatalog.length);
  });

  ngOnInit(): void {
    const tabParam = this.route.snapshot.queryParamMap.get('tab');
    if (tabParam === 'catalog' || tabParam === 'pending' || tabParam === 'rejected') {
      this.activeTab.set(tabParam);
    }
    this.loadTemplates();
    if (this.route.snapshot.queryParamMap.get('nueva') === '1') {
      this.navigateToCreateStudio();
    }
  }

  loadTemplates(): void {
    this.isLoading.set(true);
    this.templatesService.getTemplates('all').subscribe({
      next: (data) => {
        this.allTemplates.set(data);
        const requestedTab = this.route.snapshot.queryParamMap.get('tab');
        if (!requestedTab && this.activeTab() === 'pending' && this.pendingCount() === 0 && this.catalogCount() > 0) {
          this.activeTab.set('catalog');
        }
        this.isLoading.set(false);
        this.loadCacheStatuses();
      },
      error: () => {
        this.isLoading.set(false);
        this.showToast('Error al cargar las plantillas de entornos.', 'error');
      }
    });
  }

  setTab(tab: TabType): void {
    this.activeTab.set(tab);
    this.selectedTech.set('all');
    this.catalogPage.set(1);
    this.rejectedPage.set(1);
  }

  setViewMode(mode: ViewMode): void {
    this.viewMode.set(mode);
  }

  openReviewModal(template: AdminTemplateItem): void {
    this.selectedTemplateForReview.set(template);
  }

  closeReviewModal(): void {
    this.selectedTemplateForReview.set(null);
  }

  openRejectModal(template: AdminTemplateItem): void {
    this.selectedTemplateForReject.set(template);
  }

  closeRejectModal(): void {
    this.selectedTemplateForReject.set(null);
  }

  navigateToCreateStudio(): void {
    this.router.navigate(['/admin/plantillas/nueva']);
  }

  navigateToEditStudio(template: AdminTemplateItem): void {
    this.router.navigate(['/admin/plantillas', template.id, 'editar']);
  }

  handleApprove(event: { id: string; base_ram_mb: number }): void {
    const dto: ReviewTemplateDTO = {
      status: 'pending_audit',
      base_ram_mb: event.base_ram_mb
    };

    this.templatesService.reviewTemplate(event.id, dto).subscribe({
      next: (updated) => {
        this.allTemplates.update(list => 
          list.map(t => t.id === updated.id ? updated : t)
        );
        this.closeReviewModal();
        this.showToast(`Solicitud de plantilla "${updated.name}" aprobada y derivada a borrador en auditoría técnica.`, 'success');
      },
      error: (err) => {
        const msg = err.error?.message || 'No se pudo aprobar la plantilla.';
        this.showToast(msg, 'error');
        this.closeReviewModal();
      }
    });
  }

  templateToPromote = signal<AdminTemplateItem | null>(null);

  openPromoteModal(template: AdminTemplateItem): void {
    if (template.status !== 'approved' && template.status !== 'APROBADA') {
      this.showToast('Solo plantillas aprobadas pueden promoverse a modelo institucional.', 'error');
      return;
    }
    this.templateToPromote.set(template);
  }

  closePromoteModal(): void {
    this.templateToPromote.set(null);
  }

  handlePromoted(_event: { id: string; name: string }): void {
    this.closePromoteModal();
    const msg = $localize`:@@ST-16:Plantilla promovida a modelo institucional.`;
    this.showToast(msg, 'success');
  }

  promoteToModel(template: AdminTemplateItem): void {
    this.openPromoteModal(template);
  }

  handleReject(event: { id: string; reason: string }): void {
    const dto: ReviewTemplateDTO = {
      status: 'rejected',
      rejection_reason: event.reason
    };

    this.templatesService.reviewTemplate(event.id, dto).subscribe({
      next: (updated) => {
        this.allTemplates.update(list => 
          list.map(t => t.id === updated.id ? updated : t)
        );
        this.closeRejectModal();
        this.showToast(`Solicitud rechazada con motivo registrado para el docente.`, 'success');
      },
      error: (err) => {
        const msg = err.error?.message || 'No se pudo registrar el rechazo de la plantilla.';
        this.showToast(msg, 'error');
        this.closeRejectModal();
      }
    });
  }


  toggleStatusReason = signal<string>('');

  requestToggleStatus(template: AdminTemplateItem): void {
    this.toggleStatusReason.set('');
    this.templateToToggleStatus.set(template);
  }

  cancelToggleStatus(): void {
    this.templateToToggleStatus.set(null);
    this.toggleStatusReason.set('');
  }

  handleConfirmToggleStatus(event: { template: AdminTemplateItem; reason: string }): void {
    const { template, reason } = event;
    const isPausing = template.status === 'approved' || template.status === 'APROBADA';
    const nextStatus = isPausing ? 'paused' : 'approved';
    const dto: ReviewTemplateDTO = {
      status: nextStatus,
      rejection_reason: isPausing ? reason : undefined
    };

    this.templatesService.reviewTemplate(template.id, dto).subscribe({
      next: (updated) => {
        this.allTemplates.update(list => 
          list.map(t => t.id === updated.id ? updated : t)
        );
        this.templateToToggleStatus.set(null);
        this.toggleStatusReason.set('');
        const actionText = nextStatus === 'paused' ? 'suspendida/pausada' : 'reactivada';
        this.showToast(`Plantilla "${template.name}" ${actionText} en el catálogo.`, 'success');
      },
      error: () => {
        this.templateToToggleStatus.set(null);
        this.toggleStatusReason.set('');
        this.showToast('No se pudo modificar el estado de la plantilla.', 'error');
      }
    });
  }

  confirmToggleStatus(template: AdminTemplateItem): void {
    const reason = this.toggleStatusReason().trim();
    const isPausing = template.status === 'approved' || template.status === 'APROBADA';
    if (isPausing && reason.length < 10) {
      return;
    }
    this.handleConfirmToggleStatus({ template, reason });
  }

  duplicateTemplate(template: AdminTemplateItem): void {
    this.isLoading.set(true);
    this.templatesService.duplicateTemplate(template.id).subscribe({
      next: (duplicated) => {
        this.allTemplates.update(list => [duplicated, ...list]);
        this.isLoading.set(false);
        this.showToast(`Plantilla duplicada como "${duplicated.name}" y encolada para auditoría.`, 'success');
      },
      error: (err) => {
        this.isLoading.set(false);
        const msg = err.error?.message || err.error?.error || 'No se pudo duplicar la plantilla.';
        this.showToast(msg, 'error');
      }
    });
  }

  openEditModal(template: AdminTemplateItem): void {
    this.selectedTemplateForEdit.set(template);
  }

  closeEditModal(): void {
    this.selectedTemplateForEdit.set(null);
  }

  handleSaveEdit(event: { id: string; base_ram_mb: number; description?: string }): void {
    const current = this.selectedTemplateForEdit();
    if (!current) return;

    const updateDto: CreateOfficialTemplateDTO = {
      name: current.name,
      docker_image: current.docker_image,
      base_ram_mb: event.base_ram_mb,
      description: event.description !== undefined ? event.description : (current.description || ''),
      target_environment: current.target_environment || 'IDE_PERSISTENTE',
      category_id: current.category_id || undefined,
      entrypoint: current.entrypoint || undefined,
      timeout_ms: current.timeout_ms || undefined,
      sample_input: current.sample_input || undefined,
      services_config: current.services_config,
      setup_script: current.setup_script || undefined,
      tools_declared: current.tools_declared,
      stack_config: current.stack_config
    };

    const targetStatus: 'approved' | 'paused' = this.isPausedStatus(current.status) ? 'paused' : 'approved';
    const fallbackDto: ReviewTemplateDTO = {
      status: targetStatus,
      base_ram_mb: event.base_ram_mb
    };

    const save$ = (typeof this.templatesService.updateOfficialTemplate === 'function')
      ? this.templatesService.updateOfficialTemplate(event.id, updateDto)
      : this.templatesService.reviewTemplate(event.id, fallbackDto);

    save$.subscribe({
      next: (updated) => {
        if (event.description !== undefined) {
          updated.description = event.description;
        }
        this.allTemplates.update(list => 
          list.map(t => t.id === updated.id ? updated : t)
        );
        this.closeEditModal();
        this.showToast(`Cuotas de "${updated.name}" actualizadas correctamente.`, 'success');
      },
      error: () => {
        this.closeEditModal();
        this.showToast('No se pudieron actualizar las cuotas de la plantilla.', 'error');
      }
    });
  }

  getTemplateTech(item: AdminTemplateItem): TechMeta {
    return detectTemplateTech(item);
  }

  getTech(name: string, image: string): TechMeta {
    return detectTech(name, image);
  }

  getPurposeBadge(targetEnv?: string): { label: string; shortLabel: string; variant: 'ide' | 'judge' | 'dual' } {
    if (targetEnv === 'AMBOS') {
      return { label: 'Dual: Lab + Juez', shortLabel: 'DUAL', variant: 'dual' };
    }
    if (targetEnv === 'JUEZ_EFIMERO' || targetEnv === 'JUEZ_VIRTUAL') {
      return { label: 'Evaluador Juez', shortLabel: 'JUEZ', variant: 'judge' };
    }
    return { label: 'Laboratorio IDE', shortLabel: 'LAB IDE', variant: 'ide' };
  }

  setTechFilter(techId: string): void {
    this.selectedTech.set(techId);
    this.catalogPage.set(1);
    this.rejectedPage.set(1);
  }

  onSelectExtraTech(event: Event): void {
    const target = event.target as HTMLSelectElement;
    if (target && target.value) {
      this.setTechFilter(target.value);
    }
  }

  goToCatalogPage(page: number): void {
    if (page >= 1 && page <= this.totalCatalogPages()) {
      this.catalogPage.set(page);
    }
  }

  goToRejectedPage(page: number): void {
    if (page >= 1 && page <= this.totalRejectedPages()) {
      this.rejectedPage.set(page);
    }
  }

  copyDockerTag(tag: string, event?: Event): void {
    if (event) {
      event.stopPropagation();
    }
    navigator.clipboard.writeText(tag).then(() => {
      this.copiedTag.set(tag);
      setTimeout(() => {
        if (this.copiedTag() === tag) {
          this.copiedTag.set(null);
        }
      }, 2000);
    });
  }

  loadCacheStatuses(): void {
    if (!this.templatesService.getTemplatesCacheStatus) {
      return;
    }
    this.templatesService.getTemplatesCacheStatus().subscribe({
      next: (statuses) => {
        const map = new Map<string, TemplateCacheStatus>();
        for (const s of statuses) {
          map.set(s.template_id, s);
        }
        this.cacheStatuses.set(map);
      },
      error: () => {}
    });
  }

  getCacheStatus(templateId: string): TemplateCacheStatus | undefined {
    return this.cacheStatuses().get(templateId);
  }

  formatBytes(bytes: number): string {
    if (!bytes || bytes <= 0) return '0 B';
    const k = 1024;
    const dm = 1;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
  }

  warmTemplate(template: AdminTemplateItem, event?: MouseEvent): void {
    if (event) {
      event.stopPropagation();
    }
    const id = template.id;
    this.warmingTemplateIds.update(set => {
      const next = new Set(set);
      next.add(id);
      return next;
    });

    this.templatesService.warmTemplate(id).subscribe({
      next: (status) => {
        this.warmingTemplateIds.update(set => {
          const next = new Set(set);
          next.delete(id);
          return next;
        });
        this.cacheStatuses.update(map => {
          const next = new Map(map);
          next.set(id, status);
          return next;
        });
        this.showToast(`Entorno listo en el servidor: ${template.name}`, 'success');
      },
      error: () => {
        this.warmingTemplateIds.update(set => {
          const next = new Set(set);
          next.delete(id);
          return next;
        });
        this.showToast(`No se pudo preparar el entorno de ${template.name}`, 'error');
      }
    });
  }

  warmAllTemplates(): void {
    if (this.activeCount() === 0) {
      this.showToast('No hay plantillas activas en el catálogo para preparar.', 'success');
      return;
    }
    this.isWarmingAll.set(true);
    this.templatesService.warmAllTemplates().subscribe({
      next: (summary) => {
        this.isWarmingAll.set(false);
        this.loadCacheStatuses();
        if (summary.total_checked === 0) {
          this.showToast('No hay entornos activos para preparar en el catálogo.', 'success');
        } else {
          this.showToast(
            `Entornos preparados: ${summary.warmed_count} descargados, ${summary.already_warm} ya listos`,
            'success'
          );
        }
      },
      error: () => {
        this.isWarmingAll.set(false);
        this.showToast('No se pudieron preparar los entornos del servidor', 'error');
      }
    });
  }

  showToast(message: string, type: 'success' | 'error' | 'info' = 'success'): void {
    this.toast.set({ message, type });
    setTimeout(() => {
      this.toast.set(null);
    }, 4000);
  }
}
