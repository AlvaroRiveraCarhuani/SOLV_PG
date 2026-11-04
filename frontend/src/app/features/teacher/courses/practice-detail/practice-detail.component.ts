import { Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Subscription, forkJoin, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import {
  LucideAlertTriangle,
  LucideArrowLeft,
  LucideCalendar,
  LucideCheckCircle,
  LucideClock,
  LucideCpu,
  LucideDatabase,
  LucideEdit3,
  LucideLaptop,
  LucideListChecks,
  LucideLoader2,
  LucideRotateCcw,
  LucideShieldAlert,
  LucideSparkles,
  LucideTerminal
} from '@lucide/angular';

import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherEnvironmentService } from '../../services/teacher-environment.service';
import { PracticeEnvTestService } from '../../services/practice-env-test.service';
import { PracticeEnvTestJob } from '../../models/practice-env-test.models';
import { ExerciseDetailDTO, TeacherLabStats, TestCaseDTO } from '../../models/teacher.models';
import type { RubricDTO } from '../../exercise-editor/exercise-editor.store';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { FuzzingModalComponent } from '../../evaluations/fuzzing-modal/fuzzing-modal.component';

interface ApiResponse<T> {
  data: T;
}

/**
 * Detalle de una práctica. El mismo documento sirve a las dos modalidades:
 * el juez automático muestra sus casos y el IDE persistente su plantilla y su
 * rúbrica. Todo sale del ejercicio y de las métricas que ya expone la materia.
 */
