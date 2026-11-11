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
import {
  EnvironmentTypeKey,
  environmentTypeOptions,
  isEnvironmentType,
} from '@shared/constants/environment-types.constants';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { AdminTemplatesService } from '@features/admin/services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import {
  inferTemplateTechnologies,
  resolveTemplateStack
} from '@features/admin/templates/template-tech-detector.utils';
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

interface TemplateTechDefaults {
  defaultCategory: string;
  defaultDescription: string;
}

const GENERAL_TECH_DEFAULTS: TemplateTechDefaults = {
  defaultCategory: 'General',
  defaultDescription: 'Entorno optimizado para prácticas y proyectos de laboratorio.'
};

function techDefaultsFor(primaryId: string): TemplateTechDefaults {
  switch (primaryId) {
    case 'java':
      return {
        defaultCategory: 'Empresarial & Backend',
        defaultDescription: 'Desarrollo de aplicaciones empresariales, programación orientada a objetos y servicios REST con Spring Boot.'
      };
    case 'spring':
    case 'maven':
    case 'gradle':
      return {
        defaultCategory: 'Empresarial & Backend',
        defaultDescription: 'Desarrollo de aplicaciones empresariales con el ecosistema Java y sus herramientas de construcción.'
      };
    case 'dotnet':
      return {
        defaultCategory: 'Empresarial & Backend',
        defaultDescription: 'Desarrollo en .NET y C# para aplicaciones empresariales y APIs multiplataforma.'
      };
    case 'python':
    case 'fastapi':
    case 'django':
    case 'flask':
    case 'pandas':
    case 'numpy':
    case 'jupyter':
    case 'data-science':
      return {
        defaultCategory: 'Ciencia de Datos & IA',
        defaultDescription: 'Análisis de datos, computación numérica, desarrollo web y fundamentos de inteligencia artificial.'
      };
    case 'cpp':
    case 'c':
    case 'gcc':
      return {
        defaultCategory: 'Sistemas & Computación',
        defaultDescription: 'Programación de sistemas, bajo nivel, memoria dinámica y concurrencia con compilador GCC.'
      };
    case 'rust':
    case 'webassembly':
      return {
        defaultCategory: 'Sistemas & Computación',
        defaultDescription: 'Desarrollo de software seguro en memoria, concurrencia moderna y WebAssembly.'
      };
    case 'go':
    case 'grpc':
      return {
        defaultCategory: 'Desarrollo Web & Cloud',
        defaultDescription: 'Construcción de microservicios de alto rendimiento, concurrencia nativa y gRPC.'
      };
    case 'php':
    case 'laravel':
      return {
        defaultCategory: 'Desarrollo Web & Cloud',
        defaultDescription: 'Desarrollo de aplicaciones web dinámicas y servicios backend tradicionales.'
      };
    case 'node':
    case 'typescript':
    case 'nextjs':
    case 'react':
    case 'angular':
    case 'vue':
    case 'vite':
    case 'express':
    case 'nestjs':
    case 'bun':
    case 'deno':
    case 'prisma':
      return {
        defaultCategory: 'Desarrollo Web & Cloud',
        defaultDescription: 'Desarrollo Full Stack con JavaScript y TypeScript, APIs REST y utilidades npm.'
      };
    case 'postgres':
    case 'mysql':
    case 'mariadb':
    case 'mongodb':
    case 'redis':
    case 'sqlite':
    case 'db':
      return {
        defaultCategory: 'Bases de Datos',
        defaultDescription: 'Motor de datos para prácticas de persistencia, consultas y modelado.'
      };
    default:
      return GENERAL_TECH_DEFAULTS;
  }
}

/**
 * Inferencia de respaldo cuando la plantilla no declara un stack resoluble.
 * Usa el mismo resolutor que el catálogo del administrador para que ambos
 * muestren las mismas tecnologías ante los mismos datos.
 */
