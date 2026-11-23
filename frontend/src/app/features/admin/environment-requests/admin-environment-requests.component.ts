import {
  Component,
  OnInit,
  OnDestroy,
  computed,
  inject,
  signal,
  viewChild,
  ElementRef,
  HostListener
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  LucideAlertCircle,
  LucideArrowRight,
  LucideCheck,
  LucideInbox,
  LucideMessageSquare,
  LucidePlus,
  LucideSearch
} from '@lucide/angular';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { DrawerComponent } from '@shared/components/drawer/drawer.component';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { environmentCovers } from '@shared/constants/environment-types.constants';
import { EnvironmentRequestService } from '@features/teacher/services/environment-request.service';
import {
  ENVIRONMENT_REQUEST_STATUS_LABEL,
  ENVIRONMENT_REQUEST_STATUS_TONE,
  EnvironmentMissingComponent,
  EnvironmentRequest,
  EnvironmentRequestComponent,
  EnvironmentRequestStatus,
  environmentLayerLabel
} from '@features/teacher/templates/environment-request.models';
import { AdminTemplatesService } from '../services/admin-templates.service';
import { AdminTemplateItem } from '../models/admin-templates.models';
import {
  buildEnvironmentRequestsTour,
  buildEnvironmentRequestsTutorial
} from './environment-requests-help.definitions';

const REQUESTS_HELP_KEY = 'environment-requests';

/**
 * Cola de solicitudes de entorno del docente.
 *
 * Muestra las solicitudes académicas en tabla estructurada con Drawer lateral
 * de resolución y permite vincular plantillas aprobadas o actualizar resoluciones previas.
 */
@Component({
  selector: 'admin-environment-requests',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    DateTextPipe,
    EnvironmentLabelPipe,
    SkeletonLoaderComponent,
    StatusTabsComponent,
    DrawerComponent,
    TechLogoComponent,
    LucideAlertCircle,
    LucideArrowRight,
    LucideCheck,
    LucideInbox,
    LucideMessageSquare,
    LucidePlus,
    LucideSearch
  ],
  templateUrl: './admin-environment-requests.component.html',
  styleUrl: './admin-environment-requests.component.scss'
})
export class AdminEnvironmentRequestsComponent implements OnInit, OnDestroy {
  private readonly requestService = inject(EnvironmentRequestService);
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly help = inject(ContextHelpService);
  private readonly productTour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

  readonly searchInput = viewChild<ElementRef<HTMLInputElement>>('searchInput');

  requests = signal<EnvironmentRequest[]>([]);
  templates = signal<AdminTemplateItem[]>([]);
  isLoading = signal<boolean>(true);
  activeTab = signal<string>('pending');
  searchQuery = signal<string>('');

  selected = signal<EnvironmentRequest | null>(null);
  selectedTemplateId = signal<string>('');
  reviewNote = signal<string>('');
  isSaving = signal<boolean>(false);
  error = signal<string | null>(null);
  notice = signal<string | null>(null);

  readonly pending = computed(() =>
    this.requests().filter(item => item.status === 'EN_REVISION')
  );
  readonly changes = computed(() =>
    this.requests().filter(item => item.status === 'NECESITA_CAMBIO')
  );
  readonly approved = computed(() =>
    this.requests().filter(item => item.status === 'APROBADA')
  );
  readonly rejected = computed(() =>
    this.requests().filter(item => item.status === 'NO_VIABLE')
  );

  readonly pendingCount = computed(() => this.pending().length);
  readonly changesCount = computed(() => this.changes().length);
  readonly approvedCount = computed(() => this.approved().length);
  readonly rejectedCount = computed(() => this.rejected().length);
  readonly allCount = computed(() => this.requests().length);

  readonly tabs = computed<StatusTabItem[]>(() => [
    { id: 'pending', label: 'En revisión', count: this.pendingCount(), badgeVariant: 'neutral' },
    { id: 'changes', label: 'Necesita cambio', count: this.changesCount(), badgeVariant: 'neutral' },
    { id: 'approved', label: 'Aprobadas', count: this.approvedCount(), badgeVariant: 'neutral' },
    { id: 'rejected', label: 'No viables', count: this.rejectedCount(), badgeVariant: 'neutral' },
    { id: 'all', label: 'Todas', count: this.allCount(), badgeVariant: 'neutral' }
  ]);

