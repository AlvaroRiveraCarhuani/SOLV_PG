import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import {
  LucideAlertCircle,
  LucideArrowLeft,
  LucideExternalLink,
  LucideFileCode
} from '@lucide/angular';
import { StudentService, StudentSubmissionHistoryItem } from '@core/services/student.service';
import { ProjectFile } from '@core/models/project-file.model';
import { RubricDTO } from '@core/models/rubric.model';
import { SubmissionComment } from '@core/models/submission-comment.model';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { ReviewFileTreeComponent } from '@shared/components/review-file-tree/review-file-tree.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';

function parseSnapshotFallback(code: string, language = ''): ProjectFile[] {
  if (!code) {
    return [];
  }
  if (!code.includes('# === FILE: ')) {
    const ext = language === 'typescript' ? 'ts' : language === 'python' ? 'py' : language === 'go' ? 'go' : 'txt';
    return [
      {
        path: `src/main.${ext}`,
        language,
        content: code,
        start_line: 1,
        line_count: code.split('\n').length
      }
    ];
  }
  const regex = /^# === FILE: (.+?) ===$/gm;
  const matches: { path: string; index: number }[] = [];
  let m: RegExpExecArray | null;
  while ((m = regex.exec(code)) !== null) {
    matches.push({ path: m[1].trim(), index: m.index + m[0].length });
  }
  if (matches.length === 0) return [];

  const files: ProjectFile[] = [];
  for (let i = 0; i < matches.length; i++) {
    const start = matches[i].index;
    const end = i + 1 < matches.length ? code.lastIndexOf('\n# === FILE: ', matches[i + 1].index) : code.length;
    const content = code.slice(start, end).replace(/^\r?\n/, '').replace(/\r?\n$/, '');
    const lines = content ? content.split('\n').length : 0;
    files.push({
      path: matches[i].path,
      language: '',
      content,
      start_line: 1,
      line_count: lines
    });
  }
  return files;
}

import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';

@Component({
  selector: 'student-submission-detail',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    EnvironmentLabelPipe,
    RouterLink,
    LucideArrowLeft,
    LucideExternalLink,
    LucideFileCode,
    LucideAlertCircle,
    VerdictBadgeComponent,
    ReviewFileTreeComponent,
    CodeEditorComponent,
    SkeletonLoaderComponent,
    MachineDataDirective,
    DateTextPipe
  ],
  templateUrl: './student-submission-detail.component.html',
  styleUrl: './student-submission-detail.component.scss'
})
export class StudentSubmissionDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private studentService = inject(StudentService);

  readonly isLoading = signal<boolean>(true);
  readonly loadError = signal<string | null>(null);

  readonly submission = signal<StudentSubmissionHistoryItem | null>(null);
  readonly comments = signal<SubmissionComment[]>([]);
  readonly rubric = signal<RubricDTO | null>(null);
  readonly selectedFilePath = signal<string | null>(null);

  readonly isProject = computed<boolean>(() => {
    return this.submission()?.environment_type === 'IDE_PERSISTENTE';
  });

  readonly projectFiles = computed<ProjectFile[]>(() => {
    const sub = this.submission();
    if (!sub) return [];
    if (sub.files && sub.files.length > 0) {
      return sub.files;
    }
    if (this.isProject() && sub.code) {
      return parseSnapshotFallback(sub.code, sub.language || '');
    }
    return [];
  });

  readonly activeFile = computed<ProjectFile | null>(() => {
    const files = this.projectFiles();
    if (files.length === 0) return null;
    const path = this.selectedFilePath();
    if (!path) return files[0];
    return files.find((f) => f.path === path) || files[0];
  });

  readonly fileLanguage = computed<string>(() => {
    const file = this.activeFile();
    if (file?.language) return file.language;
    const path = (file?.path || '').toLowerCase();
    if (path.endsWith('.ts')) return 'typescript';
    if (path.endsWith('.js')) return 'javascript';
    if (path.endsWith('.json')) return 'json';
    if (path.endsWith('.py')) return 'python';
    if (path.endsWith('.go')) return 'go';
    if (path.endsWith('.java')) return 'java';
    if (path.endsWith('.cpp') || path.endsWith('.c')) return 'cpp';
    if (path.endsWith('.sql')) return 'sql';
    if (path.endsWith('.html')) return 'html';
    if (path.endsWith('.css') || path.endsWith('.scss')) return 'css';
    return (this.submission()?.language || 'plaintext').toLowerCase();
  });

  readonly singleCodeLanguage = computed<string>(() => {
    const lang = (this.submission()?.language || '').toLowerCase();
    return lang || 'python';
  });

  readonly activeFileComments = computed<SubmissionComment[]>(() => {
    const filePath = this.activeFile()?.path;
    return this.comments().filter((c) => {
      if (!c.file_path) return false;
      return c.file_path === filePath;
    });
  });

  readonly generalComments = computed<SubmissionComment[]>(() => {
    return this.comments().filter((c) => !c.file_path && (!c.line_number || c.line_number === 1));
  });

  loadPromise?: Promise<void>;

  ngOnInit(): void {
    const subId = this.route.snapshot.paramMap.get('submissionId');
    if (!subId) {
      this.loadError.set('Identificador de entrega no especificado.');
      this.isLoading.set(false);
      return;
    }
    this.loadPromise = this.loadDetails(subId);
  }

  async loadDetails(submissionId: string): Promise<void> {
    this.isLoading.set(true);
    this.loadError.set(null);

    try {
      const sub = await this.studentService.getSubmissionReview(submissionId);
      if (!sub) {
        this.loadError.set('No se encontró la entrega solicitada o no pertenece a tus materias.');
        return;
      }
      this.submission.set(sub);

      // Initialize selected file
      const files = (sub.files && sub.files.length > 0)
        ? sub.files
        : (sub.environment_type === 'IDE_PERSISTENTE' && sub.code ? parseSnapshotFallback(sub.code, sub.language || '') : []);
      this.selectedFilePath.set(files[0]?.path ?? null);

      // Fetch comments & rubric in parallel
      const [comments, rubric] = await Promise.all([
        this.studentService.getSubmissionComments(submissionId),
        sub.exercise_id ? this.studentService.getExerciseRubric(sub.exercise_id) : Promise.resolve(null)
      ]);

      this.comments.set(comments);
      this.rubric.set(rubric);
    } catch {
      this.loadError.set('Ocurrió un error al cargar la información de la entrega.');
    } finally {
      this.isLoading.set(false);
    }
  }

  onFileSelected(path: string): void {
    this.selectedFilePath.set(path);
  }

  commentsForLine(lineNumber: number): SubmissionComment[] {
    const filePath = this.activeFile()?.path;
    return this.comments().filter((c) => {
      if (c.file_path) {
        return c.file_path === filePath && (c.local_line_number === lineNumber || c.line_number === lineNumber);
      }
      return false;
    });
  }
}
