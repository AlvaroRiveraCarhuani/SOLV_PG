import { Pipe, PipeTransform } from '@angular/core';
import { getEnvironmentMeta } from '../constants/environment-types.constants';

export type EnvironmentLabelFormat = 'full' | 'short' | 'description' | 'badgeClass';

@Pipe({
  name: 'environmentLabel',
  standalone: true
})
export class EnvironmentLabelPipe implements PipeTransform {
  transform(value: string | null | undefined, format: EnvironmentLabelFormat = 'full'): string {
    const meta = getEnvironmentMeta(value);
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
