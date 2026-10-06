import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  OnInit, 
  inject, 
  signal, 
  computed, 
  HostListener 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideSparkles, 
  LucideX, 
  LucideAlertTriangle, 
  LucidePlus, 
  LucideCheckSquare, 
  LucideSquare, 
  LucideEye, 
  LucideEyeOff
} from '@lucide/angular';
import { 
  GeneratedFuzzCase, 
  FuzzGenerationRequest 
} from '../../models/teacher.models';
import { TeacherFuzzingService } from '../../services/teacher-fuzzing.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'fuzzing-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSparkles,
    LucideX,
    LucideAlertTriangle,
    LucidePlus,
    LucideCheckSquare,
    LucideSquare,
    LucideEye,
    LucideEyeOff,
    MachineDataDirective
  ],
  templateUrl: './fuzzing-modal.component.html',
  styleUrl: './fuzzing-modal.component.scss'
})
export class FuzzingModalComponent implements OnInit {
  @Input({ required: true }) exerciseId!: string;
  @Input() exerciseTitle: string = 'Laboratorio';
  @Input() language: string = 'python';
  @Input() referenceSolution: string = '';
  @Output() close = new EventEmitter<void>();
  @Output() applied = new EventEmitter<number>();

  private fuzzingService = inject(TeacherFuzzingService);

  isGenerating = this.fuzzingService.isGenerating;
  isApplying = this.fuzzingService.isApplying;
  report = this.fuzzingService.currentReport;

  // Filtros / Opciones de Generación
  selectedLanguage = signal<string>('python');
  
  availableTypes = [
    { id: 'integer', label: 'Enteros' },
    { id: 'float', label: 'Decimales (Float)' },
    { id: 'string', label: 'Cadenas de Texto' },
    { id: 'array', label: 'Arreglos / Listas' },
    { id: 'matrix', label: 'Matrices' },
    { id: 'sql', label: 'Consultas SQL' }
  ];
  selectedTypes = signal<string[]>(['integer', 'string', 'array']);

  availableCategories = [
    { id: 'numeric_bounds', label: 'Límites Numéricos' },
    { id: 'empty_structures', label: 'Estructuras Vacías' },
    { id: 'unicode_special', label: 'Unicode y Especiales' },
    { id: 'injection_escape', label: 'Inyecciones y Escapes' },
    { id: 'large_input', label: 'Escala Masiva / Estrés' }
  ];
  selectedCategories = signal<string[]>([
    'numeric_bounds', 
    'empty_structures', 
    'unicode_special', 
    'injection_escape', 
    'large_input'
  ]);

  generatedCases = signal<GeneratedFuzzCase[]>([]);
  feedbackMessage = signal<string | null>(null);

  selectedCasesCount = computed(() => {
    return this.generatedCases().filter(c => c.selected).length;
  });

  allSelected = computed(() => {
    const list = this.generatedCases();
    return list.length > 0 && list.every(c => c.selected);
  });

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.onClose();
  }

  ngOnInit(): void {
    if (this.language) {
      this.selectedLanguage.set(this.language.toLowerCase());
    }
    this.generateCases();
  }

  toggleType(typeId: string): void {
    this.selectedTypes.update(types => {
      if (types.includes(typeId)) {
        return types.filter(t => t !== typeId);
      }
      return [...types, typeId];
    });
  }

  toggleCategory(catId: string): void {
    this.selectedCategories.update(cats => {
      if (cats.includes(catId)) {
        return cats.filter(c => c !== catId);
      }
      return [...cats, catId];
    });
  }

  generateCases(): void {
    const req: FuzzGenerationRequest = {
      exercise_id: this.exerciseId,
      target_language: this.selectedLanguage(),
      parameter_types: this.selectedTypes(),
      categories: this.selectedCategories(),
      reference_code: this.referenceSolution ? this.referenceSolution.trim() : undefined,
      count: 20
    };

    this.fuzzingService.generateFuzzCases(req).subscribe({
      next: (rep) => {
        this.generatedCases.set(rep.cases || []);
        this.feedbackMessage.set(null);
      },
      error: () => {
        this.feedbackMessage.set('Error al sintetizar casos de prueba.');
      }
    });
  }

  toggleSelectAll(): void {
    const targetState = !this.allSelected();
    this.generatedCases.update(list => {
      return list.map(item => ({ ...item, selected: targetState }));
    });
  }

  toggleCaseSelection(index: number): void {
    this.generatedCases.update(list => {
      const copy = [...list];
      if (copy[index]) {
        copy[index] = { ...copy[index], selected: !copy[index].selected };
      }
      return copy;
    });
  }

  toggleVisibility(index: number): void {
    this.generatedCases.update(list => {
      const copy = [...list];
      if (copy[index]) {
        copy[index] = { ...copy[index], is_public: !copy[index].is_public };
      }
      return copy;
    });
  }

  applySelectedCases(): void {
    const selected = this.generatedCases().filter(c => c.selected);
    if (selected.length === 0) return;

    this.fuzzingService.applyFuzzCases(this.exerciseId, selected).subscribe({
      next: (res) => {
        this.applied.emit(res.added_count);
        this.onClose();
      },
      error: (err) => {
        this.feedbackMessage.set(err?.error?.message || 'Error al persistir casos en el ejercicio.');
      }
    });
  }

  onClose(): void {
    this.close.emit();
  }
}
