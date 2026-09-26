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
  ComboboxComponent, 
  ComboboxOption 
} from '../../../../../../../shared/components/combobox/combobox.component';
import { 
  FieldMessageComponent 
} from '../../../../../../../shared/components/field-message/field-message.component';
import { 
  LocalImageItem, 
  ImageSuggestion,
  TargetEnvironment
} from '../../../../../services/admin-templates.service';

interface ToolEcosystem {
  name: string;
  imageKeywords: string[];
  tools: string[];
}

const TOOL_ECOSYSTEMS: ToolEcosystem[] = [
  {
    name: 'Python',
    imageKeywords: ['python'],
    tools: ['python', 'python3', 'pip', 'pip3', 'pytest', 'poetry', 'uv', 'virtualenv', 'mypy', 'black', 'flake8']
  },
  {
    name: 'Node.js',
    imageKeywords: ['node', 'javascript', 'typescript'],
    tools: ['node', 'nodejs', 'npm', 'npx', 'yarn', 'pnpm', 'bun', 'deno', 'tsc']
  },
  {
    name: 'Go',
    imageKeywords: ['golang', 'go:'],
    tools: ['go', 'gofmt', 'golint']
  },
  {
    name: 'C / C++',
    imageKeywords: ['gcc', 'clang', 'cpp'],
    tools: ['gcc', 'g++', 'clang', 'clang++', 'make', 'cmake', 'gdb', 'ninja']
  },
  {
    name: 'Java',
    imageKeywords: ['openjdk', 'eclipse-temurin', 'java', 'maven', 'gradle'],
    tools: ['javac', 'java', 'jar', 'mvn', 'gradle']
  },
  {
    name: 'Rust',
    imageKeywords: ['rust'],
    tools: ['rustc', 'cargo']
  },
  {
    name: 'Bases de datos',
    imageKeywords: ['postgres', 'mysql', 'redis', 'mariadb'],
    tools: ['psql', 'pg_dump', 'mysql', 'redis-cli', 'redis-server', 'mongosh']
  }
];

const UNIVERSAL_TOOLS = new Set([
  'bash', 'sh', 'zsh', 'git', 'curl', 'wget', 'tar', 'gzip', 'unzip', 'zip',
  'cat', 'ls', 'grep', 'awk', 'sed', 'sudo', 'env', 'jq', 'nano', 'vim', 'vi',
  'ssh', 'openssl', 'find', 'which', 'echo'
]);