  readonly currentTabRequests = computed(() => {
    switch (this.activeTab()) {
      case 'pending':
        return this.pending();
      case 'changes':
        return this.changes();
      case 'approved':
        return this.approved();
      case 'rejected':
        return this.rejected();
      case 'all':
      default:
        return this.requests();
    }
  });

  readonly filteredRequests = computed(() => {
    const list = this.currentTabRequests();
    const query = this.searchQuery().trim().toLowerCase();
    if (!query) {
      return list;
    }
    return list.filter(item => {
      const matchTitle = (item.title || '').toLowerCase().includes(query);
      const matchRequester = (item.requester_name || '').toLowerCase().includes(query);
      const matchSubjects = (item.subjects || []).some(s =>
        (s.subject_name || '').toLowerCase().includes(query)
      );
      const matchComponents = (item.components || []).some(c =>
        (c.name || '').toLowerCase().includes(query)
      );
      const matchMissing = (item.missing_components || []).some(m =>
        (m.name || '').toLowerCase().includes(query)
      );
      return matchTitle || matchRequester || matchSubjects || matchComponents || matchMissing;
    });
  });

  readonly isDrawerOpen = computed(() => this.selected() !== null);

  readonly compatibleTemplates = computed(() => {
    const request = this.selected();
    if (!request) return [];
    return this.templates().filter(template =>
      environmentCovers(template.target_environment, request.target_environment)
    );
  });

