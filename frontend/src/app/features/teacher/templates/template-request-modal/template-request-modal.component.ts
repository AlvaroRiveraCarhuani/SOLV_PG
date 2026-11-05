import {
  Component,
  ElementRef,
  OnInit,
  afterNextRender,
  computed,
  effect,
  inject,
  input,
  output,
  signal,
  viewChild
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
  LucideBoxes,
  LucideCheck,
  LucideChevronDown,
  LucideFlaskConical,
  LucideGraduationCap,
  LucideInfo,
  LucidePlus,
  LucideSearch,
  LucideSend,
  LucideWand,
  LucideWand2,
  LucideWrench,
  LucideX
} from '@lucide/angular';
import { AdminTemplatesService } from '@features/admin/services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { matchesRuntimeEcosystem } from '@features/admin/templates/components/model-library/model-library.utils';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
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

interface PickerGroup {
  id: string;
  label: string;
  hint: string;
  items: TemplateComponent[];
}

/**
 * Pedido de entorno del docente.
 *
 * Una sola pantalla: qué van a hacer los estudiantes, cómo lo van a usar y en
 * qué materia. El docente nunca escribe una imagen, una RAM ni un puerto: el
 * backend arma la receta con el catálogo y el panel lateral muestra, en las
 * mismas palabras, lo que va a recibir el administrador.
 */
