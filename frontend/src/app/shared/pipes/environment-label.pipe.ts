import { Pipe, PipeTransform } from '@angular/core';
import { getEnvironmentMeta } from '../constants/environment-types.constants';

export type EnvironmentLabelFormat = 'full' | 'short' | 'description' | 'badgeClass';

/**
 * Etiqueta visible de un tipo de entorno. Es el único punto donde los nombres
 * oficiales pasan a la interfaz: no escribir «Entorno de Desarrollo» ni
 * «Entorno de Evaluación» en plantillas o componentes.
 */
@Pipe({
  name: 'environmentLabel',
  standalone: true
})
export class EnvironmentLabelPipe implements PipeTransform {
  transform(value: string | null | undefined, format: EnvironmentLabelFormat = 'full'): string {
    const meta = getEnvironmentMeta(value);
    if (!meta) return '';
    switch (format) {
      case 'short':
        return meta.short;
      case 'description':
        return meta.description;
      case 'badgeClass':
        return meta.badgeClass;
      case 'full':
      default:
        return meta.label;
    }
  }
}
