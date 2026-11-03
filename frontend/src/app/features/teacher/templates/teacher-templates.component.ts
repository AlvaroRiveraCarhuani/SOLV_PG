import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import {
  LucideSearch,
  LucideLayers,
  LucidePlus,
  LucideBadgeCheck,
  LucideSparkles,
  LucideInfo,
  LucideUser,
  LucideBookOpen,
  LucideArrowRight,
  LucideInbox,
  LucideWand2
} from '@lucide/angular';
import { Router } from '@angular/router';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TemplateRequestModalComponent } from './template-request-modal/template-request-modal.component';
import { TemplateDetailModalComponent } from './template-detail-modal/template-detail-modal.component';
import { EnvironmentRequestService } from '../services/environment-request.service';
import {
  ENVIRONMENT_REQUEST_STATUS_LABEL,
  ENVIRONMENT_REQUEST_STATUS_TONE,
  EnvironmentRequest,
  EnvironmentRequestStackConfig
} from './environment-request.models';

export interface TechChip {
  name: string;
  iconKey?: string;
}

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
  primaryIcon?: string;
  techChips?: TechChip[];
  /** Piezas que componen el entorno, para arrancar una variante ya armada. */
  stack_config?: EnvironmentRequestStackConfig | null;
}

export interface RawTemplateItem {
  id?: string;
  name?: string;
  description?: string;
  docker_image?: string;
  default_memory_mb?: number;
  base_ram_mb?: number;
  default_cpu_cores?: number;
  category?: string;
  category_name?: string;
  environment_type?: string;
  target_environment?: string;
  is_official?: boolean;
  is_active?: boolean;
  ports?: string;
  tools_declared?: string[];
  stack_config?: EnvironmentRequestStackConfig | null;
}

export type TemplateApiResponse =
  | RawTemplateItem[]
  | { data?: RawTemplateItem[] | { data?: RawTemplateItem[] } };

