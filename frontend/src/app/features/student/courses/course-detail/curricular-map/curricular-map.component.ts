import { Component, inject, input, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import {
  LucideLock,
  LucideBookOpen,
  LucideAlertCircle,
  LucideFileCode2,
  LucideChevronRight,
  LucideLayers
} from '@lucide/angular';
import { StudentService, CourseCurricularMap, CurricularModuleMap } from '@core/services/student.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

/**
 * Prácticas de una materia, ordenadas por el mapa curricular cuando el docente
 * definió módulos.
 *
 * El mapa y las prácticas viven en paneles separados: una materia sin módulos
 * no dibuja un estado vacío enorme antes de sus prácticas, y una materia con
 * módulos no mezcla el desbloqueo académico con los ejercicios sueltos.
 */
@Component({
  selector: 'curricular-map',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    MachineDataDirective,
    LucideLock,
    LucideBookOpen,
    LucideAlertCircle,
    LucideFileCode2,
    LucideChevronRight,
    LucideLayers
  ],
  templateUrl: './curricular-map.component.html',
  styleUrl: './curricular-map.component.scss'
})
export class CurricularMapComponent {
  private studentService = inject(StudentService);

  courseId = input.required<string>();
  /** El encabezado de la ficha incrementa este valor para releer el mapa. */
  refreshTick = input(0);

  mapData = signal<CourseCurricularMap | null>(null);
  isLoading = signal<boolean>(true);
  loadError = signal<string | null>(null);

  totalModules = computed(() => this.mapData()?.modules.length || 0);
  hasModules = computed(() => this.totalModules() > 0);

  unassignedExercises = computed(() => this.mapData()?.unassigned_exercises ?? []);

  completedModules = computed(() => {
    const mods = this.mapData()?.modules || [];
    return mods.filter(m => m.state === 'completed').length;
  });

  overallProgressPercent = computed(() => {
    const total = this.totalModules();
    if (total === 0) return 0;
    return Math.round((this.completedModules() / total) * 100);
  });

  constructor() {
    effect(() => {
      const id = this.courseId();
      this.refreshTick();
      if (!id) {
        return;
      }
      void this.loadCurricularMap(id);
    });
  }

  async loadCurricularMap(id: string): Promise<void> {
    this.isLoading.set(true);
    this.loadError.set(null);
    try {
      const data = await this.studentService.getCourseCurricularMap(id);
      this.mapData.set(data);
    } catch {
      this.loadError.set('No se pudieron leer las prácticas de esta materia.');
    } finally {
      this.isLoading.set(false);
    }
  }

  getModuleStateLabel(state: CurricularModuleMap['state']): string {
    switch (state) {
      case 'completed':
        return 'Completado';
      case 'in_progress':
        return 'En curso';
      case 'unlocked':
        return 'Disponible';
      case 'locked':
        return 'Bloqueado';
      default:
        return 'Pendiente';
    }
  }

  getDifficultyClass(diff?: string): string {
    switch (diff?.toLowerCase()) {
      case 'easy':
      case 'facil':
      case 'fácil':
        return 'diff-easy';
      case 'medium':
      case 'medio':
        return 'diff-medium';
      case 'hard':
      case 'dificil':
      case 'difícil':
        return 'diff-hard';
      default:
        return 'diff-default';
    }
  }

  getDifficultyLabel(diff?: string): string {
    switch (diff?.toLowerCase()) {
      case 'easy':
      case 'facil':
      case 'fácil':
        return 'Fácil';
      case 'medium':
      case 'medio':
        return 'Medio';
      case 'hard':
      case 'dificil':
      case 'difícil':
        return 'Difícil';
      default:
        return diff || 'General';
    }
  }

  getModulePassedCount(module: CurricularModuleMap): number {
    return module.exercises.filter(ex => (ex.best_score ?? 0) >= module.pass_score).length;
  }
}
