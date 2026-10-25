import {
  Component,
  ChangeDetectionStrategy,
  computed,
  inject,
  input,
  output,
  signal
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ParsedRosterResult } from '@core/models/admin.model';
import {
  LucideUpload,
  LucideFileCheck,
  LucideAlertTriangle,
  LucideCheckCircle2
} from '@lucide/angular';

@Component({
  selector: 'roster-upload',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    MachineDataDirective,
    LucideUpload,
    LucideFileCheck,
    LucideAlertTriangle,
    LucideCheckCircle2
  ],
  templateUrl: './roster-upload.component.html',
  styleUrl: './roster-upload.component.scss'
})
export class RosterUploadComponent {
  private readonly http = inject(HttpClient);

  readonly parseEndpoint = input.required<string>();
  readonly recordLabel = input<string>('registros');

  readonly parsed = output<ParsedRosterResult>();

  readonly selectedFile = signal<File | null>(null);
  readonly isDragging = signal(false);
  readonly isParsing = signal(false);
  readonly parseError = signal<string | null>(null);
  readonly parsedRoster = signal<ParsedRosterResult | null>(null);

  readonly hasValid = computed(() => (this.parsedRoster()?.valid_emails.length ?? 0) > 0);

  onDragOver(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(true);
  }

  onDragLeave(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(false);
  }

  onDrop(e: DragEvent): void {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging.set(false);
    const files = e.dataTransfer?.files;
    if (files && files.length > 0) {
      this.processFile(files[0]);
    }
  }

  onFileInputChange(e: Event): void {
    const input = e.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.processFile(input.files[0]);
    }
  }

  processFile(file: File): void {
    const ext = file.name.slice(file.name.lastIndexOf('.')).toLowerCase();
    if (ext !== '.xlsx' && ext !== '.csv') {
      this.parseError.set('Formato no admitido. Por favor selecciona un archivo .xlsx o .csv.');
      return;
    }

    this.selectedFile.set(file);
    this.parseError.set(null);
    this.isParsing.set(true);
    this.parsedRoster.set(null);

    const formData = new FormData();
    formData.append('file', file);

    this.http.post<ParsedRosterResult>(this.parseEndpoint(), formData).subscribe({
      next: (res) => {
        this.isParsing.set(false);
        if (!res || res.total_detected === 0) {
          this.parseError.set('No se encontraron correos electrónicos en el archivo seleccionado.');
          return;
        }
        this.parsedRoster.set(res);
        this.parsed.emit(res);
      },
      error: () => {
        this.isParsing.set(false);
        this.parseError.set('No se pudo leer el archivo. Comprueba que no esté dañado e intenta de nuevo.');
      }
    });
  }

  reset(): void {
    this.selectedFile.set(null);
    this.parsedRoster.set(null);
    this.parseError.set(null);
  }
}
