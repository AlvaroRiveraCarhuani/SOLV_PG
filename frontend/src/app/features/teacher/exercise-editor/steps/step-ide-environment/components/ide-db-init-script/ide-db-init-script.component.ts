import { Component, computed, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  LucideDatabase,
  LucideUploadCloud,
  LucideFileCode,
  LucideDownload,
  LucideTrash2,
  LucideAlertTriangle,
  LucideFileUp,
  LucideInfo
} from '@lucide/angular';

export const MAX_RECOMMENDED_SQL_BYTES = 2 * 1024 * 1024; // 2 MiB

@Component({
  selector: 'ide-db-init-script',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    LucideDatabase,
    LucideUploadCloud,
    LucideFileCode,
    LucideDownload,
    LucideTrash2,
    LucideAlertTriangle,
    LucideFileUp,
    LucideInfo
  ],
  templateUrl: './ide-db-init-script.component.html',
  styleUrl: './ide-db-init-script.component.scss'
})
export class IdeDbInitScriptComponent {
  readonly dbInitScript = input<string>('');
  readonly hasDatabaseService = input<boolean>(false);

  readonly initScriptChange = output<string>();

  readonly isDragging = signal<boolean>(false);
  readonly fileName = signal<string | null>(null);
  readonly fileNames = signal<string[]>([]);
  readonly fileSize = signal<number | null>(null);
  readonly errorMessage = signal<string | null>(null);

  readonly fileCount = computed(() => this.fileNames().length);
  readonly fileNamesTitle = computed(() => this.fileNames().join(' + '));

  readonly currentScript = computed(() => this.dbInitScript() || '');
  readonly hasContent = computed(() => this.currentScript().trim().length > 0);

  readonly lineCount = computed(() => {
    const text = this.currentScript().trim();
    return text ? text.split('\n').length : 0;
  });

  readonly approxStatementsCount = computed(() => {
    const text = this.currentScript().trim();
    if (!text) return 0;
    const matches = text.match(/;\s*(\n|$)/g);
    return matches ? matches.length : 1;
  });

  readonly byteSize = computed(() => {
    if (this.fileSize() !== null) return this.fileSize()!;
    return new Blob([this.currentScript()]).size;
  });

  readonly formattedSize = computed(() => {
    const bytes = this.byteSize();
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  });

  /**
   * Tiempo estimado de aprovisionamiento por alumno. Calibrado con mediciones reales
   * sobre el PostgreSQL de aula (ADR-075): 100 sentencias 0,14 s; 500 0,18 s;
   * 1.000 0,22 s; 5.000 0,64 s; 20.000 2,05 s. La curva medida es 0,1 s fijos más
   * una diezmilésima de segundo por sentencia, con la transacción ya aplicada.
   */
  readonly estimatedProvisionSeconds = computed(() => {
    const statements = this.approxStatementsCount();
    if (statements <= 0) {
      return 0;
    }
    return 0.1 + statements / 10000;
  });

  readonly estimatedProvisionLabel = computed(() => {
    const seconds = this.estimatedProvisionSeconds();
    if (seconds <= 0) {
      return '';
    }
    if (seconds < 1) {
      return `≈ ${Math.round(seconds * 1000)} ms por alumno`;
    }
    return `≈ ${seconds.toFixed(1).replace('.', ',')} s por alumno`;
  });

  readonly isOversized = computed(() => this.byteSize() > MAX_RECOMMENDED_SQL_BYTES);

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(true);
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(false);
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging.set(false);

    const files = Array.from(event.dataTransfer?.files ?? []);
    if (files.length) {
      void this.processFiles(files);
    }
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = Array.from(input?.files ?? []);
    if (files.length) {
      void this.processFiles(files);
    }
    input.value = '';
  }

  processFile(file: File): void {
    void this.processFiles([file]);
  }

  /**
   * Concatena los archivos en el orden en que se seleccionaron. Cada archivo
   * conserva su contenido; entre uno y otro queda una línea en blanco.
   */
  async processFiles(files: File[]): Promise<void> {
    if (!files.length) {
      return;
    }
    this.errorMessage.set(null);
    const validExtensions = ['.sql', '.txt', '.js'];
    const invalid = files.filter(file =>
      !validExtensions.some(ext => file.name.toLowerCase().endsWith(ext))
    );
    if (invalid.length) {
      const names = invalid.map(file => file.name).join(', ');
      this.errorMessage.set(`Formato de archivo no habitual en ${names}. Se recomienda .sql o texto plano.`);
    }

    try {
      const texts = await Promise.all(files.map(file => this.readFileText(file)));
      const script = texts.map(text => text.replace(/\s+$/, '')).join('\n\n');
      this.fileNames.set(files.map(file => file.name));
      this.fileName.set(files.length === 1 ? files[0].name : `${files.length} archivos`);
      this.fileSize.set(files.reduce((total, file) => total + file.size, 0));
      this.initScriptChange.emit(script);
    } catch {
      this.errorMessage.set('Error al leer un archivo. Asegúrese de que sea texto plano codificado en UTF-8.');
    }
  }

  private readFileText(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
      reader.onerror = () => reject(reader.error ?? new Error('read error'));
      reader.readAsText(file);
    });
  }

  onTextareaChange(newText: string): void {
    this.fileSize.set(null);
    this.fileNames.set([]);
    this.initScriptChange.emit(newText);
  }

  clearScript(): void {
    this.fileName.set(null);
    this.fileNames.set([]);
    this.fileSize.set(null);
    this.errorMessage.set(null);
    this.initScriptChange.emit('');
  }

  downloadScript(): void {
    const script = this.currentScript();
    if (!script) return;

    const blob = new Blob([script], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = this.fileName() || 'init_script.sql';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }
}
