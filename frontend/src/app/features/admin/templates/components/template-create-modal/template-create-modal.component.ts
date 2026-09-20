import { 
  Component, 
  EventEmitter, 
  Output, 
  signal, 
  computed, 
  HostListener, 
  inject, 
  OnInit, 
  OnDestroy 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject, Subscription } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { 
  CreateOfficialTemplateDTO, 
  AdminTemplatesService, 
  AvailableSatelliteService, 
  ServiceRequirement,
  LocalImageItem,
  ImageVerificationResult
} from '../../../services/admin-templates.service';
import { 
  LucideX, 
  LucideLayers, 
  LucideHardDrive, 
  LucidePlusCircle,
  LucideAlertTriangle,
  LucideDatabase,
  LucideServer,
  LucideCheck,
  LucideTerminal,
  LucideCpu,
  LucideInfo,
  LucideRefreshCw,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideCopy,
  LucideSearch
} from '@lucide/angular';

interface ImageSuggestion {
  repoTag: string;
  isLocal: boolean;
  sizeMB?: number;
  description?: string;
  isOfficial: boolean;
}

@Component({
  selector: 'solv-template-create-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideX, 
    LucideLayers, 
    LucideHardDrive, 
    LucidePlusCircle,
    LucideAlertTriangle,
    LucideDatabase,
    LucideServer,
    LucideCheck,
    LucideTerminal,
    LucideCpu,
    LucideInfo,
    LucideRefreshCw,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideCopy,
    LucideSearch
  ],
  templateUrl: './template-create-modal.component.html',
  styleUrls: ['./template-create-modal.component.scss']
})
export class TemplateCreateModalComponent implements OnInit, OnDestroy {
  private templatesService = inject(AdminTemplatesService);

  @Output() created = new EventEmitter<CreateOfficialTemplateDTO>();
  @Output() closed = new EventEmitter<void>();

  name = signal<string>('');
  dockerImage = signal<string>('');
  baseRamMB = signal<number>(512);
  setupScript = signal<string>('');
  description = signal<string>('');
  isSubmitting = signal<boolean>(false);

  // Verificación y Caché de Imagen
  verificationState = signal<'idle' | 'checking' | 'verified' | 'error'>('idle');
  verificationResult = signal<ImageVerificationResult | null>(null);
  verificationError = signal<string | null>(null);

  // Typeahead y Sugerencias de Imágenes
  localImages = signal<LocalImageItem[]>([]);
  isDropdownOpen = signal<boolean>(false);
  copiedCommand = signal<boolean>(false);

  // Regex estricto de imagen Docker OCI (sin espacios, repo y tag obligatorio)
  private dockerRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$/;

  private imageDebounce$ = new Subject<string>();
  private sub = new Subscription();

  // Imágenes oficiales base comúnmente usadas en docencia
  private curatedOfficialImages: ImageSuggestion[] = [
    { repoTag: 'python:3.12-slim-bookworm', isLocal: false, isOfficial: true, description: 'Python 3.12 oficial ligero Debian' },
    { repoTag: 'python:3.11-slim', isLocal: false, isOfficial: true, description: 'Python 3.11 versión estable' },
    { repoTag: 'node:20-bookworm-slim', isLocal: false, isOfficial: true, description: 'Node.js LTS 20 Debian Bookworm' },
    { repoTag: 'node:20-alpine', isLocal: false, isOfficial: true, description: 'Node.js LTS 20 ultraligero Alpine' },
    { repoTag: 'gcc:13.2-bookworm', isLocal: false, isOfficial: true, description: 'Entorno de compilación C/C++ GCC' },
    { repoTag: 'golang:1.22-bookworm', isLocal: false, isOfficial: true, description: 'Go SDK 1.22 oficial' },
    { repoTag: 'eclipse-temurin:21-alpine', isLocal: false, isOfficial: true, description: 'Java OpenJDK 21 LTS Alpine' },
    { repoTag: 'postgres:16-alpine', isLocal: false, isOfficial: true, description: 'PostgreSQL 16 base ligera' }
  ];

