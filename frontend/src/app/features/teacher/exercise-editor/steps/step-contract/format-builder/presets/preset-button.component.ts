import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideHash, 
  LucideListOrdered, 
  LucideGrid, 
  LucideNetwork, 
  LucideType, 
  LucideLayoutTemplate 
} from '@lucide/angular';
import { FormatPreset } from '../format-builder.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'preset-button',
  standalone: true,
  imports: [
    CommonModule,
    LucideHash,
    LucideListOrdered,
    LucideGrid,
    LucideNetwork,
    LucideType,
    LucideLayoutTemplate,
    MachineDataDirective
  ],
  template: `
    <button 
      type="button" 
      class="preset-btn"
      [class.active]="active()"
      (click)="selected.emit(preset())"
      [title]="preset().description"
    >
      <div class="preset-icon-box">
        @switch (preset().id) {
          @case ('single_int') {
            <svg lucideHash class="icon"></svg>
          }
          @case ('n_and_ints') {
            <svg lucideListOrdered class="icon"></svg>
          }
          @case ('matrix_nm') {
            <svg lucideGrid class="icon"></svg>
          }
          @case ('graph_edges') {
            <svg lucideNetwork class="icon"></svg>
          }
          @case ('single_string') {
            <svg lucideType class="icon"></svg>
          }
          @default {
            <svg lucideLayoutTemplate class="icon"></svg>
          }
        }
      </div>
      <div class="preset-info">
        <div class="preset-label-row">
          <span class="preset-label">{{ preset().label }}</span>
          <span class="preset-badge font-mono" machineData>{{ preset().lineCountText }}</span>
        </div>
        <span class="preset-desc">{{ preset().description }}</span>
      </div>
    </button>
  `,
  styleUrl: './preset-button.component.scss'
})
export class PresetButtonComponent {
  preset = input.required<FormatPreset>();
  active = input<boolean>(false);
  selected = output<FormatPreset>();
}
