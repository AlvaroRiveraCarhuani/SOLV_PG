import { Component, ElementRef, ViewChild, computed, effect, inject, input, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Subscription } from 'rxjs';
import {
  LucideAlertTriangle,
  LucideCheck,
  LucideFileArchive,
  LucideFileCode,
  LucideFolder,
  LucideGlobe,
  LucideHistory,
  LucideLoader2,
  LucidePencil,
  LucidePlus,
  LucideSave,
  LucideSparkles,
  LucideTrash2,
  LucideUploadCloud,
} from '@lucide/angular';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
import { ReviewFileTreeComponent } from '@shared/components/review-file-tree/review-file-tree.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ProjectFile } from '@core/models/project-file.model';
import { ExerciseEditorStore } from '../exercise-editor.store';
import {
  WorkspaceGenerationStackConfigDTO,
  WorkspaceProjectGenerationRequestDTO,
  WorkspaceProjectProposalDTO,
  projectErrorMessage,
  TeacherProjectService,
  WorkspaceProjectVersionDTO,
  WorkspaceProjectViewDTO,
} from '../../services/teacher-project.service';
import type { ApprovedTemplate } from '../steps/step-ide-environment/step-ide-environment.component';
import {
  WorkspaceProjectFile,
  WORKSPACE_PROJECT_MAX_FILES,
  WORKSPACE_PROJECT_MAX_ZIP_BYTES,
  canonicalizeProjectContent,
  encodeWorkspaceSnapshot,
  languageForProjectPath,
  normalizeWorkspacePath,
  parseWorkspaceSnapshot,
  summarizeProjectChanges,
  validateProjectFiles,
} from './workspace-project.utils';
import {
  detectEnvironmentStack,
  detectProjectStack,
  findStackMismatch,
} from './workspace-project-stack';
import {
  WorkspaceDropEntryLike,
  WorkspaceImportSelection,
  canPickDirectory,
  pickDirectory,
  readDroppedEntries,
  readSelectedFiles,
} from './workspace-project-import';

@Component({
  selector: 'ide-project-initial',
  standalone: true,
  imports: [
    DateTextPipe,
    MachineDataDirective,
    CodeEditorComponent,
    FieldMessageComponent,
    ReviewFileTreeComponent,
    ModalShellComponent,
    LucideAlertTriangle,
    LucideCheck,
    LucideFileArchive,
    LucideFileCode,
    LucideFolder,
    LucideGlobe,
    LucideHistory,
    LucideLoader2,
    LucidePencil,
    LucidePlus,
    LucideSave,
    LucideSparkles,
    LucideTrash2,
    LucideUploadCloud,
  ],
  templateUrl: './ide-project-initial.component.html',
  styleUrl: './ide-project-initial.component.scss',
})
export class IdeProjectInitialComponent {
  readonly store = inject(ExerciseEditorStore);
  private readonly service = inject(TeacherProjectService);
  readonly environments = input<ApprovedTemplate[]>([]);
  readonly maxFiles = WORKSPACE_PROJECT_MAX_FILES;
  readonly languageForProjectPath = languageForProjectPath;

  @ViewChild('folderInput') private folderInput?: ElementRef<HTMLInputElement>;
  @ViewChild('zipInput') private zipInput?: ElementRef<HTMLInputElement>;