  availableServices = signal<AvailableSatelliteService[]>(
    this.templatesService.getAvailableSatelliteServices()
  );
  selectedServices = signal<ServiceRequirement[]>([]);

  ngOnInit(): void {
    this.fetchLocalImages();

    // Debounce reactivo para validación automática al dejar de escribir
    this.sub.add(
      this.imageDebounce$.pipe(
        debounceTime(600),
        distinctUntilChanged()
      ).subscribe(image => {
        if (this.canVerify(image)) {
          this.triggerVerification(image, false);
        } else {
          this.verificationState.set('idle');
          this.verificationResult.set(null);
          this.verificationError.set(null);
        }
      })
    );
  }

  ngOnDestroy(): void {
    this.sub.unsubscribe();
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.isDropdownOpen()) {
      this.isDropdownOpen.set(false);
      return;
    }
    this.closeModal();
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.typeahead-container')) {
      this.isDropdownOpen.set(false);
    }
  }

  fetchLocalImages(): void {
    this.templatesService.getLocalImages().subscribe({
      next: (images) => {
        this.localImages.set(images);
      },
      error: () => {
        // Silencioso: si no responde Docker local, operamos con presets y remoto
      }
    });
  }

  onImageInputChange(value: string): void {
    this.dockerImage.set(value);
    this.isDropdownOpen.set(true);
    this.imageDebounce$.next(value.trim());
  }

  // Lista combinada de sugerencias (dando prioridad absoluta a las residentes locales)
  imageSuggestions = computed<ImageSuggestion[]>(() => {
    const query = this.dockerImage().trim().toLowerCase();
    const local = this.localImages();
    const suggestions: ImageSuggestion[] = [];
    const seen = new Set<string>();

    // 1. Imágenes residentes en el servidor local (sin tag :latest para forzar inmutabilidad)
    for (const img of local) {
      if (!img.has_latest_tag && !seen.has(img.repo_tag)) {
        if (!query || img.repo_tag.toLowerCase().includes(query)) {
          suggestions.push({
            repoTag: img.repo_tag,
            isLocal: true,
            sizeMB: img.size_mb,
            isOfficial: img.is_official,
            description: 'En el servidor (despliegue inmediato)'
          });
          seen.add(img.repo_tag);
        }
      }
    }

    // 2. Curated presets si no colisionan
    for (const cur of this.curatedOfficialImages) {
      if (!seen.has(cur.repoTag)) {
        if (!query || cur.repoTag.toLowerCase().includes(query)) {
          // Comprobar si casualmente está descargada
          const isActuallyLocal = local.some(l => l.repo_tag === cur.repoTag);
          suggestions.push({
            ...cur,
            isLocal: isActuallyLocal
          });
          seen.add(cur.repoTag);
        }
      }
    }

    return suggestions.slice(0, 8);
  });

  selectSuggestion(suggestion: ImageSuggestion): void {
    this.dockerImage.set(suggestion.repoTag);
    this.isDropdownOpen.set(false);
    this.triggerVerification(suggestion.repoTag, false);
  }

  isLatestImage = computed(() => {
    const img = this.dockerImage().trim().toLowerCase();
    if (!img) return false;
    return img.endsWith(':latest');
  });

  isInvalidFormat = computed(() => {
    const img = this.dockerImage().trim();
    if (!img) return false;
    return !this.dockerRegex.test(img);
  });

  isRamTooLow = computed(() => {
    return this.baseRamMB() < 256;
  });

  private canVerify(image: string): boolean {
    const trimmed = image.trim();
    return trimmed.length > 0 && this.dockerRegex.test(trimmed) && !trimmed.toLowerCase().endsWith(':latest');
  }

  triggerVerification(imageToVerify?: string, force = false): void {
    const img = (imageToVerify || this.dockerImage()).trim();
    if (!this.canVerify(img)) return;

    this.verificationState.set('checking');
    this.verificationError.set(null);

    this.templatesService.verifyImage(img, force).subscribe({
      next: (result) => {
        this.verificationResult.set(result);
        if (result.exists) {
          this.verificationState.set('verified');
        } else {
          this.verificationState.set('error');
          this.verificationError.set(result.error_message || 'Imagen no encontrada en el registro OCI');
        }
      },
      error: (err) => {
        this.verificationState.set('error');
        const msg = err?.error?.message || err?.message || 'Error de conexión con el servicio de verificación';
        this.verificationError.set(msg);
      }
    });
  }

  reverify(): void {
    this.triggerVerification(this.dockerImage(), true);
  }

  copyBuildxCommand(cmd: string): void {
    if (!cmd) return;
    navigator.clipboard.writeText(cmd).then(() => {
      this.copiedCommand.set(true);
      setTimeout(() => this.copiedCommand.set(false), 2000);
    });
  }

  resourceProfilePreview = computed(() => {
    const ram = this.baseRamMB() || 256;
    const editorBase = 210;
    const usable = Math.max(0, ram - editorBase);
    const minReservation = Math.max(128, Math.floor(ram * 0.5));
    const highWarning = Math.floor(ram * 0.8);
    const estimatedCapacity = Math.floor((32 * 1024 * 0.85) / ram);
    return {
      ram,
      editorBase,
      usable,
      minReservation,
      highWarning,
      estimatedCapacity
    };
  });

  isStorageBlocked = computed(() => {
    const res = this.verificationResult();
    return res?.storage_status === 'CRITICAL_BLOCKED';
  });

  isArchIncompatible = computed(() => {
    const res = this.verificationResult();
    return res !== null && res.exists && !res.architecture_compatible;
  });

  isValid = computed(() => {
    const hasName = this.name().trim().length > 0;
    const hasImage = this.dockerImage().trim().length > 0;
    const formatValid = !this.isInvalidFormat() && !this.isLatestImage();
    const ramValid = !this.isRamTooLow();
    const notBlocked = !this.isStorageBlocked() && !this.isArchIncompatible();
    const notChecking = this.verificationState() !== 'checking';

    return hasName && hasImage && formatValid && ramValid && notBlocked && notChecking;
  });

  setRam(mb: number): void {
    this.baseRamMB.set(mb);
  }

  onRamInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    const parsed = parseInt(input.value, 10);
    if (!isNaN(parsed) && parsed > 0) {
      this.baseRamMB.set(parsed);
    }
  }

  toggleService(service: AvailableSatelliteService): void {
    const current = this.selectedServices();
    const exists = current.some(s => s.engine === service.engine);

    if (exists) {
      this.selectedServices.set(current.filter(s => s.engine !== service.engine));
    } else {
      let filtered = current;
      if (service.engine === 'postgres') {
        filtered = filtered.filter(s => s.engine !== 'mysql');
      } else if (service.engine === 'mysql') {
        filtered = filtered.filter(s => s.engine !== 'postgres');
      }

      this.selectedServices.set([
        ...filtered,
        {
          category: service.category,
          engine: service.engine,
          version: service.version
        }
      ]);
    }
  }

  isServiceSelected(engine: string): boolean {
    return this.selectedServices().some(s => s.engine === engine);
  }

  confirmCreate(): void {
    if (!this.isValid() || this.isSubmitting()) return;
    this.isSubmitting.set(true);
    this.created.emit({
      name: this.name().trim(),
      docker_image: this.dockerImage().trim(),
      base_ram_mb: this.baseRamMB(),
      setup_script: this.setupScript().trim(),
      description: this.description().trim(),
      services_config: {
        services: this.selectedServices()
      }
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    this.closed.emit();
  }
}
