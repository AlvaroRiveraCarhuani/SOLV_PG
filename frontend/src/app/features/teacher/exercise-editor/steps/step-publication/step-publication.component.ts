import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideShield, LucideCode2, LucideCheckSquare } from '@lucide/angular';
import { ExerciseEditorStore } from '../../exercise-editor.store';
import { ASTRulesBuilderComponent } from './ast-rules-builder/ast-rules-builder.component';
import { BoilerplateTabsComponent } from './boilerplate-tabs/boilerplate-tabs.component';
import { PublicationChecklistComponent } from './publication-checklist/publication-checklist.component';

@Component({
  selector: 'step-publication',
  standalone: true,
  imports: [
    CommonModule,
    LucideShield,
    LucideCode2,
    LucideCheckSquare,
    ASTRulesBuilderComponent,
    BoilerplateTabsComponent,
    PublicationChecklistComponent
  ],
  templateUrl: './step-publication.component.html',
  styleUrl: './step-publication.component.scss'
})
export class StepPublicationComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);

  ngOnInit(): void {
    // Al entrar al Paso 3, consultar checklist de publicación automáticamente
    this.store.refreshChecklist();
  }
}