@Component({
  selector: 'step-image',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideHelpCircle,
    ComboboxComponent,
    FieldMessageComponent
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
      <combobox
        [value]="dockerImage()"
        [options]="comboboxOptions()"
        [maxSuggestions]="8"
        [totalAvailableCount]="totalImagesCount()"
        [suppressListbox]="isInvalidFormat() || isLatestImage()"
        placeholder="Ej: python:3.12-slim-bookworm"
        inputAriaLabel="Imagen Docker"
        (valueChange)="onImageChange($event)"
        (optionSelected)="onImageOptionSelected($event)"
      ></combobox>

      <!-- Mensajes de verificación y validación con semántica de color (separados del listbox) -->
      @if (isLatestImage()) {
        <field-message 
          variant="error"
          message="Prohibido el tag :latest por reproducibilidad académica y gobernanza institucional."
        ></field-message>
      } @else if (isInvalidFormat()) {
        <field-message 
          variant="warning"
          message="Formato OCI inválido: debe ser repositorio:tag (ej: python:3.12-slim)."
        ></field-message>
      } @else if (verificationState() === 'checking') {
        <field-message 
          variant="info"
          message="Verificando imagen en el nodo y registro Docker..."
        ></field-message>
      } @else if (verificationState() === 'verified' && !isStorageBlocked() && !isArchIncompatible()) {
        <field-message 
          variant="success"
          [message]="verificationSuccessMessage()"
          actionLabel="Re-verificar"
          (actionClicked)="reverify.emit()"
        ></field-message>
      } @else if (verificationState() === 'failed' || verificationState() === 'error') {
        <field-message 
          variant="error"
          message="No se pudo verificar la imagen en el nodo ni en el registro público."
          actionLabel="Reintentar verificación"
          (actionClicked)="reverify.emit()"
        ></field-message>
      } @else if (isStorageBlocked()) {
        <field-message 
          variant="error"
          message="El tamaño de la imagen excede la cuota de almacenamiento del nodo."
        ></field-message>
      } @else if (isArchIncompatible()) {
        <field-message 
          variant="error"
          message="La arquitectura de la imagen es incompatible con el host (requiere amd64)."
        ></field-message>
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

      @if (invalidToolSyntaxError()) {
        <div class="mt-2">
          <field-message 
            variant="error"
            [message]="invalidToolSyntaxError()!"
          ></field-message>
        </div>
      } @else if (toolsAffinityWarning()) {
        <div class="mt-2">
          <field-message 
            variant="warning"
            [message]="toolsAffinityWarning()!"
          ></field-message>
        </div>
      }
      
      <div class="tools-suggested-chips mt-2">
        <span class="chips-label" i18n="@@AY-15">
          {{ hasImageFamilyMatch() ? 'Sugerencias según la imagen elegida:' : 'Sugerencias según el propósito:' }}
        </span>
        <div class="chips-row">
          @for (tool of reactiveSuggestedTools(); track $index) {
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
export class StepImageComponent {
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
    'go:': ['go', 'gofmt'],
    'gcc': ['gcc', 'g++', 'make'],
    'eclipse-temurin': ['javac', 'java', 'jar'],
    'openjdk': ['javac', 'java', 'jar'],
    'rust': ['rustc', 'cargo'],
    'postgres': ['psql', 'pg_dump'],
    'redis': ['redis-cli', 'redis-server']
  };

  invalidToolSyntaxError = computed<string | null>(() => {
    const raw = this.toolsDeclared().trim();
    if (!raw) return null;
    const tools = raw.split(',').map(t => t.trim()).filter(Boolean);
    const validRegex = /^[a-zA-Z0-9_.-]+$/;

    for (const tool of tools) {
      if (!validRegex.test(tool)) {
        return `Error: La herramienta "${tool}" contiene caracteres no válidos. Usá solo letras, números, guiones y punto.`;
      }
    }
    return null;
  });

  toolsAffinityWarning = computed<string | null>(() => {
    const img = this.dockerImage().trim().toLowerCase();
    if (!img) return null;

    const currentEco = TOOL_ECOSYSTEMS.find(eco =>
      eco.imageKeywords.some(kw => img.includes(kw))
    );
    if (!currentEco) return null;

    const declared = this.toolsDeclared()
      .split(',')
      .map(t => t.trim().toLowerCase())
      .filter(Boolean);

    if (declared.length === 0) return null;

    const mismatches: { tool: string; otherEco: string }[] = [];

    for (const tool of declared) {
      if (UNIVERSAL_TOOLS.has(tool)) continue;
      if (currentEco.tools.includes(tool)) continue;

      const otherEco = TOOL_ECOSYSTEMS.find(eco => eco.name !== currentEco.name && eco.tools.includes(tool));
      if (otherEco) {
        mismatches.push({ tool, otherEco: otherEco.name });
      }
    }

    if (mismatches.length === 0) return null;

    if (mismatches.length === 1) {
      return `Aviso: La herramienta "${mismatches[0].tool}" suele pertenecer al entorno ${mismatches[0].otherEco}, inusual en imágenes base de ${currentEco.name}. Si la imagen no la incluye, fallará en el paso de verificación.`;
    }

    const list = mismatches.map(m => `"${m.tool}" (${m.otherEco})`).join(', ');
    return `Aviso: Las herramientas [${list}] pertenecen a otros entornos. Si la imagen no las incluye, fallará en el paso de verificación.`;
  });

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

export { StepImageComponent as SolvStepImageComponent };
