import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TeacherItem } from '@core/models/admin.model';
import { TeacherCourse } from '../../../services/admin-teachers.service';
import { LucideUsers, LucideBookOpen, LucideArrowRightLeft, LucidePlus } from '@lucide/angular';

@Component({
  selector: 'teacher-courses-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    FormFieldComponent,
    ModalShellComponent,
    MachineDataDirective,
    LucideUsers,
    LucideBookOpen,
    LucideArrowRightLeft,
    LucidePlus
  ],
  templateUrl: './teacher-courses-modal.component.html',
  styleUrl: './teacher-courses-modal.component.scss'
})
export class TeacherCoursesModalComponent {
  readonly teacher = input.required<TeacherItem | null>();
  readonly teacherCourses = input.required<TeacherCourse[]>();
  readonly isLoadingCourses = input.required<boolean>();
  readonly totalStudentsForTeacher = input.required<number>();
  readonly activeTeacherComboboxOptions = input.required<ComboboxOption[]>();
  readonly selectedNewTeacherLabel = input.required<string>();
  readonly selectedNewTeacherId = input.required<string>();
  readonly reassigningCourse = input.required<TeacherCourse | null>();
  readonly reassignReason = input.required<string>();
  readonly isSubmittingReassign = input.required<boolean>();
  readonly isAssigningNewCourse = input.required<boolean>();
  readonly availableCourseComboboxOptions = input.required<ComboboxOption[]>();
  readonly selectedCourseToAssignLabel = input.required<string>();
  readonly selectedCourseToAssign = input.required<string>();
  readonly isLoadingAvailableCourses = input.required<boolean>();

  readonly dismiss = output<void>();
  readonly startReassign = output<TeacherCourse>();
  readonly cancelReassign = output<void>();
  readonly confirmReassignCourse = output<void>();
  readonly startAssignCourseToTeacher = output<void>();
  readonly cancelAssignCourseToTeacher = output<void>();
  readonly confirmAssignCourseToTeacher = output<void>();
  readonly newTeacherSelected = output<ComboboxOption>();
  readonly courseToAssignSelected = output<ComboboxOption>();
  readonly reassignReasonChange = output<string>();

  getInitials(name: string): string {
    if (!name) return 'PR';
    return name.split(' ').slice(0, 2).map(w => w[0] || '').join('').toUpperCase();
  }

  onReasonInput(event: Event): void {
    this.reassignReasonChange.emit((event.target as HTMLInputElement).value);
  }
}
