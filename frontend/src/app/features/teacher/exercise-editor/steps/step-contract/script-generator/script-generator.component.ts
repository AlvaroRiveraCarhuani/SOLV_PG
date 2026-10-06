import {
  Component,
  Input,
  Output,
  EventEmitter,
  inject,
  signal,
  computed
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucidePlay,
  LucideCheck,
  LucideRefreshCw,
  LucideAlertCircle,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideTerminal,
  LucideChevronDown,
  LucideChevronUp
} from '@lucide/angular';
import { TeacherCourseService } from '../../../../services/teacher-course.service';
import { ScriptGenerationResponse, ScriptCaseValidationItem } from '../../../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

const DEFAULT_PYTHON_SCRIPT = `import json
import random

cases = []
for _ in range(50):
    n = random.randint(1, 100)
    arr = [random.randint(-1000, 1000) for _ in range(n)]
    input_str = f"{n}\\n{' '.join(map(str, arr))}"
    expected_output = str(sum(arr))
    cases.append({
        "input": input_str,
        "expected_output": expected_output
    })

print(json.dumps(cases))`;

@Component({
  selector: 'app-script-generator',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlay,
    LucideCheck,
    LucideRefreshCw,
    LucideAlertCircle,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideTerminal,
    LucideChevronDown,
    LucideChevronUp,
    MachineDataDirective
  ],
  templateUrl: './script-generator.component.html',
  styleUrl: './script-generator.component.scss'
})
export class ScriptGeneratorComponent {
  private teacherCourseService = inject(TeacherCourseService);

  @Input() exerciseId?: string;
  @Output() casesGenerated = new EventEmitter<ScriptCaseValidationItem[]>();

  isExpanded = signal<boolean>(false);
  scriptCode = signal<string>(DEFAULT_PYTHON_SCRIPT);
  isExecuting = signal<boolean>(false);
  isImporting = signal<boolean>(false);
  previewResult = signal<ScriptGenerationResponse | null>(null);
  errorMessage = signal<string | null>(null);

  toggleExpand(): void {
    this.isExpanded.set(!this.isExpanded());
  }

  loadTemplate(): void {
    this.scriptCode.set(DEFAULT_PYTHON_SCRIPT);
    this.previewResult.set(null);
    this.errorMessage.set(null);
  }

  executeScript(): void {
    const code = this.scriptCode();
    if (!code.trim() || this.isExecuting()) return;

    this.isExecuting.set(true);
    this.errorMessage.set(null);
    this.previewResult.set(null);

    // Si no se tiene un exerciseId persistido aún (creando borrador nuevo),
    // se puede usar 'draft' o simulación local si aplica
    const targetId = this.exerciseId || 'draft';

    this.teacherCourseService.generateCasesFromScript(targetId, code, true).subscribe({
      next: (res) => {
        this.previewResult.set(res);
        this.isExecuting.set(false);
      },
      error: (err) => {
        this.isExecuting.set(false);
        const msg = err.error?.message || err.error?.error || 'Error al ejecutar el script en la sandbox';
        this.errorMessage.set(msg);
      }
    });
  }

  confirmAndAdd(): void {
    const preview = this.previewResult();
    const code = this.scriptCode();

    if (!preview || !preview.can_import || this.isImporting()) return;

    const targetId = this.exerciseId || 'draft';

    this.isImporting.set(true);
    this.errorMessage.set(null);

    this.teacherCourseService.generateCasesFromScript(targetId, code, false).subscribe({
      next: (res) => {
        this.isImporting.set(false);
        if (preview.cases) {
          this.casesGenerated.emit(preview.cases);
        }
        this.previewResult.set(null);
      },
      error: (err) => {
        this.isImporting.set(false);
        const msg = err.error?.message || err.error?.error || 'Error al guardar los casos generados';
        this.errorMessage.set(msg);
      }
    });
  }
}
