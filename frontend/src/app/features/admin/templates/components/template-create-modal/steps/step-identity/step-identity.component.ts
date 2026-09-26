import { 
  Component, 
  input, 
  output, 
  signal, 
  computed, 
  ElementRef, 
  viewChild 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { 
  LucideSparkles, 
  LucideSearch, 
  LucideRotateCw, 
  LucideCopy, 
  LucideHelpCircle,
  LucideCheck,
  LucideX,
  LucideInfo
} from '@lucide/angular';
import { 
  ComboboxComponent, 
  ComboboxOption 
} from '../../../../../../../shared/components/combobox/combobox.component';
import { 
  TemplateModelItem, 
  TemplateCategory, 
  AdminTemplateItem,
  TargetEnvironment 
} from '../../../../../services/admin-templates.service';

export type CreationMode = 'blank' | 'recipe' | 'duplicate';

@Component({
  selector: 'step-identity',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    RouterModule,
    LucideSparkles, 
    LucideSearch, 
    LucideRotateCw, 
    LucideCopy, 
    LucideHelpCircle,
    LucideCheck,
    LucideX,
    LucideInfo,
    ComboboxComponent
  ],
  template: `
    <div class="step-header-with-help mb-3">
      <h4 class="step-section-title" id="step-title-identity" tabindex="-1" i18n="@@ST-01-HEADING">
        Identidad de la plantilla
      </h4>
      <button 
        type="button" 
        class="btn-step-help" 
        (click)="helpRequested.emit()" 
        title="Ayuda contextual del paso" 
        aria-label="Ayuda contextual del paso"
        i18n-aria-label="@@AY-02"
      >
        <svg lucideHelpCircle class="w-4 h-4"></svg>
      </button>
    </div>

    <!-- Menú de Entrada de 3 Puertas -->
    <div class="entry-doors-strip">
      <div class="doors-label">
        <span class="doors-title" i18n="@@ST-10">Nueva plantilla</span>
      </div>
      <div class="doors-options">
        <button 
          type="button" 
          class="door-chip" 
          [class.active]="creationMode() === 'blank'"
          (click)="setCreationMode('blank')"
        >
          <span i18n="@@ST-11">En blanco</span>
        </button>
        <button 
          type="button" 
          class="door-chip" 
          [class.active]="creationMode() === 'recipe'"
          (click)="setCreationMode('recipe')"
        >
          <span i18n="@@ST-12">Desde modelo</span>
        </button>
        <button 
          type="button" 
          class="door-chip" 
          [class.active]="creationMode() === 'duplicate'"
          (click)="setCreationMode('duplicate')"
        >
          <span i18n="@@ST-13">Duplicar existente</span>
        </button>
      </div>
    </div>

    <!-- Puerta: Desde modelo de plantilla -->
    @if (creationMode() === 'recipe') {
      <div class="models-shelf-section">
        <div class="models-shelf-header">
          <div class="models-title-wrap">
            <svg lucideSparkles class="w-4 h-4 text-warning mr-1.5"></svg>
            <span class="models-shelf-title" i18n="@@MO-01">Modelos de plantilla</span>
            <span class="models-shelf-sub" i18n="@@MO-02">Entornos preconfigurados listos para usar o personalizar</span>
          </div>
          @if (templateModels().length > 4 || modelsSearch()) {
            <div class="models-search-bar">
              <svg lucideSearch class="w-3.5 h-3.5 text-muted mr-1.5"></svg>
              <input 
                type="text" 
                class="models-search-input" 
                [ngModel]="modelsSearch()"
                (ngModelChange)="onSearchChange($event)"
                placeholder="Buscar modelos..."
                i18n-placeholder="@@MO-03"
              />
            </div>
          }
        </div>

        <!-- Línea informativa de descubribilidad de promoción -->
        <div class="models-promotion-banner">
          <div class="banner-content">
            <svg lucideInfo class="w-4 h-4 text-info mr-2 flex-shrink-0"></svg>
            <span class="banner-text" i18n="@@AY-19">
              Los modelos se originan a partir de plantillas aprobadas promovidas desde el catálogo o de configuraciones base institucionales.
            </span>
          </div>
          <a routerLink="/admin/manual" class="banner-link" target="_blank" rel="noopener noreferrer" i18n="@@AY-20">
            Ver manual de promoción
          </a>
        </div>

        <!-- Fila de chips de filtro por categoría (dato real de BD) -->
        @if (categoryFilterChips().length > 1) {
          <div class="models-category-chips-row">
            @for (chip of categoryFilterChips(); track chip.id) {
              <button 
                type="button" 
                class="cat-filter-chip" 
                [class.active]="selectedCategoryFilter() === chip.id"
                (click)="selectCategoryFilter(chip.id)"
              >
                {{ chip.label }}
              </button>
            }
          </div>
        }

        @if (filteredModels().length === 0) {
          <div class="models-empty-state">
            @if (targetEnvironment() === 'JUEZ_EFIMERO') {
              <p i18n="@@PU-12">No hay modelos de juez registrados todavía. Puede comenzar con una plantilla en blanco.</p>
            } @else {
              <span i18n="@@MO-07">No se encontraron modelos para la búsqueda</span>
            }
            <button type="button" class="btn btn-sm btn-outline-primary mt-2" (click)="setCreationMode('blank')">
              Comenzar en blanco
            </button>
          </div>
        } @else {
          <div class="disciplines-container">
            @for (group of visibleGroups(); track group.categoryName) {
              <div class="discipline-group">
                <div class="discipline-badge">{{ group.categoryName }}</div>
                <div class="models-grid">
                  @for (model of group.models; track model.id) {
                    <div class="model-card" [class.selected]="recipeUsed() === model.id">
                      <div class="model-card-header">
                        <span class="model-card-title">{{ model.name }}</span>
                        <span class="model-usage-badge" [title]="'Usado ' + model.usage_count + ' veces'">
                          <svg lucideRotateCw class="w-3 h-3 mr-1"></svg>
                          <span i18n="@@MO-05">{{ model.usage_count }} plantillas basadas en este modelo</span>
                        </span>
                      </div>
                      <p class="model-card-desc">{{ model.description || 'Sin descripción adicional' }}</p>
                      <div class="model-card-tools">
                        <span class="tools-label">Imagen: </span>
                        <span class="tools-val font-mono">{{ model.docker_image }} ({{ model.base_ram_mb }} MB)</span>
                      </div>
                      <button 
                        type="button" 
                        class="btn btn-sm btn-outline-primary mt-2 w-full"
                        (click)="applyModel.emit(model)"
                      >
                        <span i18n="@@MO-06">Usar este modelo</span>
                      </button>
                    </div>
                  }
                </div>
              </div>
            }
          </div>

          @if (hasMoreModels()) {
            <div class="models-show-more-row">
              <button 
                type="button" 
                class="btn btn-sm btn-outline-primary btn-show-more"
                (click)="showMoreModels()"
              >
                Mostrar más
              </button>
            </div>
          }
        }
      </div>
    }

    <!-- Puerta: Duplicar existente -->
    @if (creationMode() === 'duplicate') {
      <div class="models-shelf-section">
        <div class="models-shelf-header">
          <div class="models-title-wrap">
            <svg lucideCopy class="w-4 h-4 text-primary mr-1.5"></svg>
            <span class="models-shelf-title" i18n="@@MO-13">Duplicar plantilla existente</span>
            <span class="models-shelf-sub" i18n="@@MO-14">Cree una copia editable a partir de una plantilla del catálogo</span>
          </div>
        </div>

        @if (templatesList().length === 0) {
          <div class="models-empty-state">
            <span>No hay plantillas disponibles para duplicar</span>
            <button type="button" class="btn btn-sm btn-outline-primary mt-2" (click)="setCreationMode('blank')">
              Comenzar en blanco
            </button>
          </div>
        } @else {
          <div class="duplicate-list">
            @for (item of templatesList(); track item.id) {
              <div class="duplicate-card">
                <div class="duplicate-card-info">
                  <span class="duplicate-card-title">{{ item.name }}</span>
                  <span class="duplicate-card-image font-mono">{{ item.docker_image }}</span>
                </div>
                <button 
                  type="button" 
                  class="btn btn-sm btn-outline-primary"
                  (click)="duplicateTemplate.emit(item)"
                >
                  <svg lucideCopy class="w-3.5 h-3.5 mr-1"></svg>
                  <span i18n="@@MO-15">Duplicar esta plantilla</span>
                </button>
              </div>
            }
          </div>
        }
      </div>
    }

    <!-- Formulario de Identidad -->
    <div class="form-group mt-4">
      <label class="form-label font-semibold" for="template-name-input">
        Nombre de la plantilla: <span class="text-danger">*</span>
      </label>
      <input 
        #nameInput
        id="template-name-input"
        type="text" 
        class="form-control" 
        placeholder="Ej: Programación Web con Node 20" 
        [ngModel]="name()" 
        (ngModelChange)="nameChange.emit($event)"
        (keydown.enter)="onNameEnter()"
      />
      <small class="form-hint">Nombre descriptivo que verán los docentes y estudiantes en el catálogo.</small>
    </div>

    <!-- Categoría Académica con Combobox Buscable y Creación Inline -->
    <div class="form-group">
      <label class="form-label font-semibold">
        Categoría académica (Opcional):
      </label>

      @if (!isCreatingCategoryInline()) {
        <combobox
          [value]="selectedCategoryLabel()"
          [options]="categoryComboboxOptions()"
          placeholder="Seleccione o busque una categoría..."
          inputAriaLabel="Categoría académica"
          (optionSelected)="onCategoryOptionSelected($event)"
          (valueChange)="onCategorySearchChange($event)"
        ></combobox>
      } @else {
        <div class="inline-category-row">
          <input 
            #newCatInput
            type="text" 
            class="form-control inline-cat-input" 
            placeholder="Nombre de la nueva categoría..." 
            [ngModel]="inlineCategoryName()" 
            (ngModelChange)="inlineCategoryName.set($event)"
            (keydown.enter)="createCategoryInline()"
            (keydown.escape)="cancelInlineCategory()"
          />
          <div class="inline-cat-actions">
            <button 
              type="button" 
              class="btn btn-sm btn-primary"
              [disabled]="!inlineCategoryName().trim() || isSavingCategory()" 
              (click)="createCategoryInline()"
            >
              <svg lucideCheck class="w-3.5 h-3.5 mr-1"></svg>
              <span>{{ isSavingCategory() ? 'Guardando...' : 'Confirmar' }}</span>
            </button>
            <button 
              type="button" 
              class="btn btn-sm btn-outline-secondary"
              (click)="cancelInlineCategory()"
            >
              <svg lucideX class="w-3.5 h-3.5 mr-1"></svg>
              <span>Cancelar</span>
            </button>
          </div>
        </div>
      }
      <small class="form-hint">Agrupa plantillas afines en el catálogo institucional.</small>
    </div>

    <!-- Descripción opcional -->
    <div class="form-group">
      <label class="form-label font-semibold" for="template-desc-input">
        Descripción:
      </label>
      <textarea
        id="template-desc-input"
        class="form-control"
        rows="2"
        placeholder="Breve descripción pedagógica o técnica del entorno..."
        [ngModel]="description()"
        (ngModelChange)="descriptionChange.emit($event)"
      ></textarea>
    </div>
  `,
  styleUrls: ['./step-identity.component.scss']
})
export class StepIdentityComponent {
  nameInput = viewChild<ElementRef<HTMLInputElement>>('nameInput');
  newCatInput = viewChild<ElementRef<HTMLInputElement>>('newCatInput');

  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  creationMode = input<CreationMode>('blank');
  recipeUsed = input<string | null>(null);
  name = input<string>('');
  description = input<string>('');
  selectedCategoryId = input<string | null>(null);
  categories = input<TemplateCategory[]>([]);
  templateModels = input<TemplateModelItem[]>([]);
  templatesList = input<AdminTemplateItem[]>([]);
  isSavingCategory = input<boolean>(false);

