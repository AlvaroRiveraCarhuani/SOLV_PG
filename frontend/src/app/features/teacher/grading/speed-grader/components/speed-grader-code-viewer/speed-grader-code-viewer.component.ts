import { Component, computed, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { TimeTravelReplayComponent } from '@shared/components/time-travel-replay/time-travel-replay.component';
import { CodeDiffViewerComponent } from '@shared/components/code-diff-viewer/code-diff-viewer.component';
import {
  BenchmarkReport,
  ProjectFile,
  SubmissionComment,
  SubmissionTimeline,
  TeacherSubmissionReviewDTO
} from '../../../../models/teacher.models';
import {
  commentRangeLabel,
  globalToLocalLine,
  localRangeForComment,
  localToGlobalLine
} from '../../review-line-map';

@Component({
  selector: 'speed-grader-code-viewer',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DateTextPipe,
    MachineDataDirective,
    CodeEditorComponent,
    CodeDiffViewerComponent,
    TimeTravelReplayComponent
  ],
  templateUrl: './speed-grader-code-viewer.component.html',
  styleUrl: './speed-grader-code-viewer.component.scss'
})
export class SpeedGraderCodeViewerComponent {
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly showReplayPlayer = input<boolean>(false);
  readonly showDiffView = input<boolean>(false);
  readonly timeline = input<SubmissionTimeline | null>(null);
  readonly isLoadingTimeline = input<boolean>(false);
  readonly benchmarkReport = input<BenchmarkReport | null>(null);
  readonly isLoadingBenchmark = input<boolean>(false);
  readonly selectedLineNumber = input<number | null>(null);
  readonly selectedLineEndNumber = input<number | null>(null);
  readonly newCommentText = input<string>('');
  readonly isAddingComment = input<boolean>(false);
  readonly codeLines = input<string[]>([]);
  readonly commentsByLine = input<Map<number, SubmissionComment[]>>(new Map());
  readonly starterBoilerplate = input<string>('');

  /** Diff nativo: base y versión visible, ya resueltas por el shell. */
  readonly diffOriginal = input<string>('');
  readonly diffModified = input<string>('');
  readonly diffLanguage = input<string>('plaintext');

  /** IDE: archivo seleccionado en el árbol. En juez queda en null. */
  readonly activeFile = input<ProjectFile | null>(null);
  readonly fileComments = input<SubmissionComment[]>([]);
  readonly generalComments = input<SubmissionComment[]>([]);
  readonly flatSnapshot = input<boolean>(false);

  readonly lineSelected = output<number | null>();
  readonly lineRangeSelected = output<{ start: number; end: number }>();
  readonly commentSubmitted = output<{ line: number; endLine: number | null }>();
  readonly newCommentTextChange = output<string>();
  readonly toggleDiff = output<void>();
  readonly toggleReplay = output<void>();

  readonly detectedLanguage = computed(() => {
    const code = this.review()?.code || '';
    if (/^\s*(import\s+|def\s+|class\s+.*:|print\(|from\s+.*import)/m.test(code)) {
      return 'python';
    }
    if (/^\s*(#include|int\s+main|std::|void\s+|using\s+namespace)/m.test(code)) {
      return 'cpp';
    }
    if (/^\s*(public\s+class|System\.out\.println|package\s+)/m.test(code)) {
      return 'java';
    }
    if (/^\s*(const\s+|let\s+|function\s+|console\.log|import\s+.*from)/m.test(code)) {
      return 'javascript';
    }
    return 'python';
  });

  readonly editorCode = computed(() => this.activeFile()?.content ?? this.review().code ?? '');

  readonly editorLanguage = computed(() => {
    const file = this.activeFile();
    if (file?.language) return file.language;
    if (this.review().language) return this.review().language;
    return this.detectedLanguage();
  });

  readonly lineCount = computed(() => {
    const file = this.activeFile();
    if (file) return file.line_count;
    return this.review().code ? this.codeLines().length : 0;
  });

  readonly selectedLocalLine = computed(() => {
    const file = this.activeFile();
    const globalLine = this.selectedLineNumber();
    if (!file || globalLine === null) return null;
    return globalToLocalLine(file, globalLine);
  });

  readonly selectedLocalLineEnd = computed(() => {
    const file = this.activeFile();
    const globalLine = this.selectedLineEndNumber();
    if (!file || globalLine === null) return null;
    return globalToLocalLine(file, globalLine);
  });

  /** Rango que muestra el formulario: local en IDE, global en juez. */
  readonly commentTargetStart = computed(() =>
    this.activeFile() ? this.selectedLocalLine() : this.selectedLineNumber()
  );

  readonly commentTargetEnd = computed(() =>
    this.activeFile() ? this.selectedLocalLineEnd() : this.selectedLineEndNumber()
  );

  readonly commentTargetLabel = computed(() => {
    const start = this.commentTargetStart();
    if (start === null) return null;
    const end = this.commentTargetEnd();
    return commentRangeLabel(start, end !== null && end > start ? end : start);
  });

  /** Línea global que espera el backend, incluso cuando la vista muestra una local. */
  readonly commentGlobalLine = computed(() => {
    const file = this.activeFile();
    const target = this.commentTargetStart();
    if (target === null) return null;
    return file ? localToGlobalLine(file, target) : target;
  });

  readonly commentGlobalEndLine = computed<number | null>(() => {
    const file = this.activeFile();
    const target = this.commentTargetEnd();
    if (target === null) return null;
    return file ? localToGlobalLine(file, target) : target;
  });

  readonly displayedComments = computed(() => {
    const file = this.activeFile();
    const comments = file ? this.fileComments() : this.review().comments || [];

    return comments.flatMap((comment) => {
      if (file) {
        const range = localRangeForComment(file, comment);
        return range
          ? [{ comment, localLine: range.start, rangeLabel: commentRangeLabel(range.start, range.end) }]
          : [];
      }
      return [{ comment, localLine: comment.line_number, rangeLabel: commentRangeLabel(comment.line_number, comment.line_number) }];
    });
  });

  onEditorLineClick(line: number): void {
    const file = this.activeFile();
    if (!file) {
      this.lineSelected.emit(line);
      return;
    }

    const globalLine = localToGlobalLine(file, line);
    if (globalLine !== null) {
      this.lineSelected.emit(globalLine);
    }
  }

  onEditorRangeSelected(range: { start: number; end: number }): void {
    const file = this.activeFile();
    if (!file) {
      this.lineRangeSelected.emit(range);
      return;
    }

    const start = localToGlobalLine(file, range.start);
    const end = localToGlobalLine(file, range.end);
    if (start !== null && end !== null && end > start) {
      this.lineRangeSelected.emit({ start, end });
    }
  }

  submitComment(): void {
    const globalLine = this.commentGlobalLine();
    if (globalLine === null) return;
    const end = this.commentGlobalEndLine();
    this.commentSubmitted.emit({
      line: globalLine,
      endLine: end !== null && end > globalLine ? end : null
    });
  }
}