@Component({
  selector: 'template-request-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    LucideAlertCircle,
    LucideBoxes,
    LucideCheck,
    LucideChevronDown,
    LucideFlaskConical,
    LucideGraduationCap,
    LucideInfo,
    LucidePlus,
    LucideSearch,
    LucideSend,
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
  private dashboardService = inject(TeacherDashboardService);
  private requestService = inject(EnvironmentRequestService);
  private draftService = inject(EnvironmentRequestDraftService);

  subjectId = input<string>('');
  subjectName = input<string>('');
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
  description = signal<string>('');
  title = signal<string>('');
  titleTouched = signal<boolean>(false);
  subjectIdSelected = signal<string>('');
  subjectNameSelected = signal<string>('');

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
  pickerQuery = signal<string>('');
  showMissingNames = signal<boolean>(false);
  showTechnical = signal<boolean>(false);
  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);
  result = signal<{ message: string; joined: boolean } | null>(null);
  private presetApplied = false;
  private practiceText = viewChild<ElementRef<HTMLTextAreaElement>>('practiceText');

  constructor() {
    // El borrador vive en la raíz: cerrar el modal no pierde lo escrito.
    const draft = this.draftService.draft();
    this.description.set(draft.description);
    this.title.set(draft.title);
    this.titleTouched.set(draft.titleTouched);
    this.subjectIdSelected.set(draft.subjectId);
    this.subjectNameSelected.set(draft.subjectName);
    this.uses.set(draft.uses.length ? draft.uses : ['practice']);
    this.runtimeId.set(draft.runtimeId);
    this.webPlatformId.set(draft.webPlatformId);
    this.databaseId.set(draft.databaseId);
    this.addonIds.set(draft.addonIds);
    this.missingFromAI.set(draft.missingFromAI);
    this.aiNotice.set(draft.aiNotice);

    // El foco entra al campo de la práctica: el docente llega escribiendo.
    afterNextRender(() => this.practiceText()?.nativeElement.focus());

    effect(() => {
      this.draftService.patch({
        description: this.description(),
        title: this.title(),
        titleTouched: this.titleTouched(),
        subjectId: this.subjectIdSelected(),
        subjectName: this.subjectNameSelected(),
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

  readonly subjects = computed(() => this.dashboardService.courses());

  /** Materias del docente para el combobox compartido. */
  readonly subjectOptions = computed<ComboboxOption[]>(() =>
    this.subjects().map(course => ({
      id: course.id,
      label: course.name,
      value: course.id,
      meta: course.code
    }))
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
        iconKey: found.icon_key
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

  /** Nombre que verá la administración: el escrito o el que propone el sistema. */
  readonly displayName = computed(() => {
    const written = this.title().trim();
    return written || this.preview()?.suggested_name || '';
  });

  /** Nombres que la IA nombró y que el catálogo todavía no tiene. */
  readonly missingNames = computed(() => this.missingFromAI().map(item => item.name));

  /**
   * Lista del selector, agrupada como el catálogo del estudio.
   *
   * Una grilla plana mezcla Rust, Angular y Prisma como si fueran lo mismo. Acá
   * el docente ve lenguajes, interfaz web, complementos y bases de datos, en ese
   * orden. Se muestra el catálogo completo: lo que no entra se desplaza, no se
   * recorta, para que una pieza no dependa de saber su nombre de memoria.
   */
  readonly pickerGroups = computed<PickerGroup[]>(() => {
    const query = this.pickerQuery().trim().toLowerCase();
    const matches = (item: TemplateComponent) =>
      !query || item.name.toLowerCase().includes(query) || item.slug.toLowerCase().includes(query);
    const byOrder = (a: TemplateComponent, b: TemplateComponent) =>
      (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.name.localeCompare(b.name);

    const items = this.catalog().filter(matches);
    const groups: PickerGroup[] = [
      {
        id: 'languages',
        label: 'Lenguajes y frameworks',
        hint: 'La base del entorno',
        items: items.filter(item => item.layer === 'runtime' && item.sub_category !== 'web').sort(byOrder)
      },
      {
        id: 'web',
        label: 'Interfaz web',
        hint: 'Opcional',
        items: items.filter(item => item.layer === 'runtime' && item.sub_category === 'web').sort(byOrder)
      },
      {
        id: 'addons',
        label: 'Complementos',
        hint: 'Frameworks, ORM y librerías',
        items: items.filter(item => item.layer === 'addon').sort(byOrder)
      },
      {
        id: 'databases',
        label: 'Bases de datos',
        hint: 'Una por entorno',
        items: items.filter(item => item.layer === 'database').sort(byOrder)
      }
    ];
    return groups.filter(group => group.items.length > 0);
  });

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
    if (this.subjectId()) {
      this.subjectIdSelected.set(this.subjectId());
      this.subjectNameSelected.set(this.subjectName());
    }
    if (!this.subjects().length) {
      this.dashboardService.loadDashboardData().subscribe({ next: () => this.syncSubjectName(), error: () => undefined });
    } else {
      this.syncSubjectName();
    }

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

  private syncSubjectName(): void {
    if (this.subjectNameSelected()) {
      return;
    }
    const id = this.subjectIdSelected();
    const match = this.subjects().find(course => course.id === id);
    if (match) {
      this.subjectNameSelected.set(match.name);
    }
  }

  onSubjectChange(value: string): void {
    const match = this.subjects().find(course => course.id === value);
    this.subjectIdSelected.set(match ? match.id : '');
    this.subjectNameSelected.set(match ? match.name : '');
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
      this.databaseId.set(item.id);
    } else if (item.layer === 'runtime' && item.sub_category === 'web') {
      this.webPlatformId.set(item.id);
    } else if (item.layer === 'runtime') {
      this.runtimeId.set(item.id);
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
    return {
      mode: 'visual',
      runtime_id: this.runtimeId() || undefined,
      web_platform_id: this.webPlatformId() || undefined,
      addon_ids: this.addonIds().length ? this.addonIds() : undefined,
      database_id: this.databaseId() || undefined
    };
  }

  refreshPreview(): void {
    this.previewLoading.set(true);
    this.requestService.preview(this.uses(), this.stackConfig()).subscribe({
      next: preview => {
        this.preview.set(preview);
        this.previewLoading.set(false);
        if (!this.titleTouched() && preview.suggested_name) {
          this.title.set(preview.suggested_name);
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

  // --- IA --------------------------------------------------------------------

  /** Corta la espera: el docente no queda atado a un modelo saturado. */
  cancelAI(): void {
    this.aiOperation.cancel();
    this.aiNotice.set('Detuviste la consulta. Puedes volver a intentar o elegir las herramientas a mano.');
  }

  interpret(): void {
    const prompt = this.description().trim();
    if (prompt.length < 10) {
      this.formError.set('Cuéntanos un poco más: qué deben construir o resolver los estudiantes.');
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
    if (runtime) {
      this.runtimeId.set(runtime.id);
    }

    const web = this.findCatalogPiece(suggestion.web_platform_slug);
    this.webPlatformId.set(web ? web.id : '');

    const db = this.findCatalogPiece(suggestion.database_slug);
    this.databaseId.set(db ? db.id : '');

    const addons = (suggestion.addon_slugs || [])
      .map(slug => this.findCatalogPiece(slug))
      .filter((item): item is TemplateComponent => Boolean(item))
      .map(item => item.id);
    this.addonIds.set(addons);

    this.missingFromAI.set(suggestion.missing_components || []);

    if (!this.titleTouched() && suggestion.name) {
      this.title.set(suggestion.name);
      this.titleTouched.set(true);
    }

    this.aiNotice.set(
      suggestion.missing_components?.length
        ? 'Esto es lo que entendí. De lo que nombraste, unas piezas no están en el catálogo: las pedimos junto con tu solicitud.'
        : 'Esto es lo que entendí. Puedes quitar o agregar herramientas antes de enviar.'
    );
    this.refreshPreview();
  }

  // --- envío ----------------------------------------------------------------

  submitRequest(): void {
    this.formError.set(null);

    if (this.description().trim().length < 10) {
      this.formError.set('Cuéntanos qué van a hacer tus estudiantes: la administración decide con ese texto.');
      return;
    }
    if (this.uses().length === 0) {
      this.formError.set('Marca al menos un uso: practicar en clase o entregar y calificar.');
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
        missing_components: this.missingFromAI(),
        subject_id: this.subjectIdSelected() || undefined,
        subject_name: this.subjectNameSelected() || undefined
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
            err?.error?.message || err?.error?.error || 'No pudimos registrar la solicitud. Intenta de nuevo.'
          );
        }
      });
  }

  /** La solicitud ya salió: el borrador cumplió su función. */
  private clearDraft(): void {
    this.draftService.clear();
    this.aiOperation.cancel();
    this.aiOperation.reset();
    this.description.set('');
    this.title.set('');
    this.titleTouched.set(false);
    this.runtimeId.set('');
    this.webPlatformId.set('');
    this.databaseId.set('');
    this.addonIds.set([]);
    this.missingFromAI.set([]);
    this.aiNotice.set(null);
    this.uses.set(['practice']);
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
