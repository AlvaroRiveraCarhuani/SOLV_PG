import {
  Component,
  input,
  output,
  signal,
  viewChild,
  computed,
  ChangeDetectionStrategy
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { RosterUploadComponent } from '@shared/components/roster-upload/roster-upload.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SubjectOption } from '../../../services/admin-students.service';
import { LucideBookOpen } from '@lucide/angular';

@Component({
  selector: 'student-enroll-roster-modal',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    ModalShellComponent,
    RosterUploadComponent,
    ComboboxComponent,
    LucideBookOpen
  ],
  templateUrl: './student-enroll-roster-modal.component.html',
  styleUrl: './student-enroll-roster-modal.component.scss'
})
export class StudentEnrollRosterModalComponent {
  readonly rosterUpload = viewChild(RosterUploadComponent);

  readonly subjects = input<SubjectOption[]>([]);
  readonly isSubmitting = input<boolean>(false);

  readonly selectedSubjectId = signal('');
  readonly closed = output<void>();
  readonly rosterSubmit = output<{ subjectId: string; emails: string[] }>();

  readonly subjectOptions = computed<ComboboxOption[]>(() =>
    this.subjects().map(subject => ({
      id: subject.id,
      label: `${subject.code} - ${subject.name}`,
      value: subject.id
    }))
  );

  readonly selectedSubjectLabel = computed(() => {
    const opt = this.subjectOptions().find(o => o.value === this.selectedSubjectId());
    return opt?.label ?? '';
  });

  onSubjectSelected(option: ComboboxOption): void {
    this.selectedSubjectId.set(String(option.value));
  }

  confirm(): void {
    const emails = this.rosterUpload()?.parsedRoster()?.valid_emails ?? [];
    const subjectId = this.selectedSubjectId();
    if (!subjectId || emails.length === 0) {
      return;
    }
    this.rosterSubmit.emit({ subjectId, emails });
  }
}