  readonly view = signal<WorkspaceProjectViewDTO | null>(null);
  readonly files = signal<WorkspaceProjectFile[]>([]);
  readonly selectedPath = signal<string | null>(null);
  readonly savedSnapshot = signal('');
  readonly revision = signal(0);
  readonly loading = signal(false);
  readonly saving = signal(false);
  readonly importing = signal(false);
  readonly publishing = signal(false);
  readonly errorMessage = signal<string | null>(null);
  readonly notice = signal<string | null>(null);
  readonly conflict = signal(false);
  readonly publishModal = signal(false);
  readonly publishError = signal<string | null>(null);
  readonly versionPreview = signal<WorkspaceProjectVersionDTO | null>(null);
  readonly addFileOpen = signal(false);
  readonly newFilePath = signal('');
  readonly renameOpen = signal(false);
  readonly renamePath = signal('');
  readonly deleteConfirm = signal(false);
  readonly dropActive = signal(false);
  readonly pendingImport = signal<WorkspaceImportSelection | null>(null);
  readonly aiOpen = signal(false);
  readonly aiGenerating = signal(false);
  readonly aiError = signal<string | null>(null);
  readonly aiProposal = signal<WorkspaceProjectProposalDTO | null>(null);
  readonly aiPreviewPath = signal<string | null>(null);
  readonly aiForm = signal<
    Omit<WorkspaceProjectGenerationRequestDTO, 'technology' | 'template_id' | 'stack_config'>
  >({
    description: '',
    purpose: '',
    requirements: '',
  });
  readonly pendingProposal = signal<WorkspaceProjectProposalDTO | null>(null);
  readonly retrying = signal(false);
  readonly stackWarning = computed(() => this.stackWarningFor(this.files()));
  readonly aiStackWarning = computed(() => {
    const proposal = this.aiProposal();
    return proposal
      ? this.stackWarningFor(
          proposal.files.map((file) => ({ path: file.path, content: file.content })),
        )
      : null;
  });
  readonly aiEnvironmentName = computed(() => this.store.metadata().template?.name ?? '');
  readonly aiTechnology = computed(() => {
    const template = this.store.metadata().template;
    if (!template) return '';
    const labels = detectEnvironmentStack(template).techs.map(tech => tech.label);
    return labels.length > 0 ? `${template.name} (${labels.join(', ')})` : template.name;
  });
  readonly aiStackConfig = computed<WorkspaceGenerationStackConfigDTO | null>(() => {
    const raw = this.store.metadata().template?.stack_config;
    if (!raw || typeof raw !== 'object') return null;
    const config = raw as Record<string, unknown>;
    const result: WorkspaceGenerationStackConfigDTO = {};
    if (typeof config['runtime_id'] === 'string') result.runtime_id = config['runtime_id'];
    if (typeof config['web_platform_id'] === 'string') result.web_platform_id = config['web_platform_id'];
    if (Array.isArray(config['addon_ids'])) {
      result.addon_ids = config['addon_ids'].filter((id): id is string => typeof id === 'string');
    }
    if (typeof config['database_id'] === 'string') result.database_id = config['database_id'];
    return Object.keys(result).length > 0 ? result : null;
  });
  readonly stackAcknowledged = signal(false);
  readonly pendingImportWarning = signal<string | null>(null);
  readonly compatibleEnvironments = computed(() => {
    const currentId = this.store.metadata().template?.id;
    const project = detectProjectStack(this.files());
    if (project.techs.length === 0) return [];
    return this.environments()
      .filter(environment => environment.id !== currentId)
      .filter(environment => {
        const families = new Set(
          detectEnvironmentStack(environment).techs.map(tech => tech.family),
        );
        return project.techs.every(tech => families.has(tech.family));
      })
      .slice(0, 3);
  });
  readonly preparation = computed(() => this.view()?.preparation ?? null);
  readonly preparationLabel = computed(() => {
    switch (this.preparation()?.status) {
      case 'prepared':
        return 'Dependencias preparadas';
      case 'pending':
        return 'Preparación en cola';
      case 'running':
        return 'Preparando dependencias';
      case 'failed':
        return 'La preparación falló';
      case 'skipped':
        return 'Esta versión no prepara dependencias';
      default:
        return 'Preparación de dependencias';
    }
  });

  private loadedExerciseId: string | null = null;
  private aiRequest?: Subscription;

