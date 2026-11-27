import {
  Component,
  OnInit,
  computed,
  effect,
  inject,
  input,
  output,
  signal
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { FormsModule } from '@angular/forms';
import { TimeoutError, timeout } from 'rxjs';
import { AiOperations } from '@shared/services/ai-operation';
import { EnvironmentRequestDraftService } from '../../services/environment-request-draft.service';
import { SuggestEnvironmentResponse } from '../../services/environment-request.service';
import {
  LucideAlertCircle,
  LucideArrowLeft,
  LucideBarChart3,
  LucideBoxes,
  LucideBug,
  LucideCheck,
  LucideChevronDown,
  LucideCpu,
  LucideDatabase,
  LucideFlaskConical,
  LucideGlobe,
  LucideGraduationCap,
  LucideInfo,
  LucidePlus,
  LucideRotateCcw,
  LucideSearch,
  LucideSend,
  LucideShieldCheck,
  LucideWand,
  LucideWand2,
  LucideWrench,
  LucideX
} from '@lucide/angular';
import { AdminTemplatesService } from '@features/admin/services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import {
  KNOWN_TECH_ICON_KEYS,
  TechLogoComponent
} from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { matchesRuntimeEcosystem } from '@features/admin/templates/components/model-library/model-library.utils';
import { AiProgressIndicatorComponent } from '@shared/components/ai-progress-indicator/ai-progress-indicator.component';
import { EnvironmentRequestService } from '../../services/environment-request.service';
import {
  EnvironmentMissingComponent,
  EnvironmentRequestPreview,
  EnvironmentRequestStackConfig,
  EnvironmentRequestUse
} from '../environment-request.models';

interface SelectedChip {
  id: string;
  name: string;
  layer: string;
  iconKey?: string;
  /** El chip repite el mismo ícono que el selector: logo o Lucide funcional. */
  usesLogo?: boolean;
  functionalIconKey?: string;
  missing?: boolean;
}

interface TechnicalPiece {
  id: string;
  name: string;
  layer: string;
  baseImage?: string;
}

/**
 * Corte del cliente para la consulta a la IA.
 *
 * Queda por encima del techo del servidor (90 s) para que el mensaje de corte
 * lo escriba el backend, que sabe qué pasó. Si la red o el proxy se cuelgan,
 * este es el que devuelve el botón en vez de dejarlo girando sin fin.
 */
const AI_CLIENT_TIMEOUT_MS = 100_000;

/**
 * Nombre legible de cada familia de ejecución.
 *
 * La clave es el `icon_key` del runtime que resuelve el token (o el token mismo
 * si el catálogo todavía no tiene ese lenguaje), nunca un slug de complemento:
 * un complemento nuevo hereda su grupo por `compatible_runtimes`.
 */
const ADDON_FAMILY_LABELS: Record<string, string> = {
  python: 'Python',
  node: 'Node.js',
  web: 'Interfaz web',
  java: 'Java',
  c: 'C / C++',
  cpp: 'C / C++',
  go: 'Go',
  rust: 'Rust',
  php: 'PHP',
  dotnet: '.NET',
  universal: 'Generales'
};

/**
 * Claves de ícono que son logo de ecosistema o relleno, no de la herramienta.
 * Un complemento que las declare se dibuja con ícono funcional: el logo de
 * Python en «Ciencia de Datos» no dice nada que el nombre no diga.
 */
const GENERIC_ADDON_ICON_KEYS = new Set<string>([
  'docker',
  'python',
  'c',
  'cpp',
  'java',
  'go',
  'rust',
  'php',
  'dotnet',
  'node',
  'nodejs'
]);

/** Complementos con logo de ecosistema que esconden su función real. */
const ADDON_FUNCTIONAL_ONLY_SLUGS = new Set<string>(['spring-maven']);

const ADDON_FAMILY_ORDER = [
  'python',
  'node',
  'web',
  'java',
  'c',
  'cpp',
  'go',
  'rust',
  'php',
  'dotnet',
  'universal'
];

/**
 * Clasificador de íconos para complementos de tipo `toolkit`.
 *
 * Se evalúa sobre `name`, `description`, `tools_provided` y `slug`; el orden
 * decide los empates («Pruebas y Calidad» es de pruebas antes que de linter).
 * Un complemento futuro cae en `tool` sin tocar código ni subir un SVG.
 */
const ADDON_ICON_RULES: { key: string; terms: string[] }[] = [
  // La depuración manda antes que cualquier kit: «Depuración y Memoria»
  // describe «estructuras de datos» y sin esta prioridad caía en BarChart3.
  { key: 'debug', terms: ['gdb', 'valgrind', 'depura', 'debug'] },
  // Los kits de datos se reconocen antes que las pruebas: "Ciencia de Datos"
  // trae pytest en sus herramientas y sin esta prioridad caía en FlaskConical.
  {
    key: 'analytics',
    terms: [
      'numpy',
      'pandas',
      'scikit',
      'machine learning',
      'ciencia de datos',
      'analisis de datos',
      'análisis de datos',
      'data science',
      'dataframe',
      'ia',
      'ml'
    ]
  },
  { key: 'testing', terms: ['prueba', 'test', 'junit', 'pytest', 'xunit', 'vitest', 'jest', 'phpunit', 'googletest'] },
  { key: 'quality', terms: ['linter', 'linte', 'calidad', 'clippy', 'ruff', 'golangci', 'fmt'] },
  {
    key: 'database',
    terms: ['sql', 'jdbc', 'pdo', 'mongo', 'redis', 'conect', 'driver', 'prisma', 'typeorm', 'gorm', 'hibernate', 'ef core']
  },
  { key: 'concurrency', terms: ['concurren', 'openmp', 'pthread', 'paralel'] }
];

