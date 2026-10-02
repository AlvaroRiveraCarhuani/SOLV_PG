import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import {
  LucideServer,
  LucideSearch,
  LucideCpu,
  LucideHardDrive,
  LucideLayers,
  LucidePlus,
  LucideBadgeCheck,
  LucideSparkles
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { TemplateRequestModalComponent } from './template-request-modal/template-request-modal.component';
import { ExerciseEditorModalComponent } from '../courses/exercise-editor/exercise-editor-modal.component';

export interface PublishedTemplate {
  id: string;
  name: string;
  description?: string;
  docker_image: string;
  default_memory_mb?: number;
  default_cpu_cores?: number;
  category?: string;
  environment_type?: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | string;
  is_official?: boolean;
  is_active?: boolean;
  ports?: string;
}

interface ApiResponse<T> {
  data: T;
  error?: string;
  message?: string;
}

@Component({
  selector: 'teacher-templates',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideServer,
    LucideSearch,
    LucideCpu,
    LucideHardDrive,
    LucideLayers,
    LucidePlus,
    LucideBadgeCheck,
    LucideSparkles,
    MachineDataDirective,
    SkeletonLoaderComponent,
    TemplateRequestModalComponent,
    ExerciseEditorModalComponent
  ],
  templateUrl: './teacher-templates.component.html',
  styleUrl: './teacher-templates.component.scss'
})
export class TeacherTemplatesComponent implements OnInit {
  private http = inject(HttpClient);
  private dashboardService = inject(TeacherDashboardService);

  courses = this.dashboardService.courses;
  templates = signal<PublishedTemplate[]>([]);
  isLoading = signal<boolean>(false);

  // Filtros reactivos
  categoryFilter = signal<string>('all');
  envTypeFilter = signal<string>('all');
  searchTerm = signal<string>('');

  // Modales
  isRequestModalOpen = signal<boolean>(false);
  isExerciseModalOpen = signal<boolean>(false);
  selectedTemplateForExercise = signal<PublishedTemplate | null>(null);

  // Computeds
  filteredTemplates = computed(() => {
    let items = this.templates();
    const category = this.categoryFilter();
    const envType = this.envTypeFilter();
    const query = this.searchTerm().trim().toLowerCase();

    if (category !== 'all') {
      items = items.filter(t => (t.category || '').toLowerCase() === category.toLowerCase());
    }

    if (envType !== 'all') {
      items = items.filter(t => t.environment_type === envType);
    }

    if (query) {
      items = items.filter(t =>
        t.name.toLowerCase().includes(query) ||
        (t.description && t.description.toLowerCase().includes(query)) ||
        t.docker_image.toLowerCase().includes(query)
      );
    }

    return items;
  });

  kpiStats = computed(() => {
    const list = this.templates();
    const total = list.length;
    const persistent = list.filter(t => t.environment_type === 'IDE_PERSISTENTE').length;
    const ephemeral = list.filter(t => t.environment_type === 'JUEZ_EFIMERO').length;
    const official = list.filter(t => t.is_official).length;

    return { total, persistent, ephemeral, official };
  });

  categories = computed(() => {
    const set = new Set<string>();
    this.templates().forEach(t => {
      if (t.category) set.add(t.category);
    });
    return Array.from(set);
  });

  ngOnInit(): void {
    this.loadTemplates();
  }

  loadTemplates(): void {
    this.isLoading.set(true);
    this.http.get<ApiResponse<PublishedTemplate[]>>('/api/v1/templates').subscribe({
      next: res => {
        const data = res.data || [];
        this.templates.set(data.filter(t => t.is_active !== false));
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  openRequestModal(): void {
    this.isRequestModalOpen.set(true);
  }

  closeRequestModal(): void {
    this.isRequestModalOpen.set(false);
  }

  onRequestSubmitted(): void {
    this.closeRequestModal();
    this.loadTemplates();
  }

  createLabFromTemplate(template: PublishedTemplate): void {
    this.selectedTemplateForExercise.set(template);
    this.isExerciseModalOpen.set(true);
  }

  closeExerciseModal(): void {
    this.isExerciseModalOpen.set(false);
    this.selectedTemplateForExercise.set(null);
  }
}
