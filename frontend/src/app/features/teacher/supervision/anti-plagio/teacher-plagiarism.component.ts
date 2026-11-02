import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  LucideRefreshCw,
  LucideExternalLink,
  LucideCheckCircle2,
  LucideGitCompare,
  LucideInfo,
  LucideFileCode,
  LucideFiles,
  LucideArrowLeftRight,
  LucideClock,
  LucideClipboardPaste,
  LucideKeyboard,
  LucideGauge
} from '@lucide/angular';
import { TeacherCourseService } from '../../services/teacher-course.service';
import {
  PlagiarismPair,
  PlagiarismPairSide,
  PlagiarismRange,
  PlagiarismReport,
  PlagiarismMatch
} from '../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { CodeEditorComponent, CodeLineRange } from '@shared/components/code-editor/code-editor.component';

interface AffectedFile {
  path: string;
  language: string;
  rangesA: number;
  rangesB: number;
}

/**
 * Auditoría de similitud de una práctica. La pantalla enfrenta dos entregas del
 * mismo ejercicio en tres columnas: el par y sus archivos, el diff con tramos
 * resaltados y la cadencia de escritura. El contexto viene de la ficha de la
 * práctica y el motor ya restó el código base antes de comparar.
 */
@Component({
  selector: 'teacher-plagiarism',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideRefreshCw,
    LucideExternalLink,
    LucideCheckCircle2,
    LucideGitCompare,
    LucideInfo,
    LucideFileCode,
    LucideFiles,
    LucideArrowLeftRight,
    LucideClock,
    LucideClipboardPaste,
    LucideKeyboard,
    LucideGauge,
    MachineDataDirective,
    SkeletonLoaderComponent,
    ComboboxComponent,
    CodeEditorComponent
  ],
  templateUrl: './teacher-plagiarism.component.html',
  styleUrl: './teacher-plagiarism.component.scss'
})
export class TeacherPlagiarismComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private courseService = inject(TeacherCourseService);

  readonly courseId = signal<string>('');
  readonly exerciseId = signal<string>('');

  report = signal<PlagiarismReport | null>(null);
  selectedMatch = signal<PlagiarismMatch | null>(null);
  pair = signal<PlagiarismPair | null>(null);
  selectedPath = signal<string>('');
  isLoading = signal<boolean>(false);
  isScanning = signal<boolean>(false);
  isLoadingPair = signal<boolean>(false);
  pairError = signal<string>('');
  syncScroll = signal<boolean>(true);
  /** Posición vertical compartida por los dos editores del diff. */
  scrollRatio = signal<number | null>(null);

  // Filtros
  riskFilter = signal<'all' | 'critical' | 'warning' | 'info'>('all');
  searchTerm = signal<string>('');

  readonly riskOptions: ComboboxOption<'all' | 'critical' | 'warning' | 'info'>[] = [
    { id: 'all', label: 'Todos', value: 'all' },
    { id: 'critical', label: 'Crítico (>= 85%)', value: 'critical' },
    { id: 'warning', label: 'Advertencia (70-84%)', value: 'warning' },
    { id: 'info', label: 'Informativo (< 70%)', value: 'info' }
  ];

  readonly practiceTitle = computed(() => this.report()?.exercise_title ?? '');

  /** La modalidad decide el visor: IDE multipanel o evaluador de archivo único. */
  readonly isIde = computed(() => this.pair()?.exercise.environment_type === 'IDE_PERSISTENTE');
  readonly baseFiles = computed(() => this.pair()?.exercise.base_files ?? []);
  readonly pairRanges = computed<PlagiarismRange[]>(
    () => this.pair()?.ranges ?? this.selectedMatch()?.ranges ?? []
  );

  // Computeds
  filteredMatches = computed(() => {
    const rep = this.report();
    if (!rep || !rep.matches) return [];

    let list = rep.matches;
    const risk = this.riskFilter();
    const query = this.searchTerm().trim().toLowerCase();

    if (risk !== 'all') {
      list = list.filter(m => m.risk_level === risk);
    }

    if (query) {
      list = list.filter(m =>
        m.student_name_a.toLowerCase().includes(query) ||
        m.student_name_b.toLowerCase().includes(query) ||
        m.exercise_title.toLowerCase().includes(query)
      );
    }

    return list;
  });

  kpiStats = computed(() => {
    const rep = this.report();
    if (!rep || !rep.matches) {
      return { totalPairs: 0, criticalCount: 0, warningCount: 0, uniqueStudents: 0 };
    }

    const matches = rep.matches;
    const criticalCount = matches.filter(m => m.risk_level === 'critical').length;
    const warningCount = matches.filter(m => m.risk_level === 'warning').length;

    const studentIds = new Set<string>();
    matches.forEach(m => {
      if (m.student_id_a) studentIds.add(m.student_id_a);
      if (m.student_id_b) studentIds.add(m.student_id_b);
    });

    return {
      totalPairs: matches.length,
      criticalCount,
      warningCount,
      uniqueStudents: studentIds.size
    };
  });

  /** Archivos con tramos compartidos. En el evaluador no hay árbol que mostrar. */
  affectedFiles = computed<AffectedFile[]>(() => {
    if (!this.isIde()) return [];

    const ranges = this.pairRanges();
    const files = new Map<string, AffectedFile>();

    for (const range of ranges) {
      for (const side of [range.source_a, range.source_b]) {
        if (!side.file_path) continue;
        const entry = files.get(side.file_path) ?? {
          path: side.file_path,
          language: this.fileLanguage(this.pair()?.a, side.file_path) ?? 'text',
          rangesA: 0,
          rangesB: 0
        };
        if (side === range.source_a) entry.rangesA += 1;
        if (side === range.source_b) entry.rangesB += 1;
        files.set(side.file_path, entry);
      }
    }

    if (files.size > 0) {
      return [...files.values()].sort((a, b) => a.path.localeCompare(b.path));
    }

    return (this.pair()?.a.files ?? []).map(file => ({
      path: file.path,
      language: file.language,
      rangesA: 0,
      rangesB: 0
    }));
  });

  /** Otras entregas comparadas contra alguna de las dos del par actual. */
  counterpartOptions = computed<PlagiarismMatch[]>(() => {
    const selected = this.selectedMatch();
    if (!selected) return [];

    return this.filteredMatches().filter(match => {
      if (match === selected) return false;
      return (
        match.submission_id_a === selected.submission_id_a ||
        match.submission_id_b === selected.submission_id_b ||
        match.submission_id_a === selected.submission_id_b ||
        match.submission_id_b === selected.submission_id_a
      );
    });
  });

  readonly codeA = computed(() => this.fileContent(this.pair()?.a, this.selectedPath()));
  readonly codeB = computed(() => this.fileContent(this.pair()?.b, this.selectedPath()));
  readonly languageA = computed(
    () => this.fileLanguage(this.pair()?.a, this.selectedPath()) ?? this.pair()?.a.language ?? 'text'
  );
  readonly languageB = computed(
    () => this.fileLanguage(this.pair()?.b, this.selectedPath()) ?? this.pair()?.b.language ?? 'text'
  );

  readonly highlightsA = computed<CodeLineRange[]>(() =>
    this.toLineRanges(this.pairRanges().map(range => range.source_a))
  );
  readonly highlightsB = computed<CodeLineRange[]>(() =>
    this.toLineRanges(this.pairRanges().map(range => range.source_b))
  );

  ngOnInit(): void {
    const courseId = this.route.snapshot.paramMap.get('courseId') ?? '';
    const exerciseId = this.route.snapshot.paramMap.get('exerciseId') ?? '';

    if (!courseId || !exerciseId) {
      void this.router.navigate(['/teacher/por-calificar'], { replaceUrl: true });
      return;
    }

    this.courseId.set(courseId);
    this.exerciseId.set(exerciseId);
    this.runScan();
  }

  runScan(): void {
    const courseId = this.courseId();
    const exerciseId = this.exerciseId();
    if (!courseId || !exerciseId) return;

    this.isLoading.set(true);
    this.isScanning.set(true);

    this.courseService.analyzePlagiarism(courseId, exerciseId).subscribe({
      next: report => {
        this.report.set(report);
        const first = report.matches?.[0] ?? null;
        this.isLoading.set(false);
        this.isScanning.set(false);
        if (first) {
          this.selectMatch(first);
        } else {
          this.selectedMatch.set(null);
          this.pair.set(null);
        }
      },
      error: () => {
        this.isLoading.set(false);
        this.isScanning.set(false);
      }
    });
  }

  selectMatch(match: PlagiarismMatch): void {
    this.selectedMatch.set(match);
    this.selectedPath.set('');
    this.scrollRatio.set(null);
    this.loadPair(match);
  }

  selectPath(path: string): void {
    this.selectedPath.set(path);
  }

  toggleSyncScroll(): void {
    this.syncScroll.update(value => !value);
  }

  onScrollRatioChange(ratio: number | null): void {
    if (!this.syncScroll()) return;
    this.scrollRatio.set(ratio);
  }

  formatDuration(ms: number): string {
    if (!ms) return '0 s';
    if (ms < 60000) return `${Math.round(ms / 1000)} s`;
    const minutes = Math.floor(ms / 60000);
    const seconds = Math.round((ms % 60000) / 1000);
    return seconds > 0 ? `${minutes} min ${seconds} s` : `${minutes} min`;
  }

  formatPercent(value: number): string {
    return `${value.toFixed(1)}%`;
  }

  verdictText(verdict: string): string {
    switch (verdict) {
      case 'AC':
        return 'Todos los tests pasan';
      case 'WA':
        return 'Fallan tests';
      case 'TLE':
        return 'Tiempo excedido';
      case 'RE':
        return 'Error en ejecución';
      case 'AST_BLOCKED':
        return 'Bloqueado por análisis';
      default:
        return 'Sin veredicto';
    }
  }

  verdictClass(verdict: string): string {
    switch (verdict) {
      case 'AC':
        return 'verdict-success';
      case 'TLE':
        return 'verdict-warning';
      case 'RE':
        return 'verdict-orange';
      default:
        return 'verdict-error';
    }
  }

  private loadPair(match: PlagiarismMatch): void {
    this.isLoadingPair.set(true);
    this.pairError.set('');

    this.courseService.getPlagiarismPair(match.submission_id_a, match.submission_id_b).subscribe({
      next: pair => {
        this.pair.set(pair);
        this.isLoadingPair.set(false);
        if (!this.selectedPath()) {
          this.selectedPath.set(this.affectedFiles()[0]?.path ?? '');
        }
      },
      error: () => {
        this.pair.set(null);
        this.isLoadingPair.set(false);
        this.pairError.set('No se pudo cargar el par de entregas. Reintentá el escaneo.');
      }
    });
  }

  private fileContent(side: PlagiarismPairSide | undefined, path: string): string {
    if (!side) return '';
    if (this.isIde()) {
      for (const file of side.files ?? []) {
        if (file.path === path) return file.content;
      }
      if (path) return '';
    }
    return side.code ?? (side.files?.[0]?.content ?? '');
  }

  private fileLanguage(side: PlagiarismPairSide | undefined, path: string): string | null {
    if (!side || !path) return null;
    return side.files?.find(file => file.path === path)?.language ?? null;
  }

  private toLineRanges(sources: PlagiarismRange['source_a'][]): CodeLineRange[] {
    const path = this.selectedPath();
    return sources
      .filter(source => !this.isIde() || !path || !source.file_path || source.file_path === path)
      .map(source => ({ start: source.start_line, end: source.end_line }));
  }
}