@Component({
  selector: 'practice-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    CodeEditorComponent,
    LucideAlertTriangle,
    LucideArrowLeft,
    LucideCalendar,
    LucideCheckCircle,
    LucideClock,
    LucideCpu,
    LucideDatabase,
    LucideEdit3,
    LucideLaptop,
    LucideListChecks,
    LucideLoader2,
    LucideRotateCcw,
    LucideShieldAlert,
    LucideSparkles,
    LucideTerminal,
    FuzzingModalComponent
  ],
  templateUrl: './practice-detail.component.html',
  styleUrl: './practice-detail.component.scss'
})
export class PracticeDetailComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private courseService = inject(TeacherCourseService);
  private practiceEnvTest = inject(PracticeEnvTestService);
  private environment = inject(TeacherEnvironmentService);
  private http = inject(HttpClient);
  private envTestPoll?: Subscription;

  courseId = signal<string>('');
  exercise = signal<ExerciseDetailDTO | null>(null);
  labStats = signal<TeacherLabStats | null>(null);
  rubric = signal<RubricDTO | null>(null);
  isLoading = signal<boolean>(true);
  loadError = signal<string | null>(null);
  selectedCaseIndex = signal<number>(0);
  selectedFuzzExercise = signal<{ id: string; title: string } | null>(null);
  envTestJob = signal<PracticeEnvTestJob | null>(null);
  isEnvTestRunning = signal<boolean>(false);
  envTestError = signal<string | null>(null);
  isOpeningIde = signal<boolean>(false);
  ideError = signal<string | null>(null);

  isIde = computed(() => this.exercise()?.environment_type === 'IDE_PERSISTENTE');
  isJudge = computed(() => !this.isIde());

  testCases = computed<TestCaseDTO[]>(() => {
    const exercise = this.exercise();
    return exercise?.test_cases ?? exercise?.config?.algorithm?.test_cases ?? [];
  });

  activeCase = computed<TestCaseDTO | null>(
    () => this.testCases()[this.selectedCaseIndex()] ?? null
  );

  casesTotalPoints = computed(() =>
    this.testCases().reduce((total, testCase) => total + (testCase.weight ?? 0), 0)
  );

  rubricTotal = computed(() =>
    (this.rubric()?.criteria ?? []).reduce((total, criterion) => total + (criterion.weight ?? 0), 0)
  );

  templateName = computed(() => this.exercise()?.template?.name ?? null);
  dockerImage = computed(() => this.exercise()?.template?.docker_image ?? null);
  dbEngine = computed(() => this.exercise()?.config?.database?.engine ?? null);
  hasDatabase = computed(() => {
    const config = this.exercise()?.config;
    return Boolean(
      config?.database?.engine ||
        config?.database?.init_script ||
        config?.ide?.db_init_script
    );
  });
  allowResubmit = computed<boolean | null>(
    () => this.exercise()?.config?.ide?.allow_resubmit ?? null
  );
  hasDbInitScript = computed(() => {
    const config = this.exercise()?.config;
    return Boolean(config?.database?.init_script || config?.ide?.db_init_script);
  });
  hasLimits = computed(() => {
    if (this.isIde()) {
      return false;
    }
    const exercise = this.exercise();
    return Boolean(exercise?.time_limit_ms || exercise?.memory_limit_mb);
  });
  language = computed(
    () => this.exercise()?.language ?? this.exercise()?.template?.language ?? null
  );

  envTestTone = computed<'neutral' | 'running' | 'success' | 'error'>(() => {
    if (this.isEnvTestRunning()) {
      return 'running';
    }
    const status = this.envTestJob()?.status;
    if (status === 'success') {
      return 'success';
    }
    if (status === 'failed') {
      return 'error';
    }
    return 'neutral';
  });

  envTestTitle = computed(() => {
    if (this.isEnvTestRunning()) {
      return 'Prueba de entorno en curso';
    }
    const job = this.envTestJob();
    if (!job) {
      return 'Sin pruebas de entorno';
    }
    switch (job.status) {
      case 'success':
        return 'Entorno listo';
      case 'failed':
        return 'La prueba falló';
      case 'canceled':
        return 'Prueba cancelada';
      case 'pending':
        return 'Prueba en cola';
      case 'pulling':
        return 'Descargando la imagen';
      case 'testing':
        return 'Validando el entorno';
      default:
        return 'Sin pruebas de entorno';
    }
  });

  envTestDetail = computed(() => {
    const job = this.envTestJob();
    if (!job) {
      return 'Ejecuta una prueba para validar el repositorio base, las dependencias y los satélites antes de la clase.';
    }
    if (this.isEnvTestRunning()) {
      return job.progress?.current_action || 'Preparando la prueba…';
    }
    if (job.status === 'failed') {
      return job.error_message || 'La prueba no pudo completarse.';
    }
    if (job.status === 'canceled') {
      return 'La prueba se canceló antes de terminar.';
    }
    if (job.status === 'success') {
      return 'Repositorio base, dependencias y satélites verificados.';
    }
    return job.progress?.current_action || '';
  });

  ngOnInit(): void {
    const courseId = this.route.snapshot.paramMap.get('courseId') ?? '';
    const exerciseId = this.route.snapshot.paramMap.get('id') ?? '';
    this.courseId.set(courseId);

    if (!exerciseId) {
      this.isLoading.set(false);
      this.loadError.set('No se indicó la práctica a mostrar.');
      return;
    }

    forkJoin({
      exercise: this.courseService.getExercise(exerciseId),
      labs: courseId
        ? this.courseService
            .getCourseLabs(courseId)
            .pipe(catchError(() => of([] as TeacherLabStats[])))
        : of([] as TeacherLabStats[])
    }).subscribe({
      next: ({ exercise, labs }) => {
        this.exercise.set(exercise);
        this.labStats.set(labs.find(lab => lab.id === exerciseId) ?? null);
        this.isLoading.set(false);
        this.loadRubric(exerciseId);
        if (exercise.environment_type === 'IDE_PERSISTENTE') {
          this.loadLatestEnvTest(exerciseId);
        }
      },
      error: () => {
        this.isLoading.set(false);
        this.loadError.set('No se pudo cargar la práctica. Revisa que siga existiendo en la materia.');
      }
    });
  }

  selectCase(index: number): void {
    this.selectedCaseIndex.set(index);
  }

  /**
   * Abre el IDE de la práctica en una pestaña nueva. La pestaña se abre antes de
   * pedir el entorno, con la respuesta ya en camino: así el navegador no la toma
   * por ventana emergente y no queda una pestaña en blanco si la apertura falla.
   */
  openIde(): void {
    const practice = this.exercise();
    if (!practice || this.isOpeningIde()) {
      return;
    }

    const tab = window.open('', '_blank');
    this.ideError.set(null);
    this.isOpeningIde.set(true);

    this.environment.openPracticeIDE(this.courseId(), practice.id).subscribe({
      next: session => {
        this.isOpeningIde.set(false);
        if (!session?.access_url) {
          tab?.close();
          this.ideError.set('El entorno arrancó pero todavía no tiene una URL de acceso.');
          return;
        }
        if (tab) {
          tab.opener = null;
          tab.location.href = session.access_url;
          return;
        }
        // Sin pestaña disponible (el navegador bloqueó la ventana emergente), el
        // entorno ya está corriendo: se abre en esta misma pestaña.
        window.location.assign(session.access_url);
      },
      error: err => {
        this.isOpeningIde.set(false);
        tab?.close();
        this.ideError.set(this.environment.describePracticeIDEError(err));
      }
    });
  }

  startEnvTest(): void {
    const exercise = this.exercise();
    if (!exercise || this.isEnvTestRunning()) {
      return;
    }

    this.envTestError.set(null);
    this.isEnvTestRunning.set(true);
    this.practiceEnvTest.start(this.courseId(), exercise.id).subscribe({
      next: job => {
        this.envTestJob.set(job);
        this.pollEnvTest(job.id);
      },
      error: err => {
        this.isEnvTestRunning.set(false);
        this.envTestError.set(err?.error?.message || 'No se pudo iniciar la prueba de entorno.');
      }
    });
  }

  cancelEnvTest(): void {
    const job = this.envTestJob();
    if (!job) {
      return;
    }

    this.practiceEnvTest.cancelJob(job.id).subscribe({
      next: () => {
        this.stopEnvTestPoll();
        this.isEnvTestRunning.set(false);
        this.envTestJob.update(current =>
          current ? { ...current, status: 'canceled' } : current
        );
      },
      error: () => this.envTestError.set('No se pudo cancelar la prueba de entorno.')
    });
  }

  openFuzzingModal(): void {
    const exercise = this.exercise();
    if (!exercise) {
      return;
    }
    this.selectedFuzzExercise.set({ id: exercise.id, title: exercise.title });
  }

  closeFuzzingModal(): void {
    this.selectedFuzzExercise.set(null);
  }

  onFuzzCasesApplied(): void {
    const exerciseId = this.exercise()?.id;
    if (!exerciseId) {
      return;
    }

    this.courseService
      .getExercise(exerciseId)
      .pipe(catchError(() => of(null)))
      .subscribe(exercise => {
        if (!exercise) {
          return;
        }
        this.exercise.set(exercise);
        const lastIndex = this.testCases().length - 1;
        if (this.selectedCaseIndex() > lastIndex) {
          this.selectedCaseIndex.set(Math.max(lastIndex, 0));
        }
      });
  }

  caseTitle(testCase: TestCaseDTO, index: number): string {
    const description = testCase.description?.trim();
    return description ? description : `Caso ${index + 1}`;
  }

  caseVisibility(testCase: TestCaseDTO): 'Público' | 'Oculto' | 'Ejemplo' {
    if (testCase.visibility === 'example' || testCase.is_sample) {
      return 'Ejemplo';
    }
    if (testCase.visibility === 'hidden' || testCase.is_hidden) {
      return 'Oculto';
    }
    return 'Público';
  }

  isHiddenCase(testCase: TestCaseDTO): boolean {
    return this.caseVisibility(testCase) === 'Oculto';
  }

  casePoints(testCase: TestCaseDTO): string {
    const weight = testCase.weight ?? 0;
    return weight > 0 ? `${weight} pts` : 'Sin puntaje';
  }

  rubricShare(weight: number): number {
    const total = this.rubricTotal();
    if (total <= 0) {
      return 0;
    }
    return Math.round((weight / total) * 100);
  }

  private loadLatestEnvTest(exerciseId: string): void {
    this.practiceEnvTest
      .latest(exerciseId)
      .pipe(catchError(() => of(null)))
      .subscribe(job => {
        if (!job) {
          return;
        }
        this.envTestJob.set(job);
        if (!this.practiceEnvTest.isTerminal(job.status)) {
          this.pollEnvTest(job.id);
        }
      });
  }

  private pollEnvTest(jobId: string): void {
    this.stopEnvTestPoll();
    this.isEnvTestRunning.set(true);
    this.envTestPoll = this.practiceEnvTest.pollJob(jobId).subscribe({
      next: job => {
        this.envTestJob.set(job);
        if (this.practiceEnvTest.isTerminal(job.status)) {
          this.isEnvTestRunning.set(false);
        }
      },
      error: () => {
        this.isEnvTestRunning.set(false);
        this.envTestError.set('Se perdió el estado de la prueba de entorno.');
      }
    });
  }

  private stopEnvTestPoll(): void {
    this.envTestPoll?.unsubscribe();
    this.envTestPoll = undefined;
  }

  ngOnDestroy(): void {
    this.stopEnvTestPoll();
  }

  private loadRubric(exerciseId: string): void {
    this.http
      .get<ApiResponse<RubricDTO> | RubricDTO>(`/api/v1/exercises/${exerciseId}/rubric`)
      .pipe(
        map(res =>
          (res && typeof res === 'object' && 'data' in res
            ? (res as ApiResponse<RubricDTO>).data
            : res) as RubricDTO | null
        ),
        catchError(() => of(null))
      )
      .subscribe(rubric => {
        this.rubric.set(rubric?.criteria?.length ? rubric : null);
      });
  }
}
