import { 
  Component, 
  input, 
  output, 
  signal, 
  computed 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideHelpCircle 
} from '@lucide/angular';
import { 
  SolvComboboxComponent, 
  ComboboxOption 
} from '../../../../../../../shared/components/combobox/combobox.component';
import { 
  SolvFieldMessageComponent 
} from '../../../../../../../shared/components/field-message/field-message.component';
import { 
  LocalImageItem, 
  ImageSuggestion,
  TargetEnvironment 
} from '../../../../../services/admin-templates.service';

@Component({
  selector: 'solv-step-image',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideHelpCircle,
    SolvComboboxComponent,
    SolvFieldMessageComponent
  ],
  template: `
    <div class="form-group">
      <div class="label-row-between">
        <div class="label-with-help">
          <label class="form-label font-semibold">
            Imagen Docker (Repositorio:Tag): <span class="text-danger">*</span>
          </label>
          <button 
            type="button" 
            class="btn-help-inline" 
            (click)="showImagePopover.set(!showImagePopover())" 
            title="Anatomía de una referencia de imagen"
            aria-label="Anatomía de una referencia de imagen"
          >
            <svg lucideHelpCircle class="w-3.5 h-3.5 text-primary"></svg>
          </button>
        </div>
      </div>

      <!-- Popover anatómico OCI -->
      @if (showImagePopover()) {
        <div class="image-popover-box animate-fade">
          <div class="popover-header">
            <h6 class="popover-title" i18n="@@AY-13">Anatomía de una referencia de imagen Docker</h6>
            <button 
              type="button" 
              class="btn-close-popover" 
              (click)="showImagePopover.set(false)" 
              aria-label="Cerrar"
            >×</button>
          </div>
          <div class="popover-content">
            <div class="image-anatomy-breakdown">
              <div class="anatomy-part"><span class="part-label">Registro:</span> <code class="part-code">docker.io/</code></div>
              <div class="anatomy-part"><span class="part-label">Namespace:</span> <code class="part-code">library/</code></div>
              <div class="anatomy-part"><span class="part-label">Repositorio:</span> <code class="part-code font-bold">python</code></div>
              <div class="anatomy-part"><span class="part-label">Tag fijado:</span> <code class="part-code tag-code">:3.12-slim-bookworm</code></div>
            </div>
            <p class="popover-warning" i18n="@@AY-14">
              El tag :latest está prohibido por reproducibilidad académica y gobernanza institucional.
            </p>
          </div>
        </div>
      }

      <!-- Combobox de imagen (solo filtra) -->
      <solv-combobox
        [value]="dockerImage()"
        [options]="comboboxOptions()"
        [maxSuggestions]="8"
        [totalAvailableCount]="totalImagesCount()"
        [suppressListbox]="isInvalidFormat() || isLatestImage()"
        placeholder="Ej: python:3.12-slim-bookworm"
        inputAriaLabel="Imagen Docker"
        (valueChange)="onImageChange($event)"
        (optionSelected)="onImageOptionSelected($event)"
      ></solv-combobox>

      <!-- Mensajes de verificación y validación con semántica de color (separados del listbox) -->
      @if (isLatestImage()) {
        <solv-field-message 
          variant="error"
          message="Prohibido el tag :latest por reproducibilidad académica y gobernanza institucional."
        ></solv-field-message>
      } @else if (isInvalidFormat()) {
        <solv-field-message 
          variant="warning"
          message="Formato OCI inválido: debe ser repositorio:tag (ej: python:3.12-slim)."
        ></solv-field-message>
      } @else if (verificationState() === 'checking') {
        <solv-field-message 
          variant="info"
          message="Verificando imagen en el nodo y registro Docker..."
        ></solv-field-message>
      } @else if (verificationState() === 'verified' && !isStorageBlocked() && !isArchIncompatible()) {
        <solv-field-message 
          variant="success"
          [message]="verificationSuccessMessage()"
          actionLabel="Re-verificar"
          (actionClicked)="reverify.emit()"
        ></solv-field-message>
      } @else if (verificationState() === 'failed' || verificationState() === 'error') {
        <solv-field-message 
          variant="error"
          message="No se pudo verificar la imagen en el nodo ni en el registro público."
          actionLabel="Reintentar verificación"
          (actionClicked)="reverify.emit()"
        ></solv-field-message>
      } @else if (isStorageBlocked()) {
        <solv-field-message 
          variant="error"
          message="El tamaño de la imagen excede la cuota de almacenamiento del nodo."
        ></solv-field-message>
      } @else if (isArchIncompatible()) {
        <solv-field-message 
          variant="error"
          message="La arquitectura de la imagen es incompatible con el host (requiere amd64)."
        ></solv-field-message>
      }
    </div>

    <!-- Herramientas declaradas con Chips reactivos a la familia de la imagen -->
    <div class="form-group mt-3">
      <label class="form-label font-semibold" for="tools-declared-input">
        Herramientas requeridas en la imagen (separadas por coma):
      </label>
      <input 
        id="tools-declared-input"
        type="text" 
        class="form-control font-mono" 
        [placeholder]="suggestedToolsPlaceholder()" 
        [ngModel]="toolsDeclared()" 
        (ngModelChange)="toolsDeclaredChange.emit($event)" 
      />
      
      <div class="tools-suggested-chips mt-2">
        <span class="chips-label" i18n="@@AY-15">
          {{ hasImageFamilyMatch() ? 'Herramientas sugeridas para esta imagen:' : 'Herramientas sugeridas para este propósito:' }}
        </span>
        <div class="chips-row">
          @for (tool of reactiveSuggestedTools(); track tool) {
            <button 
              type="button" 
              class="tool-chip" 
              [class.added]="isToolDeclared(tool)"
              (click)="addTool(tool)"
            >
              + {{ tool }}
            </button>
          }
        </div>
      </div>
      <small class="form-hint">
        Binarios que el sistema comprobará dentro del contenedor antes de aprobar la plantilla.
      </small>
    </div>
  `,
  styleUrls: ['./step-image.component.scss']
})
export class SolvStepImageComponent {
  dockerImage = input<string>('');
  toolsDeclared = input<string>('');
  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  localImages = input<LocalImageItem[]>([]);
  curatedImages = input<ImageSuggestion[]>([]);
  usageMap = input<Record<string, number>>({});
  verificationState = input<'idle' | 'checking' | 'verified' | 'error' | 'failed'>('idle');
  isStorageBlocked = input<boolean>(false);
  isArchIncompatible = input<boolean>(false);
  archDetected = input<string>('amd64');
  digestDetected = input<string>('');

