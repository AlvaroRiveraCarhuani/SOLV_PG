import { Component, OnInit, inject, input, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import {
  LucideLock,
  LucideUnlock,
  LucideCheckCircle2,
  LucidePlay,
  LucideBookOpen,
  LucideAlertCircle,
  LucideFileCode2,
  LucideChevronRight,
  LucideRefreshCw,
  LucideSparkles,
  LucideLayers
} from '@lucide/angular';
import { StudentService, CourseCurricularMap, CurricularModuleMap, CurricularExercise } from '@core/services/student.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'curricular-map',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    MachineDataDirective,
    LucideLock,
    LucideUnlock,
    LucideCheckCircle2,
    LucidePlay,
    LucideBookOpen,
    LucideAlertCircle,
    LucideFileCode2,
    LucideChevronRight,
    LucideRefreshCw,
    LucideSparkles,
    LucideLayers
  ],
  templateUrl: './curricular-map.component.html',
  styleUrl: './curricular-map.component.scss'
})
export class CurricularMapComponent implements OnInit {
  private studentService = inject(StudentService);

  courseId = input.required<string>();

  mapData = signal<CourseCurricularMap | null>(null);
  isLoading = signal<boolean>(true);
  loadError = signal<string | null>(null);

  totalModules = computed(() => this.mapData()?.modules.length || 0);
  completedModules = computed(() => {
    const mods = this.mapData()?.modules || [];
    return mods.filter(m => m.state === 'completed').length;
  });

  overallProgressPercent = computed(() => {
    const total = this.totalModules();
    if (total === 0) return 100;
    return Math.round((this.completedModules() / total) * 100);
  });

  constructor() {
    effect(() => {
      const id = this.courseId();
      if (id) {
        this.loadCurricularMap(id);
      }
    });
  }

  ngOnInit(): void {
    const id = this.courseId();
    if (id && !this.mapData()) {
      this.loadCurricularMap(id);
    }
  }

  async loadCurricularMap(id: string): Promise<void> {
    this.isLoading.set(true);
    this.loadError.set(null);
    try {
      const data = await this.studentService.getCourseCurricularMap(id);
      this.mapData.set(data);
    } catch {
      this.loadError.set('No se pudo cargar el mapa curricular.');
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