  creationModeChange = output<CreationMode>();
  nameChange = output<string>();
  descriptionChange = output<string>();
  categorySelected = output<string | null>();
  createCategory = output<string>();
  applyModel = output<TemplateModelItem>();
  duplicateTemplate = output<AdminTemplateItem>();
  helpRequested = output<void>();
  advance = output<void>();

  modelsSearch = signal<string>('');
  selectedCategoryFilter = signal<string | null>(null);
  modelsLimit = signal<number>(6);
  isCreatingCategoryInline = signal<boolean>(false);
  inlineCategoryName = signal<string>('');

  getCategoryName(m: TemplateModelItem): string {
    if (m.category_id) {
      const cat = this.categories().find(c => c.id === m.category_id);
      if (cat?.name) return cat.name;
    }
    if (m.category_name) return m.category_name;
    return 'Sin categoría';
  }

  categoryFilterChips = computed<{ id: string | null; label: string }[]>(() => {
    const cats = this.categories();
    const chips: { id: string | null; label: string }[] = [
      { id: null, label: 'Todas' }
    ];
    for (const c of cats) {
      chips.push({ id: c.id, label: c.name });
    }
    return chips;
  });

  filteredModels = computed<TemplateModelItem[]>(() => {
    const list = this.templateModels();
    const query = this.modelsSearch().trim().toLowerCase();
    const catFilter = this.selectedCategoryFilter();

    return list.filter(m => {
      if (catFilter !== null && m.category_id !== catFilter) {
        return false;
      }
      if (query) {
        const matchesName = m.name.toLowerCase().includes(query);
        const matchesDesc = !!m.description && m.description.toLowerCase().includes(query);
        const matchesImage = m.docker_image.toLowerCase().includes(query);
        const catName = this.getCategoryName(m).toLowerCase();
        const matchesCat = catName.includes(query);
        if (!matchesName && !matchesDesc && !matchesImage && !matchesCat) {
          return false;
        }
      }
      return true;
    });
  });

