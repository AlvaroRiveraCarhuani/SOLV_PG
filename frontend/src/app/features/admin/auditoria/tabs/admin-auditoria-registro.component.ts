import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { AdminAuditoriaService, AuditLogListResponse } from '../admin-auditoria.service';
import {
  AuditLog,
  EnrichedAction,
  StatusInfo,
  describeStatus,
  enrichAction,
  metadataReason
} from '@core/models/audit-log.model';
import { LucideActivity, LucidePanelRightOpen } from '@lucide/angular';

interface AuditRow {
  log: AuditLog;
  enriched: EnrichedAction;
  status: StatusInfo;
  reason: string | null;
}

const PAGE_SIZE = 20;

/**
 * Pestaña Registro de Auditoría (wireframe AUDIT_LOGS.md, ADR-027).
 * Tabla cronológica con enriquecimiento semántico, filtros, paginación y
 * off-canvas drawer con la cronología aislada del actor seleccionado.
 */
@Component({
  selector: 'admin-auditoria-registro',
  standalone: true,
  imports: [CommonModule, DateTextPipe, SearchBarComponent, PaginationBarComponent, ComboboxComponent, MachineDataDirective, DismissibleDirective, LucideActivity, LucidePanelRightOpen],
  templateUrl: './admin-auditoria-registro.component.html',
  styleUrl: './admin-auditoria-registro.component.scss'
})
export class AdminAuditoriaRegistroComponent implements OnInit {
  private readonly auditoriaService = inject(AdminAuditoriaService);

  readonly PAGE_SIZE = PAGE_SIZE;
  readonly isLoading = signal(true);
  readonly rows = signal<AuditRow[]>([]);
  readonly currentPage = signal(1);
  readonly totalPages = signal(1);
  readonly totalKnown = signal(0);

  readonly searchQuery = signal('');
  readonly actionFilter = signal('');

  readonly drawerOpen = signal(false);
  readonly drawerActorEmail = signal('');
  readonly drawerTimeline = signal<AuditRow[]>([]);
  readonly drawerLoading = signal(false);

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

  readonly actionOptions: ComboboxOption<string>[] = [
    { id: 'all-actions', value: '', label: 'Todas las acciones' },
    { id: 'create-action', value: 'POST', label: 'Creación' },
    { id: 'update-action', value: 'PUT', label: 'Actualización' },
    { id: 'delete-action', value: 'DELETE', label: 'Eliminación' }
  ];

  readonly selectedActionLabel = computed(() =>
    this.actionOptions.find((option) => option.value === this.actionFilter())?.label ?? this.actionOptions[0].label
  );

  ngOnInit(): void {
    this.load();
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
        this.rows.set(resp.data.map((log) => this.toRow(log)));
        this.totalKnown.set(resp.total);
        this.totalPages.set(Math.max(1, Math.ceil(resp.total / PAGE_SIZE)));
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
    this.drawerOpen.set(true);
    this.drawerActorEmail.set(row.log.actor_email);
    this.drawerTimeline.set([]);
    this.drawerLoading.set(true);

    this.auditoriaService.getActorTimeline(row.log.actor_id, 200).subscribe({
      next: (resp) => {
        this.drawerTimeline.set(resp.data.map((log) => this.toRow(log)));
        this.drawerLoading.set(false);
      },
      error: () => {
        this.drawerLoading.set(false);
      }
    });
  }

  closeDrawer(): void {
    this.drawerOpen.set(false);
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
      status: describeStatus(log.status_code, enriched.kind),
      reason: metadataReason(log)
    };
  }
}
