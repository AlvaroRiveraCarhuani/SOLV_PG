import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { AdminStudentItem } from '../../../services/admin-students.service';
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
  LucideMoreVertical
} from '@lucide/angular';

@Component({
  selector: 'student-table',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    SearchBarComponent,
    PaginationBarComponent,
    DismissibleDirective,
    LucideAlertTriangle,
    LucideShieldAlert,
    LucideRotateCcw,
    LucideBookOpen,
    LucideUserX,
    LucideUserCheck,
    LucideCopy,
    LucideCheck,
    LucideMoreVertical
  ],
  templateUrl: './student-table.component.html',
  styleUrls: ['./student-table.component.scss']
})
export class StudentTableComponent {
  readonly students = input.required<AdminStudentItem[]>();
  readonly isLoading = input<boolean>(false);
  readonly searchTerm = input<string>('');
  readonly selectedPeriodLabel = input<string>('Todos los periodos');
  readonly periodComboboxOptions = input<ComboboxOption[]>([]);
  readonly selectedSubjectLabel = input<string>('Todas las materias');
  readonly subjectComboboxOptions = input<ComboboxOption[]>([]);
  readonly selectedStatusLabel = input<string>('Todos los estados');
  readonly statusComboboxOptions = input<ComboboxOption[]>([]);
  readonly currentPage = input<number>(1);
  readonly totalPages = input<number>(1);
  readonly totalItems = input<number>(0);
  readonly paginationFrom = input<number>(0);
  readonly paginationTo = input<number>(0);
  readonly activeMenuStudentId = input<string | null>(null);

  readonly searchChange = output<string>();
  readonly periodSelected = output<ComboboxOption>();
  readonly subjectSelected = output<ComboboxOption>();
  readonly statusSelected = output<ComboboxOption>();
  readonly pageChange = output<number>();
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

  onPeriodSelected(opt: ComboboxOption): void {
    this.periodSelected.emit(opt);
  }

  onSubjectSelected(opt: ComboboxOption): void {
    this.subjectSelected.emit(opt);
  }

  onStatusSelected(opt: ComboboxOption): void {
    this.statusSelected.emit(opt);
  }

  onPageChange(page: number): void {
    this.pageChange.emit(page);
  }

  onOpenCourses(student: AdminStudentItem): void {
    this.openCourses.emit(student);
  }

  onOpenResetOOM(student: AdminStudentItem): void {
    this.menuClosed.emit();
    this.openResetOOM.emit(student);
  }

  onToggleStatus(student: AdminStudentItem): void {
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

  onCopyEmail(email: string): void {
    this.copyEmail.emit(email);
    this.copiedEmail.set(email);
    setTimeout(() => {
      if (this.copiedEmail() === email) {
        this.copiedEmail.set(null);
      }
    }, 2000);
  }

  onResetFilters(): void {
    this.searchChange.emit('');
    this.subjectSelected.emit({ id: 'all', label: 'Todas las materias', value: 'all' });
    this.statusSelected.emit({ id: 'all', label: 'Todos los estados', value: 'all' });
  }

  getInitials(firstName: string, lastName: string): string {
    const first = firstName ? firstName.trim().charAt(0).toUpperCase() : '';
    const last = lastName ? lastName.trim().charAt(0).toUpperCase() : '';
    return first + last || 'E';
  }

  formatDate(dateStr?: string | null): string {
    if (!dateStr || dateStr.trim() === '') return 'Sin registros';
    return formatSolvDate(dateStr, 'datetime') ?? dateStr;
  }
}