  readonly exerciseId = computed(() => this.store.exerciseId());
  readonly project = computed(() => this.view()?.project ?? null);
  readonly draft = computed(() => this.view()?.draft ?? null);
  readonly versions = computed(() => this.view()?.versions ?? []);
  readonly publishingEnabled = computed(() => !!this.view()?.publishing_enabled);
  readonly publishedVersion = computed(() => {
    const project = this.project();
    if (!project?.current_published_version_id) return null;
    return (
      this.versions().find((version) => version.id === project.current_published_version_id) ?? null
    );
  });
  readonly currentSnapshot = computed(() => encodeWorkspaceSnapshot(this.files()).text ?? '');
  readonly snapshotError = computed(() => encodeWorkspaceSnapshot(this.files()).error ?? null);
  readonly dirty = computed(
    () =>
      this.files().length > 0 &&
      (this.snapshotError() !== null || this.currentSnapshot() !== this.savedSnapshot()),
  );
  readonly canSave = computed(
    () => this.dirty() && !this.saving() && this.snapshotError() === null,
  );
  readonly hasPublishableChanges = computed(() => {
    const version = this.publishedVersion();
    return !version || this.currentSnapshot() !== version.text_snapshot;
  });
  readonly canPublish = computed(
    () =>
      this.publishingEnabled() &&
      !!this.draft() &&
      !this.dirty() &&
      !this.saving() &&
      this.hasPublishableChanges(),
  );
  readonly selectedFile = computed(
    () => this.files().find((file) => file.path === this.selectedPath()) ?? null,
  );
  readonly totalBytes = computed(() =>
    this.files().reduce((total, file) => total + new TextEncoder().encode(file.content).length, 0),
  );
  readonly fileTree = computed<ProjectFile[]>(() =>
    this.files().map((file) => ({
      path: file.path,
      language: languageForProjectPath(file.path),
      content: file.content,
      start_line: 0,
      line_count: file.content === '' ? 0 : file.content.split('\n').length,
    })),
  );
  readonly publishedFiles = computed(() => {
    const version = this.publishedVersion();
    if (!version) return [];
    const parsed = parseWorkspaceSnapshot(version.text_snapshot);
    return parsed.files ?? [];
  });
  readonly changeSummary = computed(() =>
    summarizeProjectChanges(this.publishedFiles(), this.files()),
  );
  readonly changeCount = computed(() => {
    const changes = this.changeSummary();
    return changes.added.length + changes.modified.length + changes.removed.length;
  });
  readonly nextVersionNumber = computed(
    () => Math.max(0, ...this.versions().map((version) => version.version_number)) + 1,
  );
  readonly aiPreviewFile = computed(() => {
    const proposal = this.aiProposal();
    if (!proposal) return null;
    return (
      proposal.files.find((file) => file.path === this.aiPreviewPath()) ?? proposal.files[0] ?? null
    );
  });

  constructor() {
    effect(() => {
      const id = this.store.exerciseId();
      if (!id) {
        this.loadedExerciseId = null;
        this.resetState();
        return;
      }
      if (this.loadedExerciseId === id) return;
      this.loadedExerciseId = id;
      this.load(id);
    });
  }

  openFolderPicker(input: HTMLInputElement): void {
    if (!this.exerciseId() || this.importing()) return;
    if (!canPickDirectory()) {
      input.click();
      return;
    }
    this.importing.set(true);
    this.errorMessage.set(null);
    pickDirectory()
      .then((files) => {
        this.importing.set(false);
        this.importFolderFiles(files);
      })
      .catch((error: unknown) => {
        this.importing.set(false);
        if (error instanceof DOMException && error.name === 'AbortError') return;
        this.errorMessage.set(
          projectErrorMessage(error, 'No se pudo leer la carpeta seleccionada.'),
        );
      });
  }

  onTextInput(event: Event, target: 'new' | 'rename'): void {
    const value = (event.target as HTMLInputElement).value;
    if (target === 'new') this.newFilePath.set(value);
    else this.renamePath.set(value);
  }

  async onFolderInput(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    if (!input.files?.length) return;
    try {
      const files = await readSelectedFiles(input.files);
      this.importFolderFiles(files);
    } catch (error) {
      this.errorMessage.set(
        projectErrorMessage(error, 'No se pudo leer uno de los archivos de la carpeta.'),
      );
    } finally {
      input.value = '';
    }
  }

  onZipInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (file) this.prepareZip(file);
  }

  importFolderFiles(files: WorkspaceProjectFile[]): void {
    const error = validateProjectFiles(files);
    if (error) {
      this.errorMessage.set(error);
      return;
    }
    const normalized = files.map((file) => ({ path: file.path.normalize('NFC'), content: file.content }));
    const warning = this.stackWarningFor(normalized);
    this.errorMessage.set(null);
    this.notice.set(null);
    if (warning) {
      this.pendingImportWarning.set(warning);
      this.pendingImport.set({ kind: 'folder', files: normalized });
      return;
    }
    this.prepareImport({ kind: 'folder', files: normalized });
  }

  prepareZip(file: File): void {
    if (!file.name.toLowerCase().endsWith('.zip')) {
      this.errorMessage.set('Selecciona un archivo con extensión .zip.');
      return;
    }
    if (file.size > WORKSPACE_PROJECT_MAX_ZIP_BYTES) {
      this.errorMessage.set(
        `El ZIP supera el máximo de ${formatBytes(WORKSPACE_PROJECT_MAX_ZIP_BYTES)}.`,
      );
      return;
    }
    this.prepareImport({ kind: 'zip', file });
  }

  prepareImport(selection: WorkspaceImportSelection): void {
    this.errorMessage.set(null);
    this.notice.set(null);
    this.pendingImportWarning.set(null);
    if (this.dirty() || this.conflict()) {
      this.pendingImport.set(selection);
      return;
    }
    this.executeImport(selection);
  }

  confirmReplaceImport(): void {
    const selection = this.pendingImport();
    this.pendingImport.set(null);
    this.pendingImportWarning.set(null);
    if (selection) this.executeImport(selection);
  }

  cancelReplaceImport(): void {
    this.pendingImport.set(null);
    this.pendingImportWarning.set(null);
  }

  openAiGenerator(): void {
    if (!this.exerciseId() || this.aiGenerating()) return;
    this.aiError.set(null);
    this.aiOpen.set(true);
  }

  closeAiGenerator(): void {
    if (this.aiGenerating()) return;
    this.aiOpen.set(false);
  }

  onAiFieldInput(
    event: Event,
    field: keyof Omit<WorkspaceProjectGenerationRequestDTO, 'technology' | 'template_id' | 'stack_config'>,
  ): void {
    const value = (event.target as HTMLInputElement | HTMLTextAreaElement).value;
    this.aiForm.update((form) => ({ ...form, [field]: value }));
  }

  selectAiFile(path: string): void {
    this.aiPreviewPath.set(path);
  }

  generateProposal(): void {
    const id = this.exerciseId();
    if (!id || this.aiGenerating()) return;
    const template = this.store.metadata().template;
    if (!template) {
      this.aiError.set('Elegí un entorno de trabajo antes de generar el proyecto.');
      return;
    }
    const form = this.aiForm();
    if (!form.description.trim()) {
      this.aiError.set('Describe qué proyecto querés preparar antes de generar.');
      return;
    }

    this.aiGenerating.set(true);
    this.aiError.set(null);
    this.aiProposal.set(null);
    this.aiRequest = this.service
      .generateProject(id, {
        description: form.description.trim(),
        purpose: form.purpose?.trim() || undefined,
        technology: this.aiTechnology(),
        template_id: template.id,
        stack_config: this.aiStackConfig(),
        requirements: form.requirements?.trim() || undefined,
      })
      .subscribe({
        next: (proposal) => {
          this.aiGenerating.set(false);
          this.aiProposal.set(proposal);
          this.aiPreviewPath.set(proposal.files[0]?.path ?? null);
        },
        error: (error) => {
          this.aiGenerating.set(false);
          this.aiError.set(
            projectErrorMessage(
              error,
              'No se pudo generar la propuesta. El proyecto no se modificó.',
            ),
          );
        },
      });
  }

  cancelAiGeneration(): void {
    if (!this.aiGenerating()) return;
    this.aiRequest?.unsubscribe();
    this.aiGenerating.set(false);
    this.notice.set('Generación cancelada. El borrador no cambió.');
  }

  discardProposal(): void {
    this.aiProposal.set(null);
    this.aiPreviewPath.set(null);
    this.aiError.set(null);
  }

  requestApplyProposal(): void {
    const proposal = this.aiProposal();
    if (!proposal) return;
    if (this.conflict()) {
      // El borrador del servidor es más nuevo que el de esta pantalla: aplicar
      // la propuesta acá perdería lo que guardó la otra persona.
      this.aiError.set(
        'Otra persona guardó una revisión más nueva. Recargá el borrador antes de aplicar la propuesta.',
      );
      return;
    }
    if (this.dirty()) {
      this.pendingProposal.set(proposal);
      return;
    }
    this.applyProposal(proposal);
  }

  confirmReplaceProposal(): void {
    const proposal = this.pendingProposal();
    this.pendingProposal.set(null);
    if (proposal) this.applyProposal(proposal);
  }

  cancelReplaceProposal(): void {
    this.pendingProposal.set(null);
  }

  // Incorpora los archivos al editor sin guardar ni publicar: el docente revisa
  // y usa el guardado optimista existente cuando decide conservarlos.
  private applyProposal(proposal: WorkspaceProjectProposalDTO): void {
    const files = proposal.files.map((file) => ({
      path: file.path,
      content: canonicalizeProjectContent(file.content),
    }));
    this.files.set(files);
    this.selectedPath.set(files[0]?.path ?? null);
    this.aiProposal.set(null);
    this.aiPreviewPath.set(null);
    this.aiError.set(null);
    this.aiOpen.set(false);
    this.errorMessage.set(null);
    this.notice.set(
      'Propuesta aplicada al editor. Revisá el contenido y guardá el borrador para conservarla.',
    );
  }

  private executeImport(selection: WorkspaceImportSelection): void {
    const id = this.exerciseId();
    if (!id || this.importing()) return;
    const form = new FormData();
    form.append('expected_revision', String(this.revision()));
    this.importing.set(true);
    this.errorMessage.set(null);
    this.conflict.set(false);

    if (selection.kind === 'folder') {
      const files = selection.files.map((file) => ({
        path: file.path,
        content: canonicalizeProjectContent(file.content),
      }));
      form.append(
        'manifest',
        JSON.stringify({
          files: files.map((file, index) => ({
            path: file.path,
            field: `file_${index}`,
            size: new TextEncoder().encode(file.content).length,
          })),
        }),
      );
      files.forEach((file, index) => {
        form.append(
          `file_${index}`,
          new Blob([file.content], { type: 'text/plain' }),
          basename(file.path),
        );
      });
      this.service.importFolder(id, form).subscribe({
        next: (result) => this.onImported(`${result.file_count} archivos importados al borrador.`),
        error: (error) => this.onImportError(error),
      });
      return;
    }

    form.append('archive', selection.file, selection.file.name);
    this.service.importZip(id, form).subscribe({
      next: (result) => this.onImported(`${result.file_count} archivos importados desde el ZIP.`),
      error: (error) => this.onImportError(error),
    });
  }

  private onImported(message: string): void {
    this.importing.set(false);
    this.notice.set(message);
    this.conflict.set(false);
    const id = this.exerciseId();
    if (id) this.load(id);
  }

  private onImportError(error: unknown): void {
    this.importing.set(false);
    if (error instanceof HttpErrorResponse && error.status === 409) this.conflict.set(true);
    this.errorMessage.set(projectErrorMessage(error, 'No se pudo importar el proyecto.'));
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.dropActive.set(false);
    if (!this.exerciseId() || this.importing()) return;
    const dataTransfer = event.dataTransfer;
    if (!dataTransfer) return;
    const entries: WorkspaceDropEntryLike[] = [];
    for (const item of Array.from(dataTransfer.items)) {
      const entry = (
        item as DataTransferItem & { webkitGetAsEntry?: () => WorkspaceDropEntryLike | null }
      ).webkitGetAsEntry?.();
      if (entry) entries.push(entry);
    }

    if (entries.length > 0) {
      readDroppedEntries(entries)
        .then((selection) =>
          selection.kind === 'zip'
            ? this.prepareZip(selection.file)
            : this.importFolderFiles(selection.files),
        )
        .catch((error: unknown) =>
          this.errorMessage.set(projectErrorMessage(error, 'No se pudo leer lo que arrastraste.')),
        );
      return;
    }
    const dropped = dataTransfer.files[0];
    if (dropped) this.prepareZip(dropped);
    else this.errorMessage.set('Arrastra una carpeta o un único archivo ZIP.');
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    if (!this.importing()) this.dropActive.set(true);
  }

  onDragLeave(event: DragEvent): void {
    if (event.currentTarget === event.target) this.dropActive.set(false);
  }

  selectFile(path: string): void {
    this.selectedPath.set(path);
    this.addFileOpen.set(false);
    this.renameOpen.set(false);
    this.deleteConfirm.set(false);
  }

  onFileContentChange(content: string): void {
    const path = this.selectedPath();
    if (!path) return;
    this.files.update((files) =>
      files.map((file) => (file.path === path ? { ...file, content } : file)),
    );
  }

  saveDraft(): void {
    const id = this.exerciseId();
    const encoded = encodeWorkspaceSnapshot(this.files());
    if (!id || !encoded.text) {
      this.errorMessage.set(encoded.error ?? 'No hay un borrador para guardar.');
      return;
    }
    this.saving.set(true);
    this.errorMessage.set(null);
    this.conflict.set(false);
    this.service.saveDraft(id, encoded.text, this.revision()).subscribe({
      next: (draft) => {
        this.saving.set(false);
        this.revision.set(draft.revision);
        this.savedSnapshot.set(encoded.text as string);
        this.view.update((view) => (view ? { ...view, draft } : view));
        this.notice.set('Borrador guardado.');
      },
      error: (error) => {
        this.saving.set(false);
        if (error instanceof HttpErrorResponse && error.status === 409) this.conflict.set(true);
        this.errorMessage.set(projectErrorMessage(error, 'No se pudo guardar el borrador.'));
      },
    });
  }

  reloadAfterConflict(): void {
    const id = this.exerciseId();
    if (id) this.load(id);
  }

  openAddFile(): void {
    this.newFilePath.set('');
    this.addFileOpen.set(true);
    this.renameOpen.set(false);
    this.deleteConfirm.set(false);
  }

  addFile(): void {
    const normalized = normalizeWorkspacePath(this.newFilePath());
    if (normalized.error) {
      this.errorMessage.set(normalized.error);
      return;
    }
    const path = normalized.path as string;
    if (this.files().some((file) => file.path === path)) {
      this.errorMessage.set(`La ruta ${path} ya existe.`);
      return;
    }
    this.files.update((files) => [...files, { path, content: '' }]);
    this.selectedPath.set(path);
    this.addFileOpen.set(false);
    this.newFilePath.set('');
    this.errorMessage.set(null);
  }

  startRename(): void {
    const selected = this.selectedPath();
    if (!selected) return;
    this.renamePath.set(selected);
    this.renameOpen.set(true);
    this.deleteConfirm.set(false);
    this.addFileOpen.set(false);
  }

  renameFile(): void {
    const previous = this.selectedPath();
    if (!previous) return;
    const normalized = normalizeWorkspacePath(this.renamePath());
    if (normalized.error) {
      this.errorMessage.set(normalized.error);
      return;
    }
    const next = normalized.path as string;
    if (next === previous) {
      this.renameOpen.set(false);
      return;
    }
    if (this.files().some((file) => file.path === next)) {
      this.errorMessage.set(`La ruta ${next} ya existe.`);
      return;
    }
    this.files.update((files) =>
      files.map((file) => (file.path === previous ? { ...file, path: next } : file)),
    );
    this.selectedPath.set(next);
    this.renameOpen.set(false);
    this.errorMessage.set(null);
  }

  confirmDeleteFile(): void {
    const selected = this.selectedPath();
    if (!selected) return;
    const remaining = this.files().filter((file) => file.path !== selected);
    this.files.set(remaining);
    this.selectedPath.set(remaining[0]?.path ?? null);
    this.deleteConfirm.set(false);
  }

  openPublish(): void {
    if (!this.canPublish()) return;
    this.publishError.set(null);
    this.stackAcknowledged.set(false);
    this.publishModal.set(true);
  }

  closePublish(): void {
    if (this.publishing()) return;
    this.publishModal.set(false);
    this.stackAcknowledged.set(false);
  }

  onStackAcknowledged(event: Event): void {
    this.stackAcknowledged.set((event.target as HTMLInputElement).checked);
  }

  focusEnvironmentSelector(): void {
    const section = document.getElementById('template-section');
    section?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    section?.querySelector<HTMLElement>('input')?.focus();
  }

  selectEnvironment(environment: ApprovedTemplate): void {
    this.store.updateMetadata({
      template_id: environment.id,
      template: environment,
      memory_limit_mb: environment.base_ram_mb,
    });
  }

  confirmPublish(): void {
    const id = this.exerciseId();
    if (!id || !this.canPublish()) return;
    if (this.stackWarning() && !this.stackAcknowledged()) return;
    this.publishing.set(true);
    this.publishError.set(null);
    this.service.publish(id).subscribe({
      next: (version) => {
        this.publishing.set(false);
        this.publishModal.set(false);
        this.notice.set(`Versión ${version.version_number} publicada.`);
        this.load(id);
      },
      error: (error) => {
        this.publishing.set(false);
        this.publishError.set(projectErrorMessage(error, 'No se pudo publicar el proyecto.'));
      },
    });
  }

  toggleVersionPreview(version: WorkspaceProjectVersionDTO): void {
    this.versionPreview.set(this.versionPreview()?.id === version.id ? null : version);
  }

  closeVersionPreview(): void {
    this.versionPreview.set(null);
  }

  // Vuelve a encolar la preparación de la versión vigente. El servidor
  // recalcula la clave con el perfil actual; acá solo se refleja el resultado.
  retryPreparation(): void {
    const id = this.exerciseId();
    const preparation = this.preparation();
    if (!id || !preparation?.can_retry || this.retrying()) return;
    this.retrying.set(true);
    this.errorMessage.set(null);
    this.service.retryPreparation(id).subscribe({
      next: () => {
        this.retrying.set(false);
        this.notice.set('Preparación encolada de nuevo.');
        this.load(id);
      },
      error: (error) => {
        this.retrying.set(false);
        this.errorMessage.set(
          projectErrorMessage(error, 'No se pudo reintentar la preparación.'),
        );
      },
    });
  }

  private stackWarningFor(files: WorkspaceProjectFile[]): string | null {
    const template = this.store.metadata().template;
    if (!template || files.length === 0) return null;
    const mismatch = findStackMismatch(
      detectProjectStack(files),
      detectEnvironmentStack(template),
      template.name,
    );
    if (!mismatch) return null;
    const labels = mismatch.unsupportedLabels;
    const list =
      labels.length === 1
        ? labels[0]
        : `${labels.slice(0, -1).join(', ')} y ${labels[labels.length - 1]}`;
    return `El proyecto incluye ${list}, pero el entorno «${mismatch.environmentName}» fue preparado para otra tecnología. Esas herramientas pueden no estar disponibles al abrirlo; revisá el proyecto o elegí otro entorno.`;
  }

  formatBytes(bytes: number): string {
    return formatBytes(bytes);
  }

  contentBytes(content: string): number {
    return new TextEncoder().encode(content).length;
  }

  private load(exerciseId: string): void {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.service.getProject(exerciseId).subscribe({
      next: (view) => {
        if (this.store.exerciseId() !== exerciseId) return;
        this.applyView(view);
        this.loading.set(false);
      },
      error: (error) => {
        this.loading.set(false);
        this.errorMessage.set(projectErrorMessage(error, 'No se pudo leer el proyecto inicial.'));
      },
    });
  }

  private applyView(view: WorkspaceProjectViewDTO): void {
    this.view.set(view);
    const published = view.versions.find(
      (version) => version.id === view.project?.current_published_version_id,
    );
    const snapshot =
      view.draft?.text_snapshot ?? published?.text_snapshot ?? view.legacy?.text_snapshot ?? '';
    const parsed = snapshot
      ? parseWorkspaceSnapshot(snapshot)
      : { files: [] as WorkspaceProjectFile[] };
    this.savedSnapshot.set(snapshot);
    this.revision.set(view.draft?.revision ?? 0);
    this.files.set(parsed.files ?? []);
    this.selectedPath.set(parsed.files?.[0]?.path ?? null);
    this.conflict.set(false);
    if (parsed.error) this.errorMessage.set(parsed.error);
  }

  private resetState(): void {
    this.aiRequest?.unsubscribe();
    this.view.set(null);
    this.files.set([]);
    this.selectedPath.set(null);
    this.savedSnapshot.set('');
    this.revision.set(0);
    this.loading.set(false);
    this.importing.set(false);
    this.saving.set(false);
    this.publishing.set(false);
    this.errorMessage.set(null);
    this.notice.set(null);
    this.conflict.set(false);
    this.publishModal.set(false);
    this.stackAcknowledged.set(false);
    this.versionPreview.set(null);
    this.pendingImport.set(null);
    this.pendingImportWarning.set(null);
    this.aiOpen.set(false);
    this.aiGenerating.set(false);
    this.aiError.set(null);
    this.aiProposal.set(null);
    this.aiPreviewPath.set(null);
    this.pendingProposal.set(null);
    this.retrying.set(false);
  }
}

function basename(path: string): string {
  const index = path.lastIndexOf('/');
  return index >= 0 ? path.slice(index + 1) : path;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}
