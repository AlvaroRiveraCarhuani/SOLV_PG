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
  readonly fileSize = signal<number | null>(null);
  readonly errorMessage = signal<string | null>(null);

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

    const file = event.dataTransfer?.files?.[0];
    if (file) {
      this.processFile(file);
    }
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input?.files?.[0];
    if (file) {
      this.processFile(file);
    }
    input.value = '';
  }

  processFile(file: File): void {
    this.errorMessage.set(null);
    const validExtensions = ['.sql', '.txt', '.js'];
    const lower = file.name.toLowerCase();
    const hasValidExt = validExtensions.some(ext => lower.endsWith(ext));

    if (!hasValidExt) {
      this.errorMessage.set('Formato de archivo no habitual. Se recomienda utilizar un archivo .sql o de texto plano.');
    }

    const reader = new FileReader();
    reader.onload = () => {
      const text = typeof reader.result === 'string' ? reader.result : '';
      this.fileName.set(file.name);
      this.fileSize.set(file.size);
      this.initScriptChange.emit(text);
    };
    reader.onerror = () => {
      this.errorMessage.set('Error al leer el archivo. Asegúrese de que sea texto plano codificado en UTF-8.');
    };
    reader.readAsText(file);
  }

  onTextareaChange(newText: string): void {
    this.fileSize.set(null);
    this.initScriptChange.emit(newText);
  }

  clearScript(): void {
    this.fileName.set(null);
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
