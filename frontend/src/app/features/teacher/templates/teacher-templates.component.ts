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
import { Router } from '@angular/router';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TemplateRequestModalComponent } from './template-request-modal/template-request-modal.component';

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
    ComboboxComponent,
    TemplateRequestModalComponent
  ],
  templateUrl: './teacher-templates.component.html',
  styleUrl: './teacher-templates.component.scss'
})
export class TeacherTemplatesComponent implements OnInit {
  private http = inject(HttpClient);
  private dashboardService = inject(TeacherDashboardService);
  private router = inject(Router);

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

  categoryComboboxOptions = computed<ComboboxOption[]>(() => {
    const opts: ComboboxOption[] = [
      { id: 'all', label: 'Todas las categorías', value: 'all' }
    ];
    this.categories().forEach(cat => {
      opts.push({ id: cat, label: cat, value: cat });
    });
    return opts;
  });

  selectedCategoryLabel = computed(() => {
    const sel = this.categoryFilter();
    if (sel === 'all') return 'Todas las categorías';
    return sel;
  });

  envTypeComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos los entornos', value: 'all' },
    { id: 'IDE_PERSISTENTE', label: 'IDE Persistente (OpenVSCode)', value: 'IDE_PERSISTENTE' },
    { id: 'JUEZ_EFIMERO', label: 'Juez Efímero (Evaluación)', value: 'JUEZ_EFIMERO' }
  ]);

  selectedEnvTypeLabel = computed(() => {
    const sel = this.envTypeFilter();
    if (sel === 'IDE_PERSISTENTE') return 'IDE Persistente (OpenVSCode)';
    if (sel === 'JUEZ_EFIMERO') return 'Juez Efímero (Evaluación)';
    return 'Todos los entornos';
  });

  onCategorySelected(opt: ComboboxOption): void {
    this.categoryFilter.set(opt.value);
  }

  onEnvTypeSelected(opt: ComboboxOption): void {
    this.envTypeFilter.set(opt.value);
  }

  ngOnInit(): void {
    this.loadTemplates();
  }

  loadTemplates(): void {
    this.isLoading.set(true);
    this.http.get<any>('/api/v1/templates').subscribe({
      next: res => {
        let items: any[] = [];
        if (Array.isArray(res)) {
          items = res;
        } else if (Array.isArray(res?.data)) {
          items = res.data;
        } else if (Array.isArray(res?.data?.data)) {
          items = res.data.data;
        }

        const mapped: PublishedTemplate[] = items.map((item: any) => ({
          id: item.id || '',
          name: item.name || '',
          description: item.description || '',
          docker_image: item.docker_image || '',
          default_memory_mb: item.default_memory_mb || item.base_ram_mb || 512,
          default_cpu_cores: item.default_cpu_cores || 1,
          category: item.category || item.category_name || 'General',
          environment_type: item.environment_type || item.target_environment || 'IDE_PERSISTENTE',
          is_official: item.is_official ?? true,
          is_active: item.is_active !== false,
          ports: item.ports || ''
        }));

        this.templates.set(mapped.filter(t => t.is_active !== false));
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
    const coursesList = this.courses();
    const courseId = coursesList.length > 0 ? coursesList[0].id : 'all';
    this.router.navigate(['/teacher/courses', courseId, 'exercises', 'new'], {
      queryParams: { template_id: template.id }
    });
  }

  closeExerciseModal(): void {
    this.isExerciseModalOpen.set(false);
    this.selectedTemplateForExercise.set(null);
  }
}