export function resolveTemplateTechInfo(name: string, dockerImage: string, description: string = ''): {
  primaryIcon: string;
  techChips: TechChip[];
  defaultCategory: string;
  defaultDescription: string;
} {
  const text = `${name} ${dockerImage} ${description}`.toLowerCase();
  const chips: TechChip[] = [];
  let primary = 'docker';
  let defaultCategory = 'General';
  let defaultDescription = 'Entorno optimizado para prácticas y proyectos de laboratorio.';

  if (text.includes('java') || text.includes('temurin') || text.includes('spring') || text.includes('openjdk')) {
    primary = 'java';
    defaultCategory = 'Empresarial & Backend';
    defaultDescription = 'Desarrollo de aplicaciones empresariales, programación orientada a objetos y servicios REST con Spring Boot.';
    chips.push({ name: 'Java', iconKey: 'java' });
    if (text.includes('spring')) chips.push({ name: 'Spring Boot', iconKey: 'spring' });
    if (text.includes('maven')) chips.push({ name: 'Maven', iconKey: 'maven' });
  } else if (text.includes('python') || text.includes('django') || text.includes('fastapi') || text.includes('flask')) {
    primary = 'python';
    defaultCategory = 'Ciencia de Datos & IA';
    defaultDescription = 'Análisis de datos, computación numérica, desarrollo web y fundamentos de inteligencia artificial.';
    chips.push({ name: 'Python', iconKey: 'python' });
    if (text.includes('fastapi')) chips.push({ name: 'FastAPI', iconKey: 'fastapi' });
    if (text.includes('django')) chips.push({ name: 'Django', iconKey: 'django' });
    if (text.includes('flask')) chips.push({ name: 'Flask', iconKey: 'flask' });
    if (text.includes('pandas') || text.includes('numpy') || text.includes('data science') || text.includes('ia')) {
      chips.push({ name: 'Data Science', iconKey: 'pandas' });
    }
  } else if (text.includes('c++') || text.includes('g++') || text.includes('gcc') || text.includes('c/c++') || text.includes('clang')) {
    primary = 'cpp';
    defaultCategory = 'Sistemas & Computación';
    defaultDescription = 'Programación de sistemas, bajo nivel, memoria dinámica y concurrencia con compilador GCC.';
    chips.push({ name: 'C / C++', iconKey: 'cpp' });
    chips.push({ name: 'GCC / Clang', iconKey: 'c' });
  } else if (text.includes('rust') || text.includes('cargo')) {
    primary = 'rust';
    defaultCategory = 'Sistemas & Computación';
    defaultDescription = 'Desarrollo de software seguro en memoria, concurrencia moderna y WebAssembly.';
    chips.push({ name: 'Rust', iconKey: 'rust' });
    if (text.includes('wasm') || text.includes('webassembly')) chips.push({ name: 'WebAssembly', iconKey: 'rust' });
  } else if (text.includes('golang') || text.includes('go 1.') || text.includes('go micro')) {
    primary = 'go';
    defaultCategory = 'Desarrollo Web & Cloud';
    defaultDescription = 'Construcción de microservicios de alto rendimiento, concurrencia nativa y gRPC.';
    chips.push({ name: 'Go', iconKey: 'go' });
    if (text.includes('grpc')) chips.push({ name: 'gRPC', iconKey: 'go' });
  } else if (text.includes('node') || text.includes('typescript') || text.includes('javascript')) {
    primary = 'node';
    defaultCategory = 'Desarrollo Web & Cloud';
    defaultDescription = 'Desarrollo Full Stack con JavaScript y TypeScript, APIs REST y utilidades npm.';
    chips.push({ name: 'Node.js', iconKey: 'node' });
    if (text.includes('typescript')) chips.push({ name: 'TypeScript', iconKey: 'typescript' });
  } else if (text.includes('php') || text.includes('laravel')) {
    primary = 'php';
    defaultCategory = 'Desarrollo Web & Cloud';
    defaultDescription = 'Desarrollo de aplicaciones web dinámicas y servicios backend tradicionales.';
    chips.push({ name: 'PHP', iconKey: 'php' });
    if (text.includes('laravel')) chips.push({ name: 'Laravel', iconKey: 'laravel' });
  } else if (text.includes('postgres') || text.includes('psql')) {
    primary = 'postgres';
    defaultCategory = 'Bases de Datos';
    defaultDescription = 'Motor de base de datos relacional para consultas SQL, procedimientos y modelado relacional.';
    chips.push({ name: 'PostgreSQL', iconKey: 'postgres' });
  } else if (text.includes('mysql') || text.includes('mariadb')) {
    primary = 'mysql';
    defaultCategory = 'Bases de Datos';
    defaultDescription = 'Motor de base de datos relacional para prácticas de SQL y persistencia estructurada.';
    chips.push({ name: 'MySQL', iconKey: 'mysql' });
  } else if (text.includes('dotnet') || text.includes('c#')) {
    primary = 'dotnet';
    defaultCategory = 'Empresarial & Backend';
    defaultDescription = 'Desarrollo en .NET y C# para aplicaciones empresariales y APIs multiplataforma.';
    chips.push({ name: '.NET / C#', iconKey: 'dotnet' });
  }

  if (chips.length === 0) {
    chips.push({ name: 'Entorno Estándar', iconKey: 'docker' });
  }

  return { primaryIcon: primary, techChips: chips, defaultCategory, defaultDescription };
}

@Component({
  selector: 'teacher-templates',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSearch,
    LucideLayers,
    LucidePlus,
    LucideBadgeCheck,
    LucideSparkles,
    LucideInfo,
    LucideUser,
    LucideBookOpen,
    LucideArrowRight,
    LucideInbox,
    LucideWand2,
    SkeletonLoaderComponent,
    ComboboxComponent,
    EnvironmentLabelPipe,
    DateTextPipe,
    TechLogoComponent,
    ModalShellComponent,
    TemplateRequestModalComponent,
    TemplateDetailModalComponent
  ],
  templateUrl: './teacher-templates.component.html',
  styleUrl: './teacher-templates.component.scss'
})
export class TeacherTemplatesComponent implements OnInit {
  private http = inject(HttpClient);
  private dashboardService = inject(TeacherDashboardService);
  private environmentRequestService = inject(EnvironmentRequestService);
  private router = inject(Router);

