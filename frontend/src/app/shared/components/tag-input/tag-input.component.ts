import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideTag, LucidePlus, LucideX } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';

@Component({
  selector: 'tag-input',
  standalone: true,
  imports: [
    CommonModule,
    LucideTag,
    LucidePlus,
    LucideX,
    MachineDataDirective,
    FieldMessageComponent
  ],
  template: `
    <div class="tags-container">
      <div class="tag-input-row">
        <input
          #inputEl
          [id]="inputId()"
          type="text"
          class="form-input tag-input"
          [placeholder]="placeholder()"
          (input)="clearError()"
          (keydown.enter)="handleAdd(inputEl); $event.preventDefault()"
        />
        <button
          type="button"
          class="btn-outline add-tag-btn"
          (click)="handleAdd(inputEl)"
        >
          <svg lucidePlus class="btn-icon"></svg>
          <span>Agregar</span>
        </button>
      </div>

      @if (errorMessage()) {
        <field-message variant="warning" [message]="errorMessage()!"></field-message>
      }

      @if (tags().length > 0) {
        <div class="tag-chips-list">
          @for (tag of tags(); track tag) {
            <span class="tag-chip">
              <svg lucideTag class="tag-icon"></svg>
              <span class="tag-text" machineData>{{ tag }}</span>
              <button
                type="button"
                class="remove-tag-btn"
                (click)="handleRemove(tag)"
                title="Eliminar etiqueta"
              >
                <svg lucideX class="remove-icon"></svg>
              </button>
            </span>
          }
        </div>
      }
    </div>
  `,
  styleUrl: './tag-input.component.scss'
})
export class TagInputComponent {
  inputId = input<string>('tag-input');
  placeholder = input<string>('Ej: arreglos, dp, recursión, ordenamiento...');
  tags = input<string[]>([]);

  tagsChange = output<string[]>();

  readonly errorMessage = signal<string | null>(null);

  clearError(): void {
    this.errorMessage.set(null);
  }

  handleAdd(inputElement: HTMLInputElement): void {
    const raw = inputElement.value;
    const clean = raw.trim().toLowerCase();

    if (!clean) {
      this.errorMessage.set('Ingresa un texto para agregar la etiqueta');
      return;
    }

    const current = this.tags();
    if (current.includes(clean)) {
      this.errorMessage.set('Esta etiqueta ya fue agregada');
      return;
    }

    this.clearError();
    this.tagsChange.emit([...current, clean]);
    inputElement.value = '';
  }

  handleRemove(tagToRemove: string): void {
    this.clearError();
    const updated = this.tags().filter(t => t !== tagToRemove);
    this.tagsChange.emit(updated);
  }
}
