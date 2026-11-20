import { Component, input } from '@angular/core';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { LanguageRunner } from '../../../models/admin-templates.models';

/**
 * Lenguajes declarados por un entorno de evaluación. Se comparte entre el
 * catálogo administrativo y el modal de revisión para que la lista de
 * ejecutores se lea igual en ambos lugares.
 */
@Component({
  selector: 'judge-language-list',
  standalone: true,
  imports: [MachineDataDirective],
  templateUrl: './judge-language-list.component.html',
  styleUrls: ['./judge-language-list.component.scss']
})
export class JudgeLanguageListComponent {
  readonly runners = input<LanguageRunner[]>([]);
  /** Muestra también el comando de cada lenguaje, no solo su nombre. */
  readonly showCommands = input(false);
}