type PickerCategoryId = 'all' | 'languages' | 'web' | 'addons' | 'databases';

interface PickerCategory {
  id: PickerCategoryId;
  label: string;
  total: number;
  selected: number;
}

interface PickerGroup {
  id: string;
  category: Exclude<PickerCategoryId, 'all'>;
  label: string;
  hint: string;
  items: TemplateComponent[];
}

/**
 * Pedido de entorno del docente.
 *
 * Una sola pantalla: qué van a hacer los estudiantes y cómo lo van a usar. El
 * entorno es institucional y reutilizable en cualquier materia, así que el
 * docente no lo ata a una: nunca escribe una imagen, una RAM ni un puerto, y el
 * backend arma la receta con el catálogo.
 */
@Component({
  selector: 'template-request-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    AiProgressIndicatorComponent,
    LucideAlertCircle,
    LucideArrowLeft,
    LucideBarChart3,
    LucideBoxes,
    LucideBug,
    LucideCheck,
    LucideChevronDown,
    LucideCpu,
    LucideDatabase,
    LucideFlaskConical,
    LucideGlobe,
    LucideGraduationCap,
    LucideInfo,
    LucidePlus,
    LucideRotateCcw,
    LucideSearch,
    LucideSend,
    LucideShieldCheck,
    LucideWand,
    LucideWand2,
    LucideWrench,
    LucideX,
    TechLogoComponent,
    EnvironmentLabelPipe
  ],
  templateUrl: './template-request-modal.component.html',
  styleUrl: './template-request-modal.component.scss',
  host: {
    '(document:keydown.escape)': 'onEscape()'
  }
})
export class TemplateRequestModalComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService);
  private requestService = inject(EnvironmentRequestService);
  private draftService = inject(EnvironmentRequestDraftService);

  /**
   * Entorno del que parte una variante. Se muestra como contexto: la práctica
   * la sigue escribiendo el docente, porque el pedido es qué van a hacer los
   * estudiantes y no una frase armada por el catálogo.
   */
  variantBase = input<string>('');
  /** Piezas de la plantilla de origen, ya resueltas contra el catálogo. */
  presetStack = input<EnvironmentRequestStackConfig | null>(null);

  closed = output<void>();
  requested = output<void>();

  // Catálogo
  catalog = signal<TemplateComponent[]>([]);
  catalogLoading = signal<boolean>(true);

  // Selección de piezas
  runtimeId = signal<string>('');
  webPlatformId = signal<string>('');
  addonIds = signal<string[]>([]);
  databaseId = signal<string>('');

  // Uso declarado
  uses = signal<EnvironmentRequestUse[]>(['practice']);

  // Práctica y contexto
  aiPrompt = signal<string>('');
  description = signal<string>('');
  descriptionTouched = signal<boolean>(false);
  title = signal<string>('');
  titleTouched = signal<boolean>(false);

  // Ayuda de IA. La llamada vive en la raíz: cerrar el modal no la aborta y el
  // resultado queda esperando para cuando el docente vuelva.
  private readonly aiOperations = inject(AiOperations);
  private readonly aiOperation = this.aiOperations.get<SuggestEnvironmentResponse>('teacher-suggest-environment', {
    resolveError: resolveSuggestionError,
    notify: {
      success: 'La lista de herramientas está lista.',
      error: (message: string) => message
    }
  });
  readonly aiLoading = this.aiOperation.isRunning;
  readonly aiElapsedSeconds = this.aiOperation.elapsedSeconds;
  missingFromAI = signal<EnvironmentMissingComponent[]>([]);
  aiNotice = signal<string | null>(null);

  // Receta armada
  preview = signal<EnvironmentRequestPreview | null>(null);
  previewLoading = signal<boolean>(false);

  // UI
  isPickerOpen = signal<boolean>(false);
  /** La asistencia con IA es secundaria: arranca plegada y se abre a pedido. */
  aiPanelOpen = signal<boolean>(false);
  pickerQuery = signal<string>('');
  pickerCategory = signal<PickerCategoryId>('all');
  /** Con lenguaje elegido, la lista arranca acotada a los complementos que aplican. */
  showAllAddons = signal<boolean>(false);
  showMissingNames = signal<boolean>(false);
  showTechnical = signal<boolean>(false);
  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);
  result = signal<{ message: string; joined: boolean } | null>(null);
  private presetApplied = false;

  constructor() {
    // El borrador vive en la raíz: cerrar el modal no pierde lo escrito.
    const draft = this.draftService.draft();
    this.aiPrompt.set(draft.aiPrompt);
    this.description.set(draft.description);
    this.descriptionTouched.set(draft.descriptionTouched);
    this.title.set(draft.title);
    this.titleTouched.set(draft.titleTouched);
    this.uses.set(draft.uses.length ? draft.uses : ['practice']);
    this.runtimeId.set(draft.runtimeId);
    this.webPlatformId.set(draft.webPlatformId);
    this.databaseId.set(draft.databaseId);
    this.addonIds.set(draft.addonIds);
    this.missingFromAI.set(draft.missingFromAI);
    this.aiNotice.set(draft.aiNotice);

    effect(() => {
      this.draftService.patch({
        aiPrompt: this.aiPrompt(),
        description: this.description(),
        descriptionTouched: this.descriptionTouched(),
        title: this.title(),
        titleTouched: this.titleTouched(),
        uses: this.uses(),
        runtimeId: this.runtimeId(),
        webPlatformId: this.webPlatformId(),
        databaseId: this.databaseId(),
        addonIds: this.addonIds(),
        missingFromAI: this.missingFromAI(),
        aiNotice: this.aiNotice()
      });
    });

    // El mensaje de la consulta llega desde el servicio aunque el modal se
    // haya cerrado y vuelto a abrir.
    effect(() => {
      const message = this.aiOperation.errorMessage();
      if (message) {
        this.aiNotice.set(message);
      }
    });

    // `take` reclama el resultado una sola vez: al reabrir, se aplica y se libera.
    // Se espera al catálogo: sin él los slugs no se pueden resolver en piezas.
    effect(() => {
      if (this.catalogLoading()) {
        return;
      }
      const suggestion = this.aiOperation.take();
      if (suggestion) {
        this.applySuggestion(suggestion);
      }
    });
  }

  /** Lenguaje elegido: define qué complementos aplican al entorno. */
  readonly selectedRuntime = computed(() => this.catalog().find(item => item.id === this.runtimeId()));

  readonly runtimeOptions = computed(() =>
    this.catalog().filter(item => item.layer === 'runtime' && item.sub_category !== 'web')
  );
  readonly webOptions = computed(() =>
    this.catalog().filter(item => item.layer === 'runtime' && item.sub_category === 'web')
  );

  /** Piezas elegidas, con el nombre que el docente reconoce. */
  readonly chips = computed<SelectedChip[]>(() => {
    const items = this.catalog();
    const byId = new Map<string, TemplateComponent>();
    items.forEach(item => {
      byId.set(item.id, item);
      byId.set(item.slug, item);
    });

    const result: SelectedChip[] = [];
    const push = (id: string) => {
      const found = byId.get(id);
      if (!found) {
        return;
      }
      result.push({
        id: found.id,
        name: found.name,
        layer: found.layer,
        iconKey: found.icon_key,
        // El chip repite el dibujo del selector: los logos de lenguaje, web y
        // base de datos siguen siendo su logo; los complementos respetan la
        // regla híbrida (logo de marca o Lucide funcional).
        usesLogo: found.layer === 'addon' ? this.addonUsesLogo(found) : true,
        functionalIconKey: found.layer === 'addon' ? this.addonIconKey(found) : undefined
      });
    };

    if (this.runtimeId()) push(this.runtimeId());
    if (this.webPlatformId()) push(this.webPlatformId());
    this.addonIds().forEach(push);
    if (this.databaseId()) push(this.databaseId());

    for (const missing of this.missingFromAI()) {
      result.push({ id: `missing:${missing.name}`, name: missing.name, layer: missing.layer, missing: true });
    }
    return result;
  });

  readonly hasAnyPiece = computed(() => this.chips().length > 0);
  readonly hasMissingPieces = computed(() => this.missingFromAI().length > 0);
  readonly hasCatalogPieces = computed(() =>
    Boolean(
      this.runtimeId() ||
        this.webPlatformId() ||
        this.addonIds().length > 0 ||
        (this.databaseId() && !this.isNoneDatabase(this.databaseId()))
    )
  );
  readonly requiresDescription = computed(() => !this.hasCatalogPieces() || this.hasMissingPieces());
  readonly isDescriptionValid = computed(() => this.description().trim().length >= 10);

  /** El placeholder acompaña al badge: nunca dice «Opcional» cuando es obligatorio. */
  readonly descriptionPlaceholder = computed(() =>
    this.requiresDescription()
      ? 'Describe qué necesitas instalar o qué harán tus estudiantes con esas piezas (mínimo 10 caracteres).'
      : 'Opcional: aclara requisitos específicos de versiones, puertos o flujo de clase.'
  );
  readonly isNoteSatisfied = computed(() => !this.requiresDescription() || this.isDescriptionValid());
  readonly canSubmit = computed(() => this.uses().length > 0 && this.isNoteSatisfied());
  readonly hasDraftContent = computed(() => this.draftService.hasContent());

  /** Nombres que la IA nombró y que el catálogo todavía no tiene. */
  readonly missingNames = computed(() => this.missingFromAI().map(item => item.name));

  private isNoneDatabase(id: string): boolean {
    if (!id || id === 'none' || id === 'db-none') {
      return true;
    }
    const found = this.catalog().find(item => item.id === id);
    return (found?.service_engine || '').toLowerCase() === 'none';
  }

  /** La base de datos centinela «Sin Base de Datos» no se ofrece como herramienta. */
  isSentinelDatabase(item: TemplateComponent): boolean {
    if (item.slug === 'db-none') {
      return true;
    }
    if (item.layer !== 'database') {
      return false;
    }
    const engine = (item.service_engine || '').trim().toLowerCase();
    const subCategory = (item.sub_category || '').trim().toLowerCase();
    return engine === 'none' || subCategory === 'none';
  }

  /** Conteo por categoría para los filtros rápidos debajo del buscador. */
  readonly pickerCategories = computed<PickerCategory[]>(() => {
    const items = this.catalog();
    const runtimes = items.filter(item => item.layer === 'runtime' && item.sub_category !== 'web');
    const web = items.filter(item => item.layer === 'runtime' && item.sub_category === 'web');
    const addons = items.filter(item => item.layer === 'addon');
    const databases = items.filter(item => item.layer === 'database' && !this.isSentinelDatabase(item));
    const selectedAddons = this.addonIds().length;

    return [
      {
        id: 'all',
        label: 'Todas',
        total: runtimes.length + web.length + addons.length + databases.length,
        selected: this.chips().filter(chip => !chip.missing).length
      },
      { id: 'languages', label: 'Lenguajes', total: runtimes.length, selected: this.runtimeId() ? 1 : 0 },
      { id: 'web', label: 'Interfaz web', total: web.length, selected: this.webPlatformId() ? 1 : 0 },
      { id: 'addons', label: 'Complementos', total: addons.length, selected: selectedAddons },
      {
        id: 'databases',
        label: 'Bases de datos',
        total: databases.length,
        selected: this.isNoneDatabase(this.databaseId()) ? 0 : 1
      }
    ];
  });

  readonly addonsFiltered = computed(() => Boolean(this.selectedRuntime()) && !this.showAllAddons());

  /** Tamaño real de la sección de complementos, para el atajo «Ver todos». */
  readonly addonTotal = computed(() => this.catalog().filter(item => item.layer === 'addon').length);

  readonly addonToggleLabel = computed(() => {
    if (this.showAllAddons()) {
      const runtime = this.selectedRuntime();
      return runtime ? `Ver solo los de ${runtime.name}` : 'Ver solo los compatibles';
    }
    return `Ver todos (${this.addonTotal()})`;
  });

  readonly pickerEmptyMessage = computed(() => {
    const query = this.pickerQuery().trim();
    if (query) {
      return `No encontramos «${query}» en el catálogo. Escríbela en la asistencia IA o en la nota y la pedimos.`;
    }
    const category = this.pickerCategories().find(item => item.id === this.pickerCategory());
    return category
      ? `Todavía no hay herramientas en ${category.label.toLowerCase()}.`
      : 'No encontramos herramientas en el catálogo.';
  });

  /**
   * Lista del selector, agrupada como el catálogo del estudio.
   *
   * Una grilla plana mezcla Rust, Angular y Prisma como si fueran lo mismo. Acá
   * el docente ve lenguajes, interfaz web, complementos y bases de datos, en ese
   * orden. Con un lenguaje elegido, los complementos se acotan a los que
   * declaran ese ecosistema en `compatible_runtimes`; el resto queda detrás de
   * «Ver complementos de otros lenguajes», no a media lista.
   */
  readonly pickerGroups = computed<PickerGroup[]>(() => {
    const query = this.pickerQuery().trim().toLowerCase();
    const matches = (item: TemplateComponent) =>
      !query ||
      item.name.toLowerCase().includes(query) ||
      item.slug.toLowerCase().includes(query) ||
      (item.description || '').toLowerCase().includes(query);
    const byOrder = (a: TemplateComponent, b: TemplateComponent) =>
      (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.name.localeCompare(b.name);

    const items = this.catalog().filter(matches);
    const addons = items.filter(item => item.layer === 'addon');
    const visibleAddons = this.addonsFiltered()
      ? addons.filter(item => this.isPieceCompatible(item) || this.isPieceSelected(item))
      : addons;
    const runtime = this.selectedRuntime();

    // Con lenguaje elegido la sección es una sola lista acotada; sin lenguaje,
    // los complementos se agrupan por ecosistema para que 41 tarjetas no queden
    // en un bloque indistinguible.
    const addonGroups: PickerGroup[] = runtime
      ? [
          {
            id: 'addons',
            category: 'addons',
            label: 'Complementos',
            hint: `Solo los de ${runtime.name}`,
            items: [...visibleAddons].sort(byOrder)
          }
        ]
      : this.buildAddonGroups([...visibleAddons].sort(byOrder));

    const groups: PickerGroup[] = [
      {
        id: 'languages',
        category: 'languages',
        label: 'Lenguajes y frameworks',
        hint: 'La base del entorno',
        items: items.filter(item => item.layer === 'runtime' && item.sub_category !== 'web').sort(byOrder)
      },
      {
        id: 'web',
        category: 'web',
        label: 'Interfaz web',
        hint: 'Opcional',
        items: items.filter(item => item.layer === 'runtime' && item.sub_category === 'web').sort(byOrder)
      },
      ...addonGroups,
      {
        id: 'databases',
        category: 'databases',
        label: 'Bases de datos',
        hint: 'Una por entorno',
        items: items
          .filter(item => item.layer === 'database' && !this.isSentinelDatabase(item))
          .sort(byOrder)
      }
    ];

    const category = this.pickerCategory();
    return groups.filter(
      group => (category === 'all' || group.category === category) && group.items.length > 0
    );
  });

  /**
   * Agrupa los complementos por ecosistema declarado en `compatible_runtimes`.
   *
   * El nombre del grupo sale del runtime que resuelve el token, no de una lista
   * de complementos: dar de alta uno nuevo en el catálogo lo coloca solo.
   */
  private buildAddonGroups(items: TemplateComponent[]): PickerGroup[] {
    const buckets = new Map<string, { label: string; items: TemplateComponent[] }>();
    for (const item of items) {
      const key = this.addonFamilyKey(item);
      const bucket = buckets.get(key);
      if (bucket) {
        bucket.items.push(item);
        continue;
      }
      buckets.set(key, { label: this.addonFamilyLabel(key), items: [item] });
    }

    return [...buckets.entries()]
      .sort(([a], [b]) => this.familyRank(a) - this.familyRank(b) || a.localeCompare(b))
      .map(([key, bucket]) => ({
        id: `addons-${key}`,
        category: 'addons' as const,
        label: bucket.label,
        hint: 'Complementos',
        items: bucket.items
      }));
  }

  /** Familia principal de un complemento seg\u00fan su primer token compatible. */
  private addonFamilyKey(item: TemplateComponent): string {
    const tokens = (item.compatible_runtimes || [])
      .map(token => token.trim().toLowerCase())
      .filter(token => token && token !== 'universal' && token !== 'any');
    if (!tokens.length) {
      return 'universal';
    }
    const token = tokens[0];
    const runtime = this.catalog().find(
      entry => entry.layer === 'runtime' && matchesRuntimeEcosystem(entry, token)
    );
    const icon = (runtime?.icon_key || '').toLowerCase();
    return icon && icon !== 'docker' ? icon : token;
  }

  private addonFamilyLabel(key: string): string {
    const label = ADDON_FAMILY_LABELS[key];
    if (label) {
      return label;
    }
    const runtime = this.catalog().find(
      entry => entry.layer === 'runtime' && (entry.icon_key || '').toLowerCase() === key
    );
    return runtime ? runtime.name : key.toUpperCase();
  }

  private familyRank(key: string): number {
    const index = ADDON_FAMILY_ORDER.indexOf(key);
    return index === -1 ? ADDON_FAMILY_ORDER.length : index;
  }

  /**
   * Regla híbrida del selector manual: un complemento con logo propio se dibuja
   * con su marca, salvo que esa clave sea solo el ecosistema (Python, Node,
   * C/C++…) o un complemento donde el logo esconde la función (Maven).
   * En el resto manda el ícono funcional de Lucide.
   */
  addonUsesLogo(item: TemplateComponent): boolean {
    if (item.layer !== 'addon') {
      return false;
    }
    const key = (item.icon_key || '').trim().toLowerCase();
    if (!key || !KNOWN_TECH_ICON_KEYS.has(key)) {
      return false;
    }
    if (GENERIC_ADDON_ICON_KEYS.has(key)) {
      return false;
    }
    if (ADDON_FUNCTIONAL_ONLY_SLUGS.has((item.slug || '').trim().toLowerCase())) {
      return false;
    }
    return true;
  }

  /** Clave de ícono para un complemento; cualquier valor nuevo cae en `tool`. */
  addonIconKey(item: TemplateComponent): string {
    const functionalOnly = ADDON_FUNCTIONAL_ONLY_SLUGS.has((item.slug || '').trim().toLowerCase());
    if (!functionalOnly) {
      if (item.sub_category === 'framework') {
        return 'framework';
      }
      if (item.sub_category === 'orm') {
        return 'orm';
      }
    }
    const haystack = [
      item.name,
      item.description,
      (item.tools_provided || []).join(' '),
      item.slug
    ]
      .join(' ')
      .toLowerCase();
    const rule = ADDON_ICON_RULES.find(entry =>
      entry.terms.some(term => this.matchesIconTerm(haystack, term))
    );
    return rule ? rule.key : 'tool';
  }

  /**
   * Coincidencia del clasificador de íconos. Las siglas cortas («ia», «ml») se
   * buscan como palabra completa: dentro de «memoria» o «html» darían un falso
   * positivo. El resto son raíces y se buscan como subcadena.
   */
  private matchesIconTerm(haystack: string, term: string): boolean {
    if (term.length <= 2) {
      return new RegExp(`(^|[^a-z0-9])${term}([^a-z0-9]|$)`).test(haystack);
    }
    return haystack.includes(term);
  }

  /** Línea secundaria de la tarjeta: la descripción real del catálogo. */
  addonSubtitle(item: TemplateComponent): string {
    const description = (item.description || '').trim();
    if (description) {
      return description;
    }
    if (item.sub_category === 'framework') {
      return 'Framework';
    }
    if (item.sub_category === 'orm') {
      return 'ORM / Datos';
    }
    return 'Herramienta';
  }

  /**
   * Piezas que el sistema va a usar por dentro.
   *
   * Se arma con la receta del backend y, si la receta todavía no llegó, con lo
   * que el docente ya eligió en el catálogo: nunca decimos "no hay piezas" con
   * herramientas puestas.
   */
  readonly technicalPieces = computed<TechnicalPiece[]>(() => {
    const resolved = (this.preview()?.components || []).filter(item => !item.missing);
    if (resolved.length) {
      return resolved.map(item => ({
        id: item.id,
        name: item.name,
        layer: item.layer,
        baseImage: item.base_image
      }));
    }

    const byId = new Map(this.catalog().map(item => [item.id, item] as const));
    return this.chips()
      .filter(chip => !chip.missing)
      .map(chip => {
        const component = byId.get(chip.id);
        return {
          id: chip.id,
          name: chip.name,
          layer: chip.layer,
          baseImage: component?.base_image
        };
      });
  });

  ngOnInit(): void {
    this.templatesService.getTechCatalog().subscribe({
      next: list => {
        this.catalog.set(list || []);
        this.catalogLoading.set(false);
        this.applyPresetStack();
        if (!this.presetStack() && this.hasAnyPiece()) {
          this.refreshPreview();
        }
      },
      error: () => {
        this.catalog.set([]);
        this.catalogLoading.set(false);
      }
    });
  }

  /**
   * Arranca la solicitud con las piezas de la plantilla de origen.
   *
   * El docente solo cambia la que no quiere (React por Angular); el resto del
   * entorno llega puesto. Si una pieza ya no existe en el catálogo se ignora en
   * silencio: el texto sigue nombrando el entorno completo.
   */
  private applyPresetStack(): void {
    const stack = this.presetStack();
    if (!stack || this.presetApplied) {
      return;
    }
    this.presetApplied = true;

    const byKey = new Map<string, TemplateComponent>();
    for (const item of this.catalog()) {
      byKey.set(item.id, item);
      byKey.set(item.slug, item);
    }

    const runtime = this.resolveRuntime(stack, byKey);
    if (runtime) {
      if (runtime.sub_category === 'web') {
        this.webPlatformId.set(runtime.id);
      } else {
        this.runtimeId.set(runtime.id);
      }
    }

    if (stack.web_platform_id) {
      const web = byKey.get(stack.web_platform_id);
      if (web) {
        this.webPlatformId.set(web.id);
      }
    }

    const addons = (stack.addon_ids || [])
      .map(id => byKey.get(id))
      .filter((item): item is TemplateComponent => Boolean(item))
      .map(item => item.id);
    if (addons.length) {
      this.addonIds.set(addons);
    }

    if (stack.database_id) {
      const database = byKey.get(stack.database_id);
      if (database) {
        this.databaseId.set(database.id);
      }
    }

    this.refreshPreview();
  }

  /**
   * Runtime de la plantilla de origen.
   *
   * Las plantillas nuevas traen el id de la pieza. Las anteriores declaran
   * `language` y `version`, así que se busca la pieza del catálogo cuyo slug
   * coincide con ese lenguaje. Si no hay coincidencia clara, no se adivina:
   * la solicitud queda sin runtime y el docente lo elige.
   */
  private resolveRuntime(
    stack: EnvironmentRequestStackConfig,
    byKey: Map<string, TemplateComponent>
  ): TemplateComponent | undefined {
    const direct = stack.runtime_id ? byKey.get(stack.runtime_id) : undefined;
    if (direct) {
      return direct;
    }

    const language = (stack.language || '').trim().toLowerCase();
    if (!language) {
      return undefined;
    }

    const runtimes = this.catalog().filter(item => item.layer === 'runtime');
    const exact = runtimes.find(item => item.slug.toLowerCase() === language);
    if (exact) {
      return exact;
    }

    const prefixed = runtimes.filter(item => item.slug.toLowerCase().startsWith(`${language}-`));
    const named = runtimes.filter(item => item.name.toLowerCase().includes(language));
    const candidates = prefixed.length ? prefixed : named;
    return candidates.find(item => item.sub_category !== 'web') || candidates[0];
  }

  toggleUse(use: EnvironmentRequestUse): void {
    const current = this.uses();
    if (current.includes(use)) {
      this.uses.set(current.filter(item => item !== use));
    } else {
      this.uses.set([...current, use]);
    }
    this.refreshPreview();
  }

  isUseSelected(use: EnvironmentRequestUse): boolean {
    return this.uses().includes(use);
  }

  layerLabel(layer: string): string {
    if (layer === 'database') return 'Base de datos';
    if (layer === 'addon') return 'Complemento';
    return 'Lenguaje o framework';
  }

  // --- selección de piezas ---------------------------------------------------

  isPieceSelected(item: TemplateComponent): boolean {
    return this.chips().some(chip => chip.id === item.id);
  }

  /**
   * Un complemento o una base de datos solo aplica si declara el ecosistema del
   * lenguaje elegido. Sin lenguaje todavía no se atenúa nada: el docente está
   * mirando el catálogo completo.
   */
  isPieceCompatible(item: TemplateComponent): boolean {
    const runtime = this.selectedRuntime();
    if (!runtime || (item.layer !== 'addon' && item.layer !== 'database')) {
      return true;
    }
    const targets = (item.compatible_runtimes || [])
      .map(target => target.trim().toLowerCase())
      .filter(Boolean);
    if (!targets.length || targets.includes('universal') || targets.includes('any')) {
      return true;
    }
    return targets.some(target => matchesRuntimeEcosystem(runtime, target));
  }

  incompatibleHint(_item: TemplateComponent): string {
    const runtime = this.selectedRuntime();
    return runtime ? `No aplica a ${runtime.name}` : 'No aplica al lenguaje elegido';
  }

  /** A qué ecosistema apunta un complemento, para escanearlo sin lenguaje elegido. */
  setPickerCategory(category: PickerCategoryId): void {
    this.pickerCategory.set(category);
  }

  /** Buscar siempre recorre el catálogo completo: el chip vuelve a «Todas». */
  onPickerQueryChange(value: string): void {
    this.pickerQuery.set(value);
    if (value.trim()) {
      this.pickerCategory.set('all');
    }
  }

  togglePicker(event: Event): void {
    event.stopPropagation();
    if (this.isPickerOpen()) {
      this.closePicker();
      return;
    }
    this.pickerQuery.set('');
    this.pickerCategory.set('all');
    this.showAllAddons.set(false);
    this.isPickerOpen.set(true);
  }

  /**
   * Un complemento o una base de datos que no declara el ecosistema del nuevo
   * lenguaje deja de aplicar: se quita en vez de viajar incompatible al backend.
   */
  private pruneIncompatiblePieces(): void {
    if (!this.selectedRuntime()) {
      return;
    }
    const catalog = this.catalog();
    const addons = this.addonIds().filter(id => {
      const item = catalog.find(entry => entry.id === id);
      return !item || item.layer !== 'addon' || this.isPieceCompatible(item);
    });
    if (addons.length !== this.addonIds().length) {
      this.addonIds.set(addons);
    }
    const database = catalog.find(entry => entry.id === this.databaseId());
    if (database && !this.isPieceCompatible(database)) {
      this.databaseId.set('');
    }
  }

  /**
   * Agrega o quita una pieza sin cerrar la lista: el docente suele necesitar
   * un lenguaje y su complemento (Python y Django) en la misma pasada.
   */
  addPiece(item: TemplateComponent): void {
    if (!this.isPieceCompatible(item)) {
      return;
    }
    if (this.isPieceSelected(item)) {
      this.removePieceById(item.id);
      this.refreshPreview();
      return;
    }

    if (item.layer === 'database') {
      if (this.isSentinelDatabase(item)) {
        return;
      }
      this.databaseId.set(item.id);
    } else if (item.layer === 'runtime' && item.sub_category === 'web') {
      this.webPlatformId.set(item.id);
    } else if (item.layer === 'runtime') {
      this.runtimeId.set(item.id);
      this.pruneIncompatiblePieces();
    } else {
      this.addonIds.set([...this.addonIds(), item.id]);
    }

    this.refreshPreview();
  }

  private removePieceById(id: string): void {
    if (this.runtimeId() === id) {
      this.runtimeId.set('');
    } else if (this.webPlatformId() === id) {
      this.webPlatformId.set('');
    } else if (this.databaseId() === id) {
      this.databaseId.set('');
    } else {
      this.addonIds.set(this.addonIds().filter(item => item !== id));
    }
  }

  removeChip(chip: SelectedChip): void {
    if (chip.missing) {
      this.missingFromAI.set(this.missingFromAI().filter(item => item.name !== chip.name));
      this.refreshPreview();
      return;
    }
    this.removePieceById(chip.id);
    this.refreshPreview();
  }

  /**
   * Cierra la lista. El panel la llama en cada click: el selector y sus
   * opciones cortan la propagación, así que solo cierra al tocar fuera.
   */
  closePicker(): void {
    this.isPickerOpen.set(false);
    this.pickerQuery.set('');
    this.pickerCategory.set('all');
    this.showAllAddons.set(false);
  }

  /** Escape cierra primero la lista y, si ya está cerrada, el diálogo. */
  onEscape(): void {
    if (this.isPickerOpen()) {
      this.closePicker();
      return;
    }
    this.close();
  }

  // --- receta ----------------------------------------------------------------

  stackConfig(): EnvironmentRequestStackConfig {
    const services: { role?: string; component_id?: string }[] = [];
    if (this.databaseId()) {
      const db = this.catalog().find(p => p.id === this.databaseId());
      const role = (db?.service_engine || '').toLowerCase() === 'redis' ? 'cache' : 'primary_database';
      services.push({ role, component_id: this.databaseId() });
    }

    return {
      mode: 'visual',
      runtime_id: this.runtimeId() || undefined,
      web_platform_id: this.webPlatformId() || undefined,
      addon_ids: this.addonIds().length ? this.addonIds() : undefined,
      database_id: this.databaseId() || undefined,
      services: services.length ? services : undefined
    };
  }

  refreshPreview(): void {
    this.previewLoading.set(true);
    this.requestService.preview(this.uses(), this.stackConfig(), this.missingNames()).subscribe({
      next: preview => {
        this.preview.set(preview);
        this.previewLoading.set(false);
        if (!this.titleTouched()) {
          if (preview.suggested_name) {
            this.title.set(preview.suggested_name);
          } else if (!this.hasCatalogPieces() && !this.hasMissingPieces()) {
            this.title.set('');
          }
        }
      },
      error: () => {
        this.previewLoading.set(false);
      }
    });
  }

  onTitleChange(value: string): void {
    this.titleTouched.set(true);
    this.title.set(value);
  }

  onDescriptionChange(value: string): void {
    this.descriptionTouched.set(true);
    this.description.set(value);
  }

  // --- IA --------------------------------------------------------------------

  /** Corta la espera: el docente no queda atado a un modelo saturado. */
  cancelAI(): void {
    this.aiOperation.cancel();
    this.aiNotice.set('Detuviste la consulta. Puedes volver a intentar o elegir las herramientas a mano.');
  }

  /** Abre o cierra la asistencia con IA. Al plegarla se descarta el error de largo. */
  toggleAiPanel(): void {
    const next = !this.aiPanelOpen();
    this.aiPanelOpen.set(next);
    if (!next && this.formError()) {
      this.formError.set(null);
    }
  }

  onAiPromptChange(value: string): void {
    this.aiPrompt.set(value);
    if (this.formError()) {
      this.formError.set(null);
    }
  }

  interpret(): void {
    const prompt = this.aiPrompt().trim();
    if (prompt.length < 10) {
      this.aiPanelOpen.set(true);
      this.formError.set('Cuéntanos un poco más en la asistencia con IA: al menos 10 caracteres.');
      return;
    }
    this.formError.set(null);
    this.aiNotice.set(null);
    this.showMissingNames.set(false);

    this.aiOperation.run(
      this.requestService.suggestEnvironment(prompt.slice(0, 600)).pipe(timeout(AI_CLIENT_TIMEOUT_MS))
    );
  }

  /**
   * Ubica una pieza del catálogo a partir del slug que devuelve la IA.
   *
   * Tolera variantes que el motor escribe de vez en cuando (`sqlite` por
   * `sqlite-3`): busca por slug, id, nombre, motor de servicio y prefijo. Una
   * pieza que existe no se descarta en silencio por una diferencia de nombre.
   */
  private findCatalogPiece(slug: string | undefined): TemplateComponent | undefined {
    const needle = (slug || '').trim().toLowerCase();
    if (!needle) {
      return undefined;
    }
    const items = this.catalog();
    return (
      items.find(item => (item.slug || '').toLowerCase() === needle) ||
      items.find(item => (item.id || '').toLowerCase() === needle) ||
      items.find(item => (item.name || '').toLowerCase() === needle) ||
      items.find(item => (item.service_engine || '').toLowerCase() === needle) ||
      items.find(item => (item.slug || '').toLowerCase().startsWith(needle + '-'))
    );
  }

  /** Aplica la interpretación de la IA, venga del modal o de una consulta previa. */
  private applySuggestion(suggestion: SuggestEnvironmentResponse): void {
    const runtime = this.findCatalogPiece(suggestion.runtime_slug);
    this.runtimeId.set(runtime ? runtime.id : '');

    const web = this.findCatalogPiece(suggestion.web_platform_slug);
    this.webPlatformId.set(web ? web.id : '');

    const db = this.findCatalogPiece(suggestion.database_slug);
    this.databaseId.set(db ? db.id : '');

    const addons = (suggestion.addon_slugs || [])
      .map(slug => this.findCatalogPiece(slug))
      .filter((item): item is TemplateComponent => Boolean(item))
      .map(item => item.id);
    this.addonIds.set(addons);
    this.pruneIncompatiblePieces();

    const missing = suggestion.missing_components || [];
    this.missingFromAI.set(missing);

    if (!this.titleTouched()) {
      this.title.set((suggestion.name || '').trim());
    }
    if (!this.descriptionTouched() && missing.length > 0 && this.aiPrompt().trim().length >= 10) {
      this.description.set(this.aiPrompt().trim());
    }

    this.aiNotice.set(
      missing.length
        ? 'Esto es lo que entendí. De lo que nombraste, unas piezas no están en el catálogo: las pedimos junto con tu solicitud.'
        : 'Esto es lo que entendí. Puedes quitar o agregar herramientas antes de enviar.'
    );
    this.refreshPreview();
  }

  // --- envío ----------------------------------------------------------------

  submitRequest(): void {
    this.formError.set(null);

    if (this.uses().length === 0) {
      this.formError.set('Marca al menos un uso: practicar en clase o entregar y calificar.');
      return;
    }
    if (this.requiresDescription() && !this.isDescriptionValid()) {
      this.formError.set(
        this.hasMissingPieces()
          ? 'Describe brevemente qué necesitan hacer los estudiantes (mínimo 10 caracteres) para evaluar las piezas faltantes.'
          : 'Elige al menos una herramienta del catálogo o cuéntanos qué van a hacer tus estudiantes (mínimo 10 caracteres).'
      );
      return;
    }

    this.isSubmitting.set(true);
    this.requestService
      .create({
        title: this.title().trim(),
        description: this.description().trim(),
        teacher_text: '',
        uses: this.uses(),
        stack_config: this.stackConfig(),
        missing_components: this.missingFromAI()
      })
      .subscribe({
        next: created => {
          this.isSubmitting.set(false);
          this.clearDraft();
          this.result.set({ message: created.message, joined: created.joined });
          this.requested.emit();
        },
        error: err => {
          this.isSubmitting.set(false);
          this.formError.set(
            err?.error?.message || err?.error?.error || 'No pudimos registrar la solicitud. Intente de nuevo.'
          );
        }
      });
  }

  /** Descarta el borrador en memoria y deja el formulario limpio. */
  clearDraft(): void {
    this.draftService.clear();
    this.aiOperation.cancel();
    this.aiOperation.reset();
    this.aiPrompt.set('');
    this.description.set('');
    this.descriptionTouched.set(false);
    this.title.set('');
    this.titleTouched.set(false);
    this.runtimeId.set('');
    this.webPlatformId.set('');
    this.databaseId.set('');
    this.addonIds.set([]);
    this.missingFromAI.set([]);
    this.aiNotice.set(null);
    this.showMissingNames.set(false);
    this.uses.set(['practice']);
    this.preview.set(null);
    this.formError.set(null);
  }

  close(): void {
    if (this.isSubmitting()) {
      return;
    }
    this.closed.emit();
  }

  trackChip = (_: number, chip: SelectedChip) => chip.id;
}


/** Mensaje de la consulta a la IA, sin depender del componente que la lanzó. */
function resolveSuggestionError(err: unknown): string {
  if (err instanceof TimeoutError) {
    return 'La IA no respondió en 100 segundos. El modelo puede estar saturado: vuelve a intentar o elige las herramientas a mano.';
  }
  const candidate = err as { status?: number; error?: { message?: string } | string };
  const body = candidate?.error;
  const detail = typeof body === 'string' ? '' : body?.message;
  if (detail) {
    return detail;
  }
  if (candidate?.status === 404) {
    return 'El servidor no reconoce la ruta de la IA. Suele pasar cuando el backend quedó desactualizado: reinícialo y vuelve a intentar.';
  }
  if (candidate?.status === 0) {
    return 'No pudimos contactar al servidor. Revisa que el backend esté corriendo y vuelve a intentar.';
  }
  return 'No pudimos interpretar la descripción. Puedes elegir las herramientas a mano.';
}
