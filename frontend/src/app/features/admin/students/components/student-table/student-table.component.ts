import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { SortHeaderComponent } from '@shared/components/sort-header/sort-header.component';
import { SortDirection } from '@shared/components/sort-header/sort-header.types';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { AdminStudentItem, AcademicPeriodOption, SubjectOption } from '../../../services/admin-students.service';
import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';
import {
  LucideAlertTriangle,
  LucideShieldAlert,
  LucideRotateCcw,
  LucideBookOpen,
  LucideUserX,
  LucideUserCheck,
  LucideCopy,
  LucideCheck,
  LucideMoreVertical,
  LucideEye
} from '@lucide/angular';

@Component({
  selector: 'student-table',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    FilterPopoverComponent,
    SearchBarComponent,
    PaginationBarComponent,
    SortHeaderComponent,
    DismissibleDirective,
    MachineDataDirective,
    LucideAlertTriangle,
    LucideShieldAlert,
    LucideRotateCcw,
    LucideBookOpen,
    LucideUserX,
    LucideUserCheck,
    LucideCopy,
    LucideCheck,
    LucideMoreVertical,
    LucideEye
  ],
  templateUrl: './student-table.component.html',
  styleUrls: ['./student-table.component.scss']
})
export class StudentTableComponent {
  readonly students = input.required<AdminStudentItem[]>();
  readonly isLoading = input<boolean>(false);
  readonly searchTerm = input<string>('');
  readonly selectedPeriod = input<string>('all');
  readonly academicPeriods = input<AcademicPeriodOption[]>([]);
  readonly selectedSubject = input<string>('all');
  readonly subjects = input<SubjectOption[]>([]);
  readonly activeFilterCount = input<number>(0);
  readonly canClearFilters = input<boolean>(false);

  // Propiedades de compatibilidad
  readonly selectedPeriodLabel = input<string>('Todos los periodos');
  readonly periodComboboxOptions = input<ComboboxOption[]>([]);
  readonly selectedSubjectLabel = input<string>('Todas las materias');
  readonly subjectComboboxOptions = input<ComboboxOption[]>([]);
  readonly selectedStatusLabel = input<string>('Todos los estados');
  readonly statusComboboxOptions = input<ComboboxOption[]>([]);

  readonly currentPage = input<number>(1);
  readonly totalPages = input<number>(1);
  readonly totalItems = input<number>(0);
  readonly totalRecords = input<number>(0);
  readonly paginationFrom = input<number>(0);
  readonly paginationTo = input<number>(0);
  readonly activeMenuStudentId = input<string | null>(null);
  readonly sortColumn = input<string | null>(null);
  readonly sortDirection = input<SortDirection>('none');

  readonly searchChange = output<string>();
  readonly periodSelected = output<string>();
  readonly subjectSelected = output<string>();
  readonly clearFilters = output<void>();
  readonly sortChange = output<{ column: string; direction: SortDirection }>();
  readonly pageChange = output<number>();
  readonly rowSelect = output<AdminStudentItem>();
  readonly openCourses = output<AdminStudentItem>();
  readonly openResetOOM = output<AdminStudentItem>();
  readonly toggleStatus = output<AdminStudentItem>();
  readonly toggleMenu = output<string>();
  readonly menuClosed = output<void>();
  readonly copyEmail = output<string>();

  readonly copiedEmail = signal<string | null>(null);

  onSearchChange(term: string): void {
    this.searchChange.emit(term);
  }

  onSelectPeriod(periodId: string): void {
    this.periodSelected.emit(periodId);
  }

  onSelectSubject(subjectId: string): void {
    this.subjectSelected.emit(subjectId);
  }

  onClearFilters(): void {
    this.clearFilters.emit();
  }

  onPageChange(page: number): void {
    this.pageChange.emit(page);
  }

  onRowClick(student: AdminStudentItem, event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (target.closest('button, a, input, select, textarea, .actions-dropdown, .actions-menu-container')) {
      return;
    }
    this.rowSelect.emit(student);
  }

  onOpenDetail(student: AdminStudentItem, event: MouseEvent): void {
    event.stopPropagation();
    this.menuClosed.emit();
    this.rowSelect.emit(student);
  }

  onDirectUnblock(student: AdminStudentItem, event: MouseEvent): void {
    event.stopPropagation();
    this.openResetOOM.emit(student);
  }

  onOpenCourses(student: AdminStudentItem, event?: MouseEvent): void {
    if (event) {
      event.stopPropagation();
    }
    this.openCourses.emit(student);
  }

  onOpenResetOOM(student: AdminStudentItem, event?: MouseEvent): void {
    if (event) {
      event.stopPropagation();
    }
    this.menuClosed.emit();
    this.openResetOOM.emit(student);
  }

  onToggleStatus(student: AdminStudentItem, event?: MouseEvent): void {
    if (event) {
      event.stopPropagation();
    }
    this.menuClosed.emit();
    this.toggleStatus.emit(student);
  }

  onToggleMenu(studentId: string, event: Event): void {
    event.stopPropagation();
    this.toggleMenu.emit(studentId);
  }

  onDismissMenu(): void {
    this.menuClosed.emit();
  }

  onCopyEmail(email: string, event?: MouseEvent): void {
    if (event) {
      event.stopPropagation();
    }
    this.copyEmail.emit(email);
    this.copiedEmail.set(email);
    setTimeout(() => {
      if (this.copiedEmail() === email) {
        this.copiedEmail.set(null);
      }
    }, 2000);
  }

  onResetFilters(): void {
    this.clearFilters.emit();
    this.searchChange.emit('');
  }

  formatDate(dateStr?: string | null): string {
    if (!dateStr || dateStr.trim() === '') return '—';
    return formatSolvDate(dateStr, 'datetime') ?? dateStr;
  }
}
