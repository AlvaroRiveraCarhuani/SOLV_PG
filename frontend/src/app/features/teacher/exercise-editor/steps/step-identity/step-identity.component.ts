import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MetadataFormComponent } from './metadata-form.component';
import { StatementEditorComponent } from './statement-editor.component';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'step-identity',
  standalone: true,
  imports: [
    CommonModule,
    MetadataFormComponent,
    StatementEditorComponent
  ],
  template: `
    <div class="step-identity-layout">
      <section class="form-section">
        <metadata-form />
      </section>

      <section class="form-section">
        <statement-editor />
      </section>
    </div>
  `,
  styleUrl: './step-identity.component.scss'
})
export class StepIdentityComponent {
  readonly store = inject(ExerciseEditorStore);
}
