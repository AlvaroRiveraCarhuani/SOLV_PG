import { Component, input, output, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { SortHeaderComponent } from '@shared/components/sort-header/sort-header.component';
import { SortDirection } from '@shared/components/sort-header/sort-header.types';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { TeacherItem } from '@core/models/admin.model';
import {
  LucideUserPlus,
  LucideSend,
  LucideRotateCcw,
  LucideCopy,
  LucideCheck,
  LucideTrash2,
  LucideBookOpen,
  LucideMoreVertical,
  LucideEye,
  LucideUserMinus
} from '@lucide/angular';

@Component({
  selector: 'teacher-table',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    FilterPopoverComponent,
    SearchBarComponent,
    StatusTabsComponent,
    PaginationBarComponent,
    SortHeaderComponent,
    DismissibleDirective,
    DateTextPipe,
    LucideUserPlus,
    LucideSend,
    LucideRotateCcw,
    LucideCopy,
    LucideCheck,
    LucideTrash2,
    LucideBookOpen,
    LucideMoreVertical,
    LucideEye,
    LucideUserMinus
  ],
  templateUrl: './teacher-table.component.html',
  styleUrl: './teacher-table.component.scss'
})
export class TeacherTableComponent {
  readonly teachers = input.required<TeacherItem[]>();
  readonly isLoading = input.required<boolean>();
  readonly searchTerm = input.required<string>();
  readonly statusFilter = input.required<string>();
  readonly originFilter = input.required<string>();
  readonly currentPage = input.required<number>();
  readonly pageSize = input.required<number>();
  readonly totalPages = input.required<number>();
  readonly filteredCount = input.required<number>();
  readonly teacherStatusTabs = input.required<StatusTabItem[]>();
  readonly originComboboxOptions = input<unknown[]>([]);
  readonly selectedOriginLabel = input<string>('Todos los orígenes');
  readonly counts = input<{ all: number; active: number; noCourses: number; pending: number; expired: number }>({
    all: 0,
    active: 0,
    noCourses: 0,
    pending: 0,
    expired: 0
  });
  readonly sortColumn = input<string | null>(null);
  readonly sortDirection = input<SortDirection>('none');
  readonly activeMenuTeacherId = input<string | null>(null);

  readonly searchChange = output<string>();
  readonly statusChange = output<string>();
  readonly originChange = output<string>();
  readonly sortChange = output<{ column: string; direction: SortDirection }>();
  readonly pageChange = output<number>();
  readonly rowSelect = output<TeacherItem>();
  readonly openTeacherCourses = output<TeacherItem>();
  readonly resendInvitation = output<TeacherItem>();
  readonly revokeInvitation = output<TeacherItem>();
  readonly renewInvitation = output<TeacherItem>();
  readonly copyEmail = output<string>();
  readonly requestRevokeTeacher = output<TeacherItem>();
  readonly toggleMenu = output<string>();
  readonly menuClosed = output<void>();

  readonly copiedEmail = signal<string | null>(null);
  private copyResetTimer?: ReturnType<typeof setTimeout>;

  readonly paginationDisplay = computed(() => {
    const total = this.filteredCount();
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.currentPage() - 1) * this.pageSize() + 1;
    const to = Math.min(this.currentPage() * this.pageSize(), total);
    return { from, to };
  });

  onCopyEmail(email: string, event: Event): void {
    event.stopPropagation();
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(email);
    }
    this.copiedEmail.set(email);
    this.copyEmail.emit(email);
    if (this.copyResetTimer) clearTimeout(this.copyResetTimer);
    this.copyResetTimer = setTimeout(() => {
      this.copiedEmail.set(null);
    }, 2000);
  }

  onRowClick(teacher: TeacherItem, event: Event): void {
    if (teacher.status === 'active') {
      this.rowSelect.emit(teacher);
    }
  }

  onOpenCourses(teacher: TeacherItem, event: Event): void {
    event.stopPropagation();
    this.openTeacherCourses.emit(teacher);
  }

  onToggleMenu(event: Event, teacherId: string): void {
    event.stopPropagation();
    this.toggleMenu.emit(teacherId);
  }

  onDismissMenu(): void {
    this.menuClosed.emit();
  }

  onOpenDrawer(event: Event, teacher: TeacherItem): void {
    event.stopPropagation();
    this.menuClosed.emit();
    this.rowSelect.emit(teacher);
  }

  onRevokeRole(event: Event, teacher: TeacherItem): void {
    event.stopPropagation();
    this.menuClosed.emit();
    this.requestRevokeTeacher.emit(teacher);
  }

  onResend(event: Event, teacher: TeacherItem): void {
    event.stopPropagation();
    this.resendInvitation.emit(teacher);
  }

  onRevokeInvitation(event: Event, teacher: TeacherItem): void {
    event.stopPropagation();
    this.revokeInvitation.emit(teacher);
  }

  onRenew(event: Event, teacher: TeacherItem): void {
    event.stopPropagation();
    this.renewInvitation.emit(teacher);
  }

  onOriginSelected(value: string): void {
    this.originChange.emit(value);
  }

  onClearFilters(): void {
    this.originChange.emit('all');
  }

  getPendingExpirationText(teacher: TeacherItem): string {
    const inv = teacher.invited_at || '';
    const matchH = /hace\s+(\d+)\s*h/i.exec(inv);
    if (matchH) {
      const elapsed = parseInt(matchH[1], 10);
      const remaining = Math.max(1, 72 - elapsed);
      return `Vence en ${remaining} h`;
    }
    const matchD = /hace\s+(\d+)\s*d/i.exec(inv);
    if (matchD) {
      const elapsedD = parseInt(matchD[1], 10);
      const remaining = Math.max(1, 72 - elapsedD * 24);
      return `Vence en ${remaining} h`;
    }
    if (inv.toLowerCase().includes('minuto') || inv.toLowerCase().includes('segundo') || inv.toLowerCase().includes('recién')) {
      return 'Vence en 72 h';
    }
    return 'Vigencia de 72 h';
  }

  getExpiredText(teacher: TeacherItem): string {
    const inv = (teacher.invited_at || '').trim();
    const matchD = /hace\s+(\d+)\s*d/i.exec(inv);
    if (matchD) {
      const d = parseInt(matchD[1], 10);
      return `Expiró hace ${d} ${d === 1 ? 'día' : 'días'}`;
    }
    const matchH = /hace\s+(\d+)\s*h/i.exec(inv);
    if (matchH) {
      return `Expiró hace ${matchH[1]} h`;
    }
    return '';
  }
}
