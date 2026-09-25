import { Component, input, output, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { LucideBookOpen, LucideX, LucideExternalLink } from '@lucide/angular';

@Component({
  selector: 'solv-help-drawer',
  standalone: true,
  imports: [CommonModule, RouterModule, LucideBookOpen, LucideX, LucideExternalLink],
  template: `
    @if (isOpen()) {
      <div 
        class="drawer-backdrop" 
        (click)="onClose()" 
        aria-hidden="true"
      ></div>
      <aside 
        class="help-drawer" 
        role="complementary" 
        [attr.aria-label]="title()"
      >
        <div class="drawer-header">
          <div class="drawer-title-box">
            <svg lucideBookOpen class="drawer-icon" aria-hidden="true"></svg>
            <div class="drawer-titles">
              <h3 class="drawer-title">{{ title() }}</h3>
              @if (stepNumber() !== null) {
                <span class="drawer-step-badge">Paso {{ stepNumber() }}</span>
              }
            </div>
          </div>
          <button 
            type="button" 
            class="btn-close-drawer" 
            (click)="onClose()" 
            aria-label="Cerrar panel de ayuda"
          >
            <svg lucideX class="icon-close" aria-hidden="true"></svg>
          </button>
        </div>

        <div class="drawer-body">
          <ng-content></ng-content>
        </div>

        <div class="drawer-footer">
          <a 
            routerLink="/admin/manual" 
            target="_blank" 
            rel="noopener noreferrer" 
            class="manual-link"
          >
            <span>{{ manualLinkText() }}</span>
            <svg lucideExternalLink class="link-icon" aria-hidden="true"></svg>
          </a>
        </div>
      </aside>
    }
  `,
  styleUrls: ['./help-drawer.component.scss']
})
export class SolvHelpDrawerComponent {
  isOpen = input<boolean>(false);
  title = input<string>('Ayuda Contextual');
  stepNumber = input<number | null>(null);
  manualLinkText = input<string>('Ver manual completo de administración');

  closed = output<void>();

  onClose(): void {
    this.closed.emit();
  }

  @HostListener('document:keydown.escape', ['$event'])
  onKeydown(event: Event): void {
    if (this.isOpen()) {
      event.stopPropagation();
      this.onClose();
    }
  }
}