  visibleModels = computed<TemplateModelItem[]>(() => {
    return this.filteredModels().slice(0, this.modelsLimit());
  });

  hasMoreModels = computed<boolean>(() => {
    return this.filteredModels().length > this.modelsLimit();
  });

  visibleGroups = computed<{ categoryName: string; models: TemplateModelItem[] }[]>(() => {
    const map = new Map<string, TemplateModelItem[]>();
    for (const m of this.visibleModels()) {
      const name = this.getCategoryName(m);
      if (!map.has(name)) {
        map.set(name, []);
      }
      map.get(name)!.push(m);
    }
    return Array.from(map.entries()).map(([categoryName, models]) => ({
      categoryName,
      models
    }));
  });

  selectCategoryFilter(catId: string | null): void {
    this.selectedCategoryFilter.set(catId);
    this.modelsLimit.set(6);
  }

  onSearchChange(query: string): void {
    this.modelsSearch.set(query);
    this.modelsLimit.set(6);
  }

  showMoreModels(): void {
    this.modelsLimit.update(lim => lim + 6);
  }

  categoryComboboxOptions = computed<ComboboxOption[]>(() => {
    const cats = this.categories();
    const opts: ComboboxOption[] = [];
    for (const c of cats) {
      opts.push({ id: c.id, label: c.name, value: c.id });
    }
    opts.push({ 
      id: '__new__', 
      label: '＋ Crear nueva categoría...', 
      value: '__new__',
      badge: 'Nuevo',
      badgeVariant: 'official'
    });
    return opts;
  });

