import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { TeacherItem } from '@core/models/admin.model';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';
import {
  LucideUserPlus,
  LucideLock,
  LucideCheckCircle,
  LucideClock,
  LucideAlertCircle,
  LucideSend,
  LucideRotateCcw,
  LucideCopy,
  LucideTrash2,
  LucideBookOpen,
  LucideUsers
} from '@lucide/angular';

const ENGLISH_MONTH_INDEX: Record<string, number> = {
  jan: 0, feb: 1, mar: 2, apr: 3, may: 4, jun: 5,
  jul: 6, aug: 7, sep: 8, oct: 9, nov: 10, dec: 11
};

@Component({
  selector: 'teacher-table',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    SearchBarComponent,
    StatusTabsComponent,
    PaginationBarComponent,
    LucideUserPlus,
    LucideLock,
    LucideCheckCircle,
    LucideClock,
    LucideAlertCircle,
    LucideSend,
    LucideRotateCcw,
    LucideCopy,
    LucideTrash2,
    LucideBookOpen
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
  readonly originComboboxOptions = input.required<ComboboxOption[]>();
  readonly selectedOriginLabel = input.required<string>();
  readonly counts = input.required<{ all: number; active: number; noCourses: number; pending: number; expired: number }>();

  readonly searchChange = output<string>();
  readonly statusChange = output<string>();
  readonly originChange = output<string>();
  readonly pageChange = output<number>();
  readonly resendInvitation = output<TeacherItem>();
  readonly revokeInvitation = output<TeacherItem>();
  readonly renewInvitation = output<TeacherItem>();
  readonly copyEmail = output<string>();
  readonly requestDeleteTeacher = output<TeacherItem>();
  readonly openTeacherCourses = output<TeacherItem>();
  readonly toggleMenu = output<string>();

  readonly paginationDisplay = computed(() => {
    const total = this.filteredCount();
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.currentPage() - 1) * this.pageSize() + 1;
    const to = Math.min(this.currentPage() * this.pageSize(), total);
    return { from, to };
  });

  formatInvitedAt(value: string): string {
    const legacyDate = /^(\d{2})-([A-Za-z]{3})-(\d{4})$/.exec(value);
    if (legacyDate) {
      const monthIndex = ENGLISH_MONTH_INDEX[legacyDate[2].toLowerCase()] ?? -1;
      if (monthIndex >= 0) {
        const iso = `${legacyDate[3]}-${String(monthIndex + 1).padStart(2, '0')}-${legacyDate[1]}`;
        return formatSolvDate(iso, 'daymonthyear') ?? value;
      }
      return value;
    }
    return formatSolvDate(value, 'daymonthyear') ?? value;
  }

  getInitials(name: string): string {
    if (!name) return 'PR';
    return name.split(' ').slice(0, 2).map(w => w[0] || '').join('').toUpperCase();
  }

  onOriginSelected(opt: ComboboxOption): void {
    this.originChange.emit(opt.value || 'all');
  }
}
