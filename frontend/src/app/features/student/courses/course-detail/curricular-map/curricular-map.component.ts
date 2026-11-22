import { Component, inject, input, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import {
  LucideAlertCircle,
  LucideBookOpen,
  LucideCheck,
  LucideFileCode2,
  LucideLaptop,
  LucideLock,
  LucideTerminal
} from '@lucide/angular';
import { StudentService, CourseCurricularMap, CurricularModuleMap, CurricularExercise } from '@core/services/student.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';

type CurricularFilterTab = 'all' | 'pending' | 'exam' | 'completed';

@Component({
  selector: 'curricular-map',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    MachineDataDirective,
    EnvironmentLabelPipe,
    LucideAlertCircle,
    LucideBookOpen,
    LucideCheck,
    LucideFileCode2,
    LucideLaptop,
    LucideLock,
    LucideTerminal
  ],
  templateUrl: './curricular-map.component.html',
  styleUrl: './curricular-map.component.scss'
})
export class CurricularMapComponent {
  private studentService = inject(StudentService);

  courseId = input.required<string>();
  refreshTick = input(0);

  mapData = signal<CourseCurricularMap | null>(null);
  isLoading = signal<boolean>(true);
  loadError = signal<string | null>(null);
  activeTab = signal<CurricularFilterTab>('all');

  totalModules = computed(() => this.mapData()?.modules.length || 0);
  hasModules = computed(() => this.totalModules() > 0);

  allExercisesList = computed<CurricularExercise[]>(() => {
    const data = this.mapData();
    if (!data) return [];
    const fromModules = data.modules.flatMap(m => m.exercises);
    const unassigned = data.unassigned_exercises || [];
    return [...fromModules, ...unassigned];
  });

  tabCounts = computed(() => {
    const list = this.allExercisesList();
    return {
      all: list.length,
      pending: list.filter(e => !this.isExerciseCompleted(e)).length,
      exam: list.filter(e => e.purpose === 'exam').length,
      completed: list.filter(e => this.isExerciseCompleted(e)).length
    };
  });

  completedModules = computed(() => {
    const mods = this.mapData()?.modules || [];
    return mods.filter(m => m.state === 'completed').length;
  });

  overallProgressPercent = computed(() => {
    const total = this.totalModules();
    if (total === 0) return 0;
    return Math.round((this.completedModules() / total) * 100);
  });

  unassignedExercises = computed(() => {
    const unassigned = this.mapData()?.unassigned_exercises ?? [];
    return this.filterExercises(unassigned);
  });

  constructor() {
    effect(() => {
      const id = this.courseId();
      this.refreshTick();
      if (!id) return;
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

  setTab(tab: CurricularFilterTab): void {
    this.activeTab.set(tab);
  }

  filterExercises(exercises: CurricularExercise[]): CurricularExercise[] {
    const tab = this.activeTab();
    if (tab === 'all') return exercises;
    if (tab === 'pending') return exercises.filter(e => !this.isExerciseCompleted(e));
    if (tab === 'exam') return exercises.filter(e => e.purpose === 'exam');
    if (tab === 'completed') return exercises.filter(e => this.isExerciseCompleted(e));
    return exercises;
  }

  isExerciseCompleted(ex: CurricularExercise): boolean {
    return (ex.best_score !== null && ex.best_score !== undefined && ex.best_score >= 60);
  }

  isExerciseInProgress(ex: CurricularExercise): boolean {
    return !this.isExerciseCompleted(ex) && ex.best_score !== null && ex.best_score !== undefined;
  }

  getModulePassedCount(module: CurricularModuleMap): number {
    return module.exercises.filter(ex => (ex.best_score ?? 0) >= module.pass_score).length;
  }

  isIdeExercise(ex: CurricularExercise): boolean {
    const item = ex as unknown as { environment_type?: string; type?: string };
    return item.environment_type === 'IDE_PERSISTENTE' || item.type === 'project';
  }

  getEnvironmentType(ex: CurricularExercise): string {
    const item = ex as unknown as { environment_type?: string };
    return item.environment_type || (this.isIdeExercise(ex) ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO');
  }
}
