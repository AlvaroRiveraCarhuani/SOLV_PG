import { Component, ChangeDetectionStrategy, computed, input } from '@angular/core';
import { TechLogoComponent } from '../../../components/tech-logo/tech-logo.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  LucideCheckCircle2,
  LucideAlertTriangle,
  LucideXCircle,
  LucideOrbit
} from '@lucide/angular';
import { ConstellationNode, StackConstellation } from '../../template-studio.utils';

@Component({
  selector: 'stack-constellation',
  standalone: true,
  imports: [
    TechLogoComponent,
    MachineDataDirective,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideXCircle,
    LucideOrbit
  ],
  templateUrl: './stack-constellation.component.html',
  styleUrl: './stack-constellation.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StackConstellationComponent {
  readonly model = input<StackConstellation | null>(null);
  readonly title = input<string>('Mapa del Entorno');

  // Satélites ordenados por severidad: primero faltantes, luego incompatibles,
  // después disponibles, para que las advertencias queden próximas al centro.
  readonly orderedSatellites = computed<ConstellationNode[]>(() => {
    const nodes = [...(this.model()?.satellites || [])];
    const rank = (n: ConstellationNode): number =>
      n.availability === 'missing' ? 0 : n.availability === 'incompatible' ? 1 : 2;
    nodes.sort((a, b) => rank(a) - rank(b));
    return nodes;
  });

  readonly hasWarnings = computed<boolean>(() =>
    (this.model()?.satellites || []).some((n) => n.availability !== 'available')
  );
}