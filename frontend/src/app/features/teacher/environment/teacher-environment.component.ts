import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { LucideExternalLink, LucidePause } from '@lucide/angular';

import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherEnvironmentService } from '../services/teacher-environment.service';

/**
 * Pantalla "Mi entorno de clase": el docente abre su entorno en una materia y
 * escribe ahí el código que después muestra en el aula. El avance del aula sale
 * de este entorno (ADR-058 §5), así que sin él abierto no hay nada que emitir.
 *
 * La pantalla no levanta nada al cargar: primero lee si el entorno ya existe y
 * solo abre cuando el docente lo pide.
 */
@Component({
  selector: 'teacher-environment',
  standalone: true,
  imports: [
    CommonModule,
    ComboboxComponent,
    StatusBadgeComponent,
    MachineDataDirective,
    LucideExternalLink,
    LucidePause
  ],
  templateUrl: './teacher-environment.component.html',
  styleUrl: './teacher-environment.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeacherEnvironmentComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly environmentService = inject(TeacherEnvironmentService);
  private readonly sanitizer = inject(DomSanitizer);

  readonly courses = this.dashboardService.courses;
  readonly environment = this.environmentService.environment;
  readonly isLoading = this.environmentService.isLoading;
  readonly isStarting = this.environmentService.isStarting;
  readonly error = this.environmentService.error;

  readonly subjectId = signal<string>('');

  readonly subjectOptions = computed<ComboboxOption[]>(() =>
    this.courses().map(course => ({
      id: course.id,
      label: `${course.code} · ${course.name}`,
      value: course.id
    }))
  );

  readonly subject = computed(() => this.courses().find(course => course.id === this.subjectId()));

  readonly isRunning = computed(() => this.environment()?.status === 'running');

  /** URL directa del editor, para abrirlo fuera del marco cuando estorba. */
  readonly rawUrl = computed(() => {
    const environment = this.environment();
    return environment ? environment.access_url || environment.url || '' : '';
  });

  readonly frameUrl = computed<SafeResourceUrl | null>(() => {
    const environment = this.environment();
    if (!environment || environment.status !== 'running') {
      return null;
    }
    const url = environment.access_url || environment.url || '';
    return url ? this.sanitizer.bypassSecurityTrustResourceUrl(url) : null;
  });

  /** Abrir y reanudar son la misma llamada; el texto dice qué va a pasar. */
  readonly openLabel = computed(() => {
    switch (this.environment()?.status) {
      case 'hibernated':
        return 'Reanudar entorno';
      case 'failed':
      case 'oom_killed':
        return 'Volver a abrir';
      default:
        return 'Abrir entorno';
    }
  });

  readonly showOpenButton = computed(() => !!this.subjectId() && !this.isRunning());

  readonly emptyTitle = computed(() => {
    const environment = this.environment();
    if (!this.subjectId()) {
      return 'Elegí una materia';
    }
    if (environment?.status === 'hibernated') {
      return 'El entorno está en pausa';
    }
    if (environment?.status === 'failed' || environment?.status === 'oom_killed') {
      return 'El entorno se detuvo';
    }
    if (environment?.status === 'pending') {
      return 'El entorno se está preparando';
    }
    return 'Todavía no abriste un entorno en esta materia';
  });

  readonly emptyCopy = computed(() => {
    const environment = this.environment();
    if (!this.subjectId()) {
      return 'Tu entorno de clase es por materia: se abre sobre el volumen de esa materia y desde ahí sale el código que mostrás en el aula.';
    }
    if (environment?.status === 'hibernated') {
      return 'Reanudalo para seguir escribiendo. Lo que dejaste en el editor sigue guardado en el volumen.';
    }
    if (environment?.status === 'failed' || environment?.status === 'oom_killed') {
      return 'Volvé a abrirlo. El volumen de la materia con tu código no se borra al detenerse el contenedor.';
    }
    if (environment?.status === 'pending') {
      return 'Puede tardar unos segundos. Cuando esté listo, el editor aparece acá mismo.';
    }
    return 'Al abrirlo levanta un editor sobre el volumen de la materia. Podés pausarlo al terminar y retomarlo en la próxima clase.';
  });

  ngOnInit(): void {
    this.dashboardService.getCourses().subscribe({
      next: courses => {
        // La URL trae la materia desde el aula o desde el panel; una que no es
        // del docente cae al estado sin selección, sin tratarse como permiso.
        const preselected = this.route.snapshot.queryParamMap.get('subject_id') ?? '';
        if (preselected && courses.some(course => course.id === preselected)) {
          this.selectSubject(preselected, false);
        }
      },
      error: () => undefined
    });
  }

  onSubjectChange(subjectId: string): void {
    this.selectSubject(subjectId, true);
  }

  open(): void {
    const subjectId = this.subjectId();
    if (!subjectId || this.isStarting()) {
      return;
    }
    this.environmentService.open(subjectId).subscribe({ error: () => undefined });
  }

  pause(): void {
    const environment = this.environment();
    if (!environment || this.isStarting()) {
      return;
    }
    this.environmentService.pause(environment.id).subscribe({ error: () => undefined });
  }

  /** Cambiar de materia deja la URL compartible y lee el entorno de la nueva. */
  private selectSubject(subjectId: string, syncUrl: boolean): void {
    this.subjectId.set(subjectId);
    if (syncUrl) {
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: subjectId ? { subject_id: subjectId } : {},
        replaceUrl: true
      });
    }
    if (!subjectId) {
      this.environmentService.environment.set(null);
      return;
    }
    this.environmentService.load(subjectId).subscribe({ error: () => undefined });
  }
}