  selectedCategoryLabel = computed<string>(() => {
    const id = this.selectedCategoryId();
    if (!id) return '';
    const found = this.categories().find(c => c.id === id);
    return found ? found.name : '';
  });

  setCreationMode(mode: CreationMode): void {
    this.creationModeChange.emit(mode);
  }

  onNameEnter(): void {
    const descEl = document.getElementById('template-desc-input');
    descEl?.focus();
  }

  onCategoryOptionSelected(opt: ComboboxOption): void {
    if (opt.id === '__new__') {
      this.isCreatingCategoryInline.set(true);
      this.inlineCategoryName.set('');
      setTimeout(() => this.newCatInput()?.nativeElement.focus(), 50);
    } else {
      this.categorySelected.emit(opt.value);
    }
  }

  onCategorySearchChange(query: string): void {
    // Si el usuario escribe y no existe, se mantiene la búsqueda interna de combobox
  }

  createCategoryInline(): void {
    const name = this.inlineCategoryName().trim();
    if (!name) return;
    this.createCategory.emit(name);
    this.isCreatingCategoryInline.set(false);
  }

  cancelInlineCategory(): void {
    this.isCreatingCategoryInline.set(false);
    this.inlineCategoryName.set('');
  }
}

export { StepIdentityComponent as SolvStepIdentityComponent };