  ngOnInit(): void {
    this.help.registerScreen(REQUESTS_HELP_KEY);
    this.help.registerGuides(REQUESTS_HELP_KEY, {
      tour: {
        label: 'Recorrido visual',
        hint: 'Señala cómo se lee una ficha y dónde se responde.',
        run: () => void this.productTour.start(buildEnvironmentRequestsTour())
      },
      tutorial: {
        label: 'Tutorial guiado',
        hint: 'Abra una solicitud y compruebe la plantilla antes de continuar.',
        run: () => this.startRequestsTutorial()
      }
    });
    this.load();
    this.templatesService.getTemplates('approved').subscribe({
      next: list => this.templates.set(list || []),
      error: () => this.templates.set([])
    });
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(REQUESTS_HELP_KEY);
    this.help.unregisterScreen(REQUESTS_HELP_KEY);
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboardShortcut(event: KeyboardEvent): void {
    if (event.key === '/' && !this.isDrawerOpen()) {
      const activeEl = document.activeElement;
      const isInput =
        activeEl instanceof HTMLInputElement ||
        activeEl instanceof HTMLTextAreaElement ||
        (activeEl as HTMLElement)?.isContentEditable;
      if (!isInput) {
        event.preventDefault();
        this.searchInput()?.nativeElement.focus();
      }
    }
  }

  load(): void {
    this.isLoading.set(true);
    this.requestService.listForReview().subscribe({
      next: list => {
        this.requests.set(list || []);
        this.isLoading.set(false);
      },
      error: () => {
        this.requests.set([]);
        this.isLoading.set(false);
      }
    });
  }

  onTabChange(tabId: string): void {
    this.activeTab.set(tabId);
  }

  onSearchChange(value: string): void {
    this.searchQuery.set(value);
  }

  clearSearch(): void {
    this.searchQuery.set('');
  }

  openDrawer(request: EnvironmentRequest): void {
    this.selected.set(request);
    this.selectedTemplateId.set(request.resolved_template_id || '');
    this.reviewNote.set('');
    this.error.set(null);
  }

  closeDrawer(): void {
    this.selected.set(null);
    this.error.set(null);
  }

  canReview(request: EnvironmentRequest): boolean {
    return request.status === 'EN_REVISION' || request.status === 'NECESITA_CAMBIO';
  }

  statusLabel(status: EnvironmentRequestStatus): string {
    return ENVIRONMENT_REQUEST_STATUS_LABEL[status] || status;
  }

  statusTone(status: EnvironmentRequestStatus): string {
    return ENVIRONMENT_REQUEST_STATUS_TONE[status] || 'neutral';
  }

  /** El docente elige tecnologías, no capas: se muestra la capa en español. */
  layerLabel(layer?: string | null): string {
    return environmentLayerLabel(layer);
  }

  actionButtonLabel(request: EnvironmentRequest): string {
    switch (request.status) {
      case 'EN_REVISION':
        return 'Responder';
      case 'NECESITA_CAMBIO':
        return 'Actualizar';
      case 'APROBADA':
      case 'NO_VIABLE':
      default:
        return 'Ver detalle';
    }
  }

  visibleComponents(request: EnvironmentRequest): EnvironmentRequestComponent[] {
    return (request.components || []).slice(0, 2);
  }

  extraComponentsCount(request: EnvironmentRequest): number {
    const total = (request.components || []).length;
    return total > 2 ? total - 2 : 0;
  }

  extraComponentsTooltip(request: EnvironmentRequest): string {
    const extra = (request.components || []).slice(2);
    return extra.map(c => c.name).join(', ');
  }

  visibleMissing(request: EnvironmentRequest): EnvironmentMissingComponent[] {
    return (request.missing_components || []).slice(0, 2);
  }

  extraMissingCount(request: EnvironmentRequest): number {
    const total = (request.missing_components || []).length;
    return total > 2 ? total - 2 : 0;
  }

  extraMissingTooltip(request: EnvironmentRequest): string {
    const extra = (request.missing_components || []).slice(2);
    return extra.map(m => m.name).join(', ');
  }

  hasAnyTech(request: EnvironmentRequest): boolean {
    return (request.components?.length || 0) > 0 || (request.missing_components?.length || 0) > 0;
  }

  subjectsTooltip(request: EnvironmentRequest): string {
    return (request.subjects || []).map(s => s.subject_name).join(', ');
  }

  emptyStateTitle(): string {
    if (this.searchQuery().trim()) {
      return 'Sin resultados para la búsqueda';
    }
    switch (this.activeTab()) {
      case 'pending':
        return 'No hay solicitudes en revisión';
      case 'changes':
        return 'No hay solicitudes con ajuste pendiente';
      case 'approved':
        return 'No hay solicitudes aprobadas';
      case 'rejected':
        return 'No hay solicitudes no viables';
      case 'all':
      default:
        return 'No hay solicitudes registradas';
    }
  }

  emptyStateDescription(): string {
    if (this.searchQuery().trim()) {
      return `No se encontraron solicitudes que coincidan con «${this.searchQuery().trim()}» en esta pestaña.`;
    }
    switch (this.activeTab()) {
      case 'pending':
        return 'Cuando un docente solicite un entorno que no existe en el catálogo, aparecerá aquí.';
      case 'changes':
        return 'Las solicitudes que requieran ajustes por parte del docente quedarán listadas aquí.';
      case 'approved':
        return 'Las solicitudes aprobadas y vinculadas a una plantilla aparecerán registradas aquí.';
      case 'rejected':
        return 'Las solicitudes marcadas como no viables aparecerán registradas en esta cola.';
      case 'all':
      default:
        return 'Las solicitudes de entornos enviadas por docentes se listarán en esta pantalla.';
    }
  }

  startRequestsTutorial(): void {
    this.tutorial.start(
      buildEnvironmentRequestsTutorial({
        reviewOpen: () => this.isDrawerOpen(),
        hasTemplate: () => this.selectedTemplateId() !== '',
        note: () => this.reviewNote()
      })
    );
  }

  approve(): void {
    if (!this.selectedTemplateId()) {
      this.error.set('Elija la plantilla que resuelve esta solicitud.');
      return;
    }
    this.submit({
      status: 'APROBADA',
      review_note: this.reviewNote(),
      resolved_template_id: this.selectedTemplateId()
    });
  }

  requestChanges(): void {
    if (this.reviewNote().trim().length < 5) {
      this.error.set('Explica qué falta para poder continuar (mínimo 5 caracteres).');
      return;
    }
    this.submit({ status: 'NECESITA_CAMBIO', review_note: this.reviewNote() });
  }

  reject(): void {
    if (this.reviewNote().trim().length < 5) {
      this.error.set('Explica por qué no es viable (mínimo 5 caracteres).');
      return;
    }
    this.submit({ status: 'NO_VIABLE', review_note: this.reviewNote() });
  }

  private submit(payload: {
    status: EnvironmentRequestStatus;
    review_note: string;
    resolved_template_id?: string;
  }): void {
    const request = this.selected();
    if (!request) {
      return;
    }
    this.isSaving.set(true);
    this.error.set(null);
    this.requestService.review(request.id, payload).subscribe({
      next: updated => {
        this.isSaving.set(false);
        this.requests.update(list => list.map(item => (item.id === updated.id ? updated : item)));
        this.notice.set(`Solicitud «${updated.title}» actualizada: ${this.statusLabel(updated.status)}.`);
        this.closeDrawer();
      },
      error: err => {
        this.isSaving.set(false);
        this.error.set(err?.error?.message || 'No pudimos registrar la respuesta.');
      }
    });
  }
}
