import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideCheckCircle2, 
  LucideAlertTriangle, 
  LucideAlertCircle, 
  LucideInfo, 
  LucideRefreshCw 
} from '@lucide/angular';

export type FieldMessageVariant = 'success' | 'warning' | 'error' | 'info';

@Component({
  selector: 'solv-field-message',
  standalone: true,
  imports: [
    CommonModule, 
    LucideCheckCircle2, 
    LucideAlertTriangle, 
    LucideAlertCircle, 
    LucideInfo, 
    LucideRefreshCw
  ],
  template: `
    <div 
      class="field-message" 
      [class]="'variant-' + variant()" 
      role="status"
      [attr.aria-live]="variant() === 'error' ? 'assertive' : 'polite'"
    >
      <div class="message-content">
        @if (showIcon()) {
          <span class="message-icon" aria-hidden="true">
            @switch (variant()) {
              @case ('success') {
                <svg lucideCheckCircle2 class="icon"></svg>
              }
              @case ('warning') {
                <svg lucideAlertTriangle class="icon"></svg>
              }
              @case ('error') {
                <svg lucideAlertCircle class="icon"></svg>
              }
              @default {
                <svg lucideInfo class="icon"></svg>
              }
            }
          </span>
        }
        <span class="message-text">
          <ng-content>{{ message() }}</ng-content>
        </span>
      </div>

      @if (actionLabel()) {
        <button 
          type="button" 
          class="message-action-btn"
          (click)="actionClicked.emit()"
          [attr.aria-label]="actionLabel()"
        >
          @if (showActionIcon()) {
            <svg lucideRefreshCw class="action-icon"></svg>
          }
          <span>{{ actionLabel() }}</span>
        </button>
      }
    </div>
  `,
  styleUrls: ['./field-message.component.scss']
})
export class SolvFieldMessageComponent {
  variant = input<FieldMessageVariant>('info');
  message = input<string>('');
  showIcon = input<boolean>(true);
  actionLabel = input<string | null>(null);
  showActionIcon = input<boolean>(true);

  actionClicked = output<void>();
}