export function resolveTemplateTechInfo(name: string, dockerImage: string, description: string = ''): {
  primaryIcon: string;
  techChips: TechChip[];
  defaultCategory: string;
  defaultDescription: string;
} {
  const detected = inferTemplateTechnologies({ name, docker_image: dockerImage, description });
  const primary = detected[0];

  if (!primary) {
    return {
      primaryIcon: 'docker',
      techChips: [{ name: 'Entorno Estándar', iconKey: 'docker' }],
      ...GENERAL_TECH_DEFAULTS
    };
  }

  return {
    primaryIcon: primary.iconType,
    techChips: detected.map(tech => ({ name: tech.name, iconKey: tech.iconType })),
    ...techDefaultsFor(primary.id)
  };
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
  private templatesCatalogService = inject(AdminTemplatesService);
  private router = inject(Router);

  courses = this.dashboardService.courses;
  templates = signal<PublishedTemplate[]>([]);
  techCatalog = signal<TemplateComponent[]>([]);
  isLoading = signal<boolean>(false);

  // Filtros reactivos
  categoryFilter = signal<string>('all');
  envTypeFilter = signal<string>('all');
  searchTerm = signal<string>('');

  // Pestañas: catálogo de entornos y las solicitudes propias del docente
  activeTab = signal<'catalog' | 'requests'>('catalog');
  requests = signal<EnvironmentRequest[]>([]);
  requestsLoading = signal<boolean>(false);
  /** Entorno del que parte una variante; se muestra como contexto, no como práctica. */
  variantBaseName = signal<string>('');
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
      // Una plantilla dual sirve a las dos modalidades: aparece en ambos filtros.
      items = items.filter(
        t => isEnvironmentType(t.environment_type, envType as EnvironmentTypeKey) || isEnvironmentType(t.environment_type, 'AMBOS')
      );
    }

    if (query) {
      items = items.filter(t =>
        t.name.toLowerCase().includes(query) ||
        (t.description && t.description.toLowerCase().includes(query)) ||
        (t.docker_image && t.docker_image.toLowerCase().includes(query)) ||
        this.techChipsFor(t).some(c => c.name.toLowerCase().includes(query))
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

  envTypeComboboxOptions = computed<ComboboxOption[]>(() =>
    environmentTypeOptions({ includeAll: true, include: ['IDE_PERSISTENTE', 'JUEZ_EFIMERO'] })
  );

  selectedEnvTypeLabel = computed<string>(() => {
    const current = this.envTypeFilter();
    const option = this.envTypeComboboxOptions().find(o => o.value === current);
    return option?.label ?? current;
  });

  onCategorySelected(opt: ComboboxOption): void {
    this.categoryFilter.set(opt.value);
  }

  onEnvTypeSelected(opt: ComboboxOption): void {
    this.envTypeFilter.set(opt.value);
  }

  ngOnInit(): void {
    this.loadTemplates();
    this.loadTechCatalog();
    this.loadRequests();
  }

  /**
   * El catálogo de componentes ya alimenta el modal de solicitud. Se reutiliza
   * aquí para traducir los ids de `stack_config` a nombres e iconos reales.
   */
  loadTechCatalog(): void {
    this.templatesCatalogService.getTechCatalog().subscribe({
      next: list => this.techCatalog.set(list || []),
      error: () => this.techCatalog.set([])
    });
  }

  techChipsFor(template: PublishedTemplate): TechChip[] {
    const resolved = resolveTemplateStack({ stack_config: template.stack_config ?? null }, this.techCatalog());
    if (resolved.length > 0) {
      return resolved.map(chip => ({ name: chip.name, iconKey: chip.iconKey }));
    }
    return template.techChips || [];
  }

  primaryIconFor(template: PublishedTemplate): string {
    const resolved = resolveTemplateStack({ stack_config: template.stack_config ?? null }, this.techCatalog());
    return resolved[0]?.iconKey || template.primaryIcon || 'docker';
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

  /** Pide una variante de un entorno que el docente ya conoce. */
  requestVariant(template: PublishedTemplate): void {
    this.variantBaseName.set(template.name);
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
    this.variantBaseName.set('');
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
    this.selectedTemplateForDetail.set({
      ...template,
      primaryIcon: this.primaryIconFor(template),
      techChips: this.techChipsFor(template)
    });
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
      this.router.navigate(['/teacher/cursos', coursesList[0].id, 'ejercicios', 'nuevo'], {
        queryParams: { template_id: template.id, environment_type: template.environment_type || undefined }
      });
    } else if (coursesList.length > 1) {
      this.targetTemplateForCourse.set(template);
      this.isCoursePickerOpen.set(true);
    } else {
      // Sin materias no existe una ruta de listado: el dashboard es el punto de partida.
      this.router.navigate(['/teacher/dashboard']);
    }
  }

  selectCourseForLab(courseId: string): void {
    const tpl = this.targetTemplateForCourse();
    this.closeCoursePicker();
    if (tpl) {
      this.router.navigate(['/teacher/cursos', courseId, 'ejercicios', 'nuevo'], {
        queryParams: { template_id: tpl.id, environment_type: tpl.environment_type || undefined }
      });
    }
  }

  closeCoursePicker(): void {
    this.isCoursePickerOpen.set(false);
    this.targetTemplateForCourse.set(null);
  }
}
