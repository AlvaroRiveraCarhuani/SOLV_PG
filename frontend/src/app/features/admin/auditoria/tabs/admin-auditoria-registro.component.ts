import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import {
  buildAuditRegistroTour,
  buildAuditRegistroTutorial
} from './audit-registro-help.definitions';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { AdminAuditoriaService, AuditLogListResponse } from '../admin-auditoria.service';
import {
  AuditLog,
  EnrichedAction,
  StatusInfo,
  auditActionLabel,
  auditResourceLabel,
  describeStatus,
  enrichAction,
  metadataReason
} from '@core/models/audit-log.model';
import { LucideActivity, LucideX } from '@lucide/angular';

export interface AuditRow {
  log: AuditLog;
  enriched: EnrichedAction;
  actionLabel: string;
  resourceLabel: string;
  status: StatusInfo;
  reason: string | null;
}

const PAGE_SIZE = 20;
const AUDIT_LOG_HELP_KEY = 'audit-log';

/**
 * Pestaña Registro de Auditoría (wireframe AUDIT_LOGS.md, ADR-027).
 * Tabla cronológica con enriquecimiento semántico, filtros, paginación y
 * off-canvas drawer con la ficha del evento seleccionado y cronología del actor.
 */
@Component({
  selector: 'admin-auditoria-registro',
  standalone: true,
  imports: [
    CommonModule,
    DateTextPipe,
    SearchBarComponent,
    PaginationBarComponent,
    DismissibleDirective,
    LucideActivity,
    LucideX
  ],
  templateUrl: './admin-auditoria-registro.component.html',
  styleUrl: './admin-auditoria-registro.component.scss'
})
export class AdminAuditoriaRegistroComponent implements OnInit, OnDestroy {
  private readonly auditoriaService = inject(AdminAuditoriaService);
  private readonly help = inject(ContextHelpService);
  private readonly productTour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

  readonly PAGE_SIZE = PAGE_SIZE;
  readonly isLoading = signal(true);
  readonly rows = signal<AuditRow[]>([]);
  readonly currentPage = signal(1);
  readonly totalPages = signal(1);
  readonly totalKnown = signal(0);

  readonly searchQuery = signal('');
  readonly actionFilter = signal('');

  readonly selectedRow = signal<AuditRow | null>(null);
  readonly drawerOpen = signal(false);
  readonly drawerActorEmail = signal('');
  readonly drawerTimeline = signal<AuditRow[]>([]);
  readonly drawerLoading = signal(false);

  readonly actionSegments = [
    { label: 'Todas', value: '' },
    { label: 'Creaciones', value: 'POST' },
    { label: 'Actualizaciones', value: 'PUT' },
    { label: 'Eliminaciones', value: 'DELETE' },
    { label: 'Emergencias', value: 'EMERGENCY_' }
  ];

  readonly hasActiveFilters = computed(() =>
    this.searchQuery().trim() !== '' || this.actionFilter() !== ''
  );

  /** Paginación estilo Docentes: rango visible "Mostrando X–Y de Z eventos" */
  readonly paginationDisplay = computed(() => {
    const total = this.totalKnown();
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.currentPage() - 1) * PAGE_SIZE + 1;
    const to = Math.min(this.currentPage() * PAGE_SIZE, total);
    return { from, to };
  });

  private searchDebounce: ReturnType<typeof setTimeout> | null = null;

  ngOnInit(): void {
    this.help.registerScreen(AUDIT_LOG_HELP_KEY);
    this.help.registerGuides(AUDIT_LOG_HELP_KEY, {
      tour: {
        label: 'Recorrido visual',
        hint: 'Señala filtros, tabla y cronología sin abrir eventos.',
        run: () => void this.productTour.start(buildAuditRegistroTour())
      },
      tutorial: {
        label: 'Tutorial guiado',
        hint: 'Aplica un filtro y abre la cronología de un actor.',
        run: () => this.startAuditTutorial()
      }
    });
    this.load();
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(AUDIT_LOG_HELP_KEY);
    this.help.unregisterScreen(AUDIT_LOG_HELP_KEY);
  }

  /** Tutorial de investigación: exige un filtro aplicado y una cronología abierta. */
  startAuditTutorial(): void {
    this.tutorial.start(
      buildAuditRegistroTutorial({
        hasActiveFilters: () => this.hasActiveFilters(),
        drawerOpen: () => this.drawerOpen()
      })
    );
  }

  load(): void {
    this.isLoading.set(true);
    const search = this.searchQuery().trim();
    const filters: { action?: string; search?: string } = {};
    if (this.actionFilter()) {
      filters.action = this.actionFilter();
    }
    if (search) {
      filters.search = search;
    }

    this.auditoriaService.listAuditLogs(this.currentPage(), PAGE_SIZE, filters).subscribe({
      next: (resp: AuditLogListResponse) => {
        const data = resp?.data ?? [];
        this.rows.set(data.map((log) => this.toRow(log)));
        this.totalKnown.set(resp?.total ?? 0);
        this.totalPages.set(Math.max(1, Math.ceil((resp?.total ?? 0) / PAGE_SIZE)));
        this.isLoading.set(false);
      },
      error: () => {
        this.rows.set([]);
        this.isLoading.set(false);
      }
    });
  }

  onSearchChange(value: string): void {
    this.searchQuery.set(value);
    if (this.searchDebounce) {
      clearTimeout(this.searchDebounce);
    }
    this.searchDebounce = setTimeout(() => {
      this.currentPage.set(1);
      this.load();
    }, 250);
  }

  onActionFilterChange(value: string): void {
    this.actionFilter.set(value);
    this.currentPage.set(1);
    this.load();
  }

  onPageChange(page: number): void {
    this.currentPage.set(page);
    this.load();
  }

  clearFilters(): void {
    this.searchQuery.set('');
    this.actionFilter.set('');
    this.currentPage.set(1);
    this.load();
  }

  openDrawer(row: AuditRow): void {
    this.selectedRow.set(row);
    this.drawerOpen.set(true);
    this.drawerActorEmail.set(row.log.actor_email);
    this.drawerTimeline.set([]);
    this.drawerLoading.set(true);

    this.auditoriaService.getActorTimeline(row.log.actor_id, 200).subscribe({
      next: (resp) => {
        const data = resp?.data ?? [];
        this.drawerTimeline.set(data.map((log) => this.toRow(log)));
        this.drawerLoading.set(false);
      },
      error: () => {
        this.drawerLoading.set(false);
      }
    });
  }

  closeDrawer(): void {
    this.drawerOpen.set(false);
    this.selectedRow.set(null);
  }

  onDrawerBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('drawer-backdrop')) {
      this.closeDrawer();
    }
  }

  private toRow(log: AuditLog): AuditRow {
    const enriched = enrichAction(log.action);
    return {
      log,
      enriched,
      actionLabel: enriched.label,
      resourceLabel: auditResourceLabel(log.resource_type),
      status: describeStatus(log.status_code),
      reason: metadataReason(log)
    };
  }
}
