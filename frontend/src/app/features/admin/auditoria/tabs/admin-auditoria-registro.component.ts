import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { AdminAuditoriaService, AuditLogListResponse } from '../admin-auditoria.service';
import {
  AuditLog,
  EnrichedAction,
  StatusInfo,
  describeStatus,
  enrichAction,
  metadataReason
} from '@core/models/audit-log.model';

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
  imports: [CommonModule, FormsModule, SearchBarComponent, PaginationBarComponent],
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
  readonly actorFilter = signal('');

  readonly drawerOpen = signal(false);
  readonly drawerActorEmail = signal('');
  readonly drawerTimeline = signal<AuditRow[]>([]);
  readonly drawerLoading = signal(false);

  readonly hasActiveFilters = computed(() =>
    this.searchQuery().trim() !== '' || this.actionFilter() !== '' || this.actorFilter() !== ''
  );

  private searchDebounce: ReturnType<typeof setTimeout> | null = null;

  readonly actionOptions = [
    { value: '', label: 'Todas las acciones' },
    { value: 'POST', label: 'Creación' },
    { value: 'PUT', label: 'Actualización' },
    { value: 'DELETE', label: 'Eliminación' }
  ];

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.isLoading.set(true);
    const raw = this.searchQuery().trim();
    const filters: { action?: string; actorId?: string } = {};
    if (this.actionFilter()) {
      filters.action = this.actionFilter();
    }
    if (this.actorFilter().trim()) {
      filters.actorId = this.actorFilter().trim();
    }

    this.auditoriaService.listAuditLogs(this.currentPage(), PAGE_SIZE, filters).subscribe({
      next: (resp: AuditLogListResponse) => {
        const filtered = raw
          ? resp.data.filter((log) => this.matchesSearch(log, raw.toLowerCase()))
          : resp.data;
        this.rows.set(filtered.map((log) => this.toRow(log)));
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

  onActorFilterChange(value: string): void {
    this.actorFilter.set(value);
    this.currentPage.set(1);
    if (this.searchDebounce) {
      clearTimeout(this.searchDebounce);
    }
    this.searchDebounce = setTimeout(() => {
      this.load();
    }, 250);
  }

  onPageChange(page: number): void {
    this.currentPage.set(page);
    this.load();
  }

  clearFilters(): void {
    this.searchQuery.set('');
    this.actionFilter.set('');
    this.actorFilter.set('');
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

  private matchesSearch(log: AuditLog, term: string): boolean {
    return (
      log.actor_email.toLowerCase().includes(term) ||
      log.action.toLowerCase().includes(term) ||
      log.resource_type.toLowerCase().includes(term)
    );
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