  dockerImageChange = output<string>();
  toolsDeclaredChange = output<string>();
  imageSelected = output<string>();
  reverify = output<void>();

  showImagePopover = signal<boolean>(false);

  private dockerRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$/;

  isLatestImage = computed<boolean>(() => {
    const img = this.dockerImage().trim().toLowerCase();
    if (!img) return false;
    return img.endsWith(':latest');
  });

  isInvalidFormat = computed<boolean>(() => {
    const img = this.dockerImage().trim();
    if (!img) return false;
    return !this.dockerRegex.test(img);
  });

  totalImagesCount = computed<number>(() => {
    return this.localImages().length + this.curatedImages().length;
  });

  // Mapeo unificado para el Combobox agrupado y acotado
  comboboxOptions = computed<ComboboxOption[]>(() => {
    const opts: ComboboxOption[] = [];
    const usage = this.usageMap();

    // Grupo 1: Locales en el daemon
    for (const img of this.localImages()) {
      opts.push({
        id: 'local-' + img.repo_tag,
        label: img.repo_tag,
        value: img.repo_tag,
        group: 'En este servidor (despliegue inmediato)',
        badge: 'Local',
        badgeVariant: 'local',
        meta: img.size_mb ? `${img.size_mb} MB` : undefined,
        description: (img.usage_count && img.usage_count > 0) 
          ? `Usada en ${img.usage_count} ${img.usage_count === 1 ? 'plantilla activa' : 'plantillas activas'}` 
          : undefined
      });
    }

    // Grupo 2: Catálogo curado oficial ordenado por uso institucional real
    const curatedSorted = [...this.curatedImages()].sort((a, b) => {
      const uA = usage[a.repoTag] || 0;
      const uB = usage[b.repoTag] || 0;
      return uB - uA;
    });

    for (const cur of curatedSorted) {
      const count = usage[cur.repoTag] || 0;
      opts.push({
        id: 'curated-' + cur.repoTag,
        label: cur.repoTag,
        value: cur.repoTag,
        group: 'Catálogo oficial curado',
        badge: 'Oficial',
        badgeVariant: 'official',
        description: count > 0 
          ? `${cur.description || ''} · Usada en ${count} plantillas`
          : cur.description
      });
    }

    return opts;
  });

  // Familias curadas de herramientas reactivas
  private imageFamilyToolsMap: Record<string, string[]> = {
    'python': ['python3', 'pip', 'pytest'],
    'node': ['node', 'npm', 'npx'],
    'golang': ['go', 'gofmt'],
    'gcc': ['gcc', 'g++', 'make'],
    'eclipse-temurin': ['javac', 'java', 'jar'],
    'openjdk': ['javac', 'java', 'jar'],
    'rust': ['rustc', 'cargo'],
    'postgres': ['psql', 'pg_dump'],
    'redis': ['redis-cli', 'redis-server']
  };

  hasImageFamilyMatch = computed<boolean>(() => {
    const img = this.dockerImage().toLowerCase();
    for (const prefix of Object.keys(this.imageFamilyToolsMap)) {
      if (img.includes(prefix)) {
        return true;
      }
    }
    return false;
  });

  reactiveSuggestedTools = computed<string[]>(() => {
    const img = this.dockerImage().toLowerCase();
    for (const [prefix, tools] of Object.entries(this.imageFamilyToolsMap)) {
      if (img.includes(prefix)) {
        return tools;
      }
    }
    // Fallback por propósito
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') {
      return ['gcc', 'python3', 'javac'];
    }
    return ['python3', 'node', 'gcc'];
  });

  suggestedToolsPlaceholder = computed<string>(() => {
    return this.targetEnvironment() === 'JUEZ_EFIMERO'
      ? 'Ej: gcc, python3, javac'
      : 'Ej: python3, node, gcc, git';
  });

  verificationSuccessMessage = computed<string>(() => {
    const arch = this.archDetected() || 'amd64';
    const digest = this.digestDetected();
    const shortDigest = digest ? ` · ${digest.substring(0, 12)}...` : '';
    return `Imagen verificada en el nodo (${arch}${shortDigest})`;
  });

  onImageChange(value: string): void {
    this.dockerImageChange.emit(value);
  }

  onImageOptionSelected(option: ComboboxOption): void {
    this.dockerImageChange.emit(option.value);
    this.imageSelected.emit(option.value);
  }

  isToolDeclared(tool: string): boolean {
    const list = this.toolsDeclared()
      .split(',')
      .map(t => t.trim())
      .filter(Boolean);
    return list.includes(tool);
  }

  addTool(tool: string): void {
    const list = this.toolsDeclared()
      .split(',')
      .map(t => t.trim())
      .filter(Boolean);
    if (!list.includes(tool)) {
      const updated = [...list, tool].join(', ');
      this.toolsDeclaredChange.emit(updated);
    }
  }
}