  courses = this.dashboardService.courses;
  templates = signal<PublishedTemplate[]>([]);
  isLoading = signal<boolean>(false);

  // Filtros reactivos
  categoryFilter = signal<string>('all');
  envTypeFilter = signal<string>('all');
  searchTerm = signal<string>('');

  // Pestañas: catálogo de entornos y las solicitudes propias del docente
  activeTab = signal<'catalog' | 'requests'>('catalog');
  requests = signal<EnvironmentRequest[]>([]);
  requestsLoading = signal<boolean>(false);
  requestPreset = signal<string>('');
  requestPresetStack = signal<EnvironmentRequestStackConfig | null>(null);

  readonly pendingRequestsCount = computed(
    () => this.requests().filter(item => item.status === 'EN_REVISION').length
  );

  // Modales
  isRequestModalOpen = signal<boolean>(false);
  isCoursePickerOpen = signal<boolean>(false);
  targetTemplateForCourse = signal<PublishedTemplate | null>(null);
  selectedTemplateForDetail = signal<PublishedTemplate | null>(null);

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
        (t.docker_image && t.docker_image.toLowerCase().includes(query)) ||
        (t.techChips && t.techChips.some(c => c.name.toLowerCase().includes(query)))
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
    const base: ComboboxOption[] = [{ id: 'all', label: 'Todas las categorías', value: 'all' }];
    const cats = this.categories().map(cat => ({
      id: cat,
      label: cat,
      value: cat
    }));
    return [...base, ...cats];
  });

  selectedCategoryLabel = computed<string>(() => {
    const current = this.categoryFilter();
    if (current === 'all') return 'Todas las categorías';
    return current;
  });

  envTypeComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todas las modalidades', value: 'all' },
    { id: 'IDE_PERSISTENTE', label: 'Entorno de Desarrollo', value: 'IDE_PERSISTENTE' },
    { id: 'JUEZ_EFIMERO', label: 'Entorno de Evaluación', value: 'JUEZ_EFIMERO' }
  ]);

  selectedEnvTypeLabel = computed<string>(() => {
    const current = this.envTypeFilter();
    if (current === 'all') return 'Todas las modalidades';
    if (current === 'IDE_PERSISTENTE') return 'Entorno de Desarrollo';
    if (current === 'JUEZ_EFIMERO') return 'Entorno de Evaluación';
    return current;
  });

  onCategorySelected(opt: ComboboxOption): void {
    this.categoryFilter.set(opt.value);
  }

  onEnvTypeSelected(opt: ComboboxOption): void {
    this.envTypeFilter.set(opt.value);
  }

  ngOnInit(): void {
    this.loadTemplates();
    this.loadRequests();
  }

  setTab(tab: 'catalog' | 'requests'): void {
    this.activeTab.set(tab);
    if (tab === 'requests') {
      this.loadRequests();
    }
  }

  loadRequests(): void {
    this.requestsLoading.set(true);
    this.environmentRequestService.listMine().subscribe({
      next: items => {
        this.requests.set(items || []);
        this.requestsLoading.set(false);
      },
      error: () => {
        this.requests.set([]);
        this.requestsLoading.set(false);
      }
    });
  }

  statusLabel(status: string): string {
    return ENVIRONMENT_REQUEST_STATUS_LABEL[status as EnvironmentRequest['status']] || status;
  }

  statusTone(status: string): string {
    return ENVIRONMENT_REQUEST_STATUS_TONE[status as EnvironmentRequest['status']] || 'neutral';
  }

  environmentLabel(environment: string): string {
    if (environment === 'AMBOS') return 'Desarrollo y evaluación';
    if (environment === 'JUEZ_EFIMERO') return 'Evaluación';
    return 'Desarrollo';
  }

  /** Pide una variante de un entorno que el docente ya conoce. */
  requestVariant(template: PublishedTemplate): void {
    this.requestPreset.set(
      `Quiero un entorno como «${template.name}», pero con…`
    );
    this.requestPresetStack.set(template.stack_config || null);
    this.isRequestModalOpen.set(true);
  }

  loadTemplates(): void {
    this.isLoading.set(true);
    this.http.get<TemplateApiResponse>('/api/v1/templates').subscribe({
      next: res => {
        let items: RawTemplateItem[] = [];
        if (Array.isArray(res)) {
          items = res;
        } else if (Array.isArray(res?.data)) {
          items = res.data;
        } else if (res?.data && typeof res.data === 'object' && 'data' in res.data && Array.isArray(res.data.data)) {
          items = res.data.data;
        }

        const mapped: PublishedTemplate[] = items.map((item: RawTemplateItem) => {
          const name = item.name || '';
          const dockerImage = item.docker_image || '';
          const rawDesc = item.description || '';
          const techInfo = resolveTemplateTechInfo(name, dockerImage, rawDesc);
          const desc = rawDesc.trim() ? rawDesc.trim() : techInfo.defaultDescription;
          const rawCat = (item.category || item.category_name || '').trim();
          const cat = (rawCat && rawCat !== 'General') ? rawCat : techInfo.defaultCategory;

          return {
            id: item.id || '',
            name,
            description: desc,
            docker_image: dockerImage,
            default_memory_mb: item.default_memory_mb || item.base_ram_mb || 512,
            default_cpu_cores: item.default_cpu_cores || 1,
            category: cat,
            environment_type: item.environment_type || item.target_environment || 'IDE_PERSISTENTE',
            is_official: item.is_official !== undefined ? Boolean(item.is_official) : true,
            is_active: item.is_active !== false,
            ports: item.ports || '',
            primaryIcon: techInfo.primaryIcon,
            techChips: techInfo.techChips,
            stack_config: item.stack_config || null
          };
        });

        this.templates.set(mapped.filter(t => t.is_active !== false));
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  openRequestModal(): void {
    this.requestPreset.set('');
    this.requestPresetStack.set(null);
    this.isRequestModalOpen.set(true);
  }

  closeRequestModal(): void {
    this.isRequestModalOpen.set(false);
  }

  onRequestSubmitted(): void {
    this.closeRequestModal();
    this.loadRequests();
  }

  openDetailModal(template: PublishedTemplate): void {
    this.selectedTemplateForDetail.set(template);
  }

  closeDetailModal(): void {
    this.selectedTemplateForDetail.set(null);
  }

  onUseFromDetail(template: PublishedTemplate): void {
    this.closeDetailModal();
    this.createLabFromTemplate(template);
  }

  createLabFromTemplate(template: PublishedTemplate): void {
    const coursesList = this.courses();
    if (coursesList.length === 1) {
      this.router.navigate(['/teacher/courses', coursesList[0].id, 'exercises', 'new'], {
        queryParams: { template_id: template.id, environment_type: template.environment_type || undefined }
      });
    } else if (coursesList.length > 1) {
      this.targetTemplateForCourse.set(template);
      this.isCoursePickerOpen.set(true);
    } else {
      this.router.navigate(['/teacher/courses']);
    }
  }

  selectCourseForLab(courseId: string): void {
    const tpl = this.targetTemplateForCourse();
    this.closeCoursePicker();
    if (tpl) {
      this.router.navigate(['/teacher/courses', courseId, 'exercises', 'new'], {
        queryParams: { template_id: tpl.id, environment_type: tpl.environment_type || undefined }
      });
    }
  }

  closeCoursePicker(): void {
    this.isCoursePickerOpen.set(false);
    this.targetTemplateForCourse.set(null);
  }
}
