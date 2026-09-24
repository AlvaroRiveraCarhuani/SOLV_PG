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
import { 
  LucideSparkles, 
  LucideSearch, 
  LucideRotateCw, 
  LucideCopy, 
  LucideHelpCircle,
  LucideCheck,
  LucideX
} from '@lucide/angular';
import { 
  SolvComboboxComponent, 
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
  selector: 'solv-step-identity',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideSparkles, 
    LucideSearch, 
    LucideRotateCw, 
    LucideCopy, 
    LucideHelpCircle,
    LucideCheck,
    LucideX,
    SolvComboboxComponent
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
          @if (filteredModels().length > 4) {
            <div class="models-search-bar">
              <svg lucideSearch class="w-3.5 h-3.5 text-muted mr-1.5"></svg>
              <input 
                type="text" 
                class="models-search-input" 
                [ngModel]="modelsSearch()"
                (ngModelChange)="modelsSearch.set($event)"
                placeholder="Buscar modelos..."
                i18n-placeholder="@@MO-03"
              />
            </div>
          }
        </div>

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
            @for (catName of availableModelCategories(); track catName) {
              <div class="discipline-group">
                <div class="discipline-badge">{{ catName }}</div>
                <div class="models-grid">
                  @for (model of modelsByCategory(catName); track model.id) {
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
                        <span class="tools-label">Imagen:</span>
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
        Categoría académica:
      </label>

      @if (!isCreatingCategoryInline()) {
        <solv-combobox
          [value]="selectedCategoryLabel()"
          [options]="categoryComboboxOptions()"
          placeholder="Seleccione o busque una categoría..."
          inputAriaLabel="Categoría académica"
          (optionSelected)="onCategoryOptionSelected($event)"
          (valueChange)="onCategorySearchChange($event)"
        ></solv-combobox>
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
export class SolvStepIdentityComponent {
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
  isCreatingCategoryInline = signal<boolean>(false);
  inlineCategoryName = signal<string>('');

  filteredModels = computed<TemplateModelItem[]>(() => {
    const list = this.templateModels();
    const query = this.modelsSearch().trim().toLowerCase();
    if (!query) return list;
    return list.filter(m => 
      m.name.toLowerCase().includes(query) || 
      (m.description && m.description.toLowerCase().includes(query)) ||
      m.docker_image.toLowerCase().includes(query)
    );
  });

  availableModelCategories = computed<string[]>(() => {
    const categoriesSet = new Set<string>();
    for (const m of this.filteredModels()) {
      categoriesSet.add(m.category_id || 'Sin Categoría');
    }
    return Array.from(categoriesSet);
  });

  categoryComboboxOptions = computed<ComboboxOption[]>(() => {
    const cats = this.categories();
    const opts: ComboboxOption[] = [
      { id: '__none__', label: '-- Sin categoría asignada --', value: null }
    ];
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

  modelsByCategory(catName: string): TemplateModelItem[] {
    return this.filteredModels().filter(m => (m.category_id || 'Sin Categoría') === catName);
  }

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
    } else if (opt.id === '__none__') {
      this.categorySelected.emit(null);
    } else {
      this.categorySelected.emit(opt.value);
    }
  }

  onCategorySearchChange(query: string): void {
    // Si el usuario escribe y no existe, se mantiene la búsqueda interna de solv-combobox
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
