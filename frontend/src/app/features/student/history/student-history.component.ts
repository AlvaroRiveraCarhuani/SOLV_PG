import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { StudentService, StudentSubmissionHistoryItem } from '@core/services/student.service';
import {
  VERDICT_FILTER_OPTIONS,
  matchesVerdictFilter
} from '@core/models/verdict.model';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { environmentTypeOptions } from '@shared/constants/environment-types.constants';

@Component({
  selector: 'student-history',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    RouterLink,
    ComboboxComponent,
    SearchBarComponent,
    VerdictBadgeComponent,
    SkeletonLoaderComponent,
    MachineDataDirective,
    DateTextPipe
  ],
  templateUrl: './student-history.component.html',
  styleUrl: './student-history.component.scss'
})
export class StudentHistoryComponent implements OnInit {
  private studentService = inject(StudentService);

  readonly isLoading = signal<boolean>(true);
  readonly subjectFilter = signal<string>('all');
  readonly verdictFilter = signal<string>('all');
  readonly environmentFilter = signal<string>('all');
  readonly searchQuery = signal<string>('');

  readonly submissions = computed<StudentSubmissionHistoryItem[]>(() =>
    this.studentService.submissionsHistory()
  );

  readonly verdictOptions: ComboboxOption[] = VERDICT_FILTER_OPTIONS;

  readonly environmentOptions: ComboboxOption[] = [
    { id: 'all', label: 'Todos los entornos', value: 'all' },
    ...environmentTypeOptions({ include: ['IDE_PERSISTENTE', 'JUEZ_EFIMERO'] })
  ];

  readonly subjectOptions = computed<ComboboxOption[]>(() => {
    const labels = new Map<string, string>();
    for (const sub of this.submissions()) {
      const key = this.subjectKey(sub);
      if (!key || labels.has(key)) {
        continue;
      }
      const code = sub.subject_code?.trim() ?? '';
      const name = sub.subject_name?.trim() ?? '';
      labels.set(key, [code, name].filter(Boolean).join(' — ') || 'Materia sin nombre');
    }
    return [
      { id: 'all', label: 'Todas las materias', value: 'all' },
      ...Array.from(labels, ([value, label]) => ({ id: value, label, value }))
    ];
  });

  readonly filteredSubmissions = computed<StudentSubmissionHistoryItem[]>(() => {
    const subject = this.subjectFilter();
    const verdict = this.verdictFilter();
    const env = this.environmentFilter();
    const query = this.searchQuery().trim().toLowerCase();

    return this.submissions().filter((sub) => {
      if (subject !== 'all' && this.subjectKey(sub) !== subject) {
        return false;
      }
      if (!matchesVerdictFilter(sub.verdict, verdict)) {
        return false;
      }
      if (env !== 'all') {
        const subEnv = sub.environment_type || 'JUEZ_EFIMERO';
        if (subEnv !== env) {
          return false;
        }
      }
      if (!query) {
        return true;
      }
      return (sub.exercise_title ?? '').toLowerCase().includes(query);
    });
  });

  ngOnInit(): void {
    void this.loadSubmissionsHistory();
  }

  private async loadSubmissionsHistory(): Promise<void> {
    try {
      await this.studentService.loadSubmissionsHistory();
    } finally {
      this.isLoading.set(false);
    }
  }

  private subjectKey(sub: StudentSubmissionHistoryItem): string {
    return sub.subject_id?.trim() || sub.subject_code?.trim() || '';
  }
}
