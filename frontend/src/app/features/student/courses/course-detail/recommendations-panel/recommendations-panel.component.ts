import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideLightbulb,
  LucideChevronDown,
  LucideChevronUp,
  LucideSparkles,
  LucideCheckCircle2,
  LucideInfo,
  LucideArrowRight,
  LucideTag
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  StudentService,
  StudentRecommendationsDTO,
  RecommendationItem,
  WeakTag
} from '@core/services/student.service';

@Component({
  selector: 'recommendations-panel',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideLightbulb,
    LucideChevronDown,
    LucideChevronUp,
    LucideSparkles,
    LucideCheckCircle2,
    LucideInfo,
    LucideArrowRight,
    LucideTag,
    MachineDataDirective
  ],
  templateUrl: './recommendations-panel.component.html',
  styleUrl: './recommendations-panel.component.scss'
})
export class RecommendationsPanelComponent implements OnInit {
  private studentService = inject(StudentService);

  courseId = input.required<string>();

  data = signal<StudentRecommendationsDTO | null>(null);
  loading = signal<boolean>(true);
  isCollapsed = signal<boolean>(false);

  hasEnoughData = computed(() => this.data()?.has_enough_data ?? false);
  weakTags = computed(() => this.data()?.weak_tags ?? []);
  recommendations = computed(() => this.data()?.recommendations ?? []);
  message = computed(() => this.data()?.message ?? '');

  hasRecommendations = computed(() => this.recommendations().length > 0);

  ngOnInit(): void {
    this.loadRecommendations();
  }

  async loadRecommendations(): Promise<void> {
    const id = this.courseId();
    if (!id) {
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    try {
      const res = await this.studentService.getCourseRecommendations(id);
      this.data.set(res);
    } catch {
      this.data.set({
        has_enough_data: false,
        weak_tags: [],
        recommendations: [],
        message: 'No se pudieron cargar las recomendaciones'
      });
    } finally {
      this.loading.set(false);
    }
  }

  toggleCollapse(): void {
    this.isCollapsed.update((collapsed) => !collapsed);
  }

  getDifficultyClass(difficulty: string): string {
    switch (difficulty?.toLowerCase()) {
      case 'easy':
      case 'facil':
      case 'fácil':
        return 'difficulty-easy';
      case 'medium':
      case 'medio':
        return 'difficulty-medium';
      case 'hard':
      case 'dificil':
      case 'difícil':
        return 'difficulty-hard';
      default:
        return 'difficulty-easy';
    }
  }

  getDifficultyLabel(difficulty: string): string {
    switch (difficulty?.toLowerCase()) {
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
        return difficulty || 'Fácil';
    }
  }
}
