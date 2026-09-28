import { Directive } from '@angular/core';

@Directive({
  selector: '[machineData]',
  standalone: true,
  host: { class: 'font-mono' }
})
export class MachineDataDirective {}