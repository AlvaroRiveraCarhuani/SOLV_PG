import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { 
  LucideHome, 
  LucideTerminal, 
  LucideBookOpen, 
  LucideAward, 
  LucideHistory, 
  LucideSettings,
  LucideActivity,
  LucideUsers,
  LucideLayers,
  LucideSliders,
  LucideShieldAlert,
  LucideFileText
} from '@lucide/angular';
import { NavSection } from './sidebar.model';

export const DEFAULT_STUDENT_SECTIONS: NavSection[] = [
  {
    title: 'PRINCIPAL',
    items: [
      { label: 'Inicio', route: '/student', iconName: 'home', exact: true },
      { label: 'Laboratorios', route: '/student/workspaces', iconName: 'terminal' },
      { label: 'Mis Materias', route: '/student/courses', iconName: 'book' }
    ]
  },
  {
    title: 'ACADÉMICO',
    items: [
      { label: 'Evaluaciones', route: '/student/evaluations', iconName: 'award' },
      { label: 'Historial', route: '/student/history', iconName: 'history' }
    ]
  },
  {
    isFooter: true,
    items: [
      { label: 'Ajustes', route: '/student/settings', iconName: 'settings' }
    ]
  }
];

@Component({
  selector: 'solv-sidebar',
  standalone: true,
  imports: [
    CommonModule, 
    RouterModule, 
    LucideHome, 
    LucideTerminal, 
    LucideBookOpen, 
    LucideAward, 
    LucideHistory, 
    LucideSettings,
    LucideActivity,
    LucideUsers,
    LucideLayers,
    LucideSliders,
    LucideShieldAlert,
    LucideFileText
  ],
  template: `
    <aside class="sidebar" [class.collapsed]="collapsed()">
      <nav class="nav-menu">
        @for (sec of sections(); track $index) {
          <div class="nav-section" [class.mt-auto]="sec.isFooter">
            @if (sec.isFooter) {
              <div class="divider"></div>
            }
            @if (sec.title && !collapsed()) {
              <span class="section-title">{{ sec.title }}</span>
            }
            @for (item of sec.items; track item.route) {
              <a 
                [routerLink]="item.route" 
                routerLinkActive="active" 
                [routerLinkActiveOptions]="{ exact: item.exact ?? false }"
                class="nav-item"
                [title]="collapsed() ? item.label : ''">
                @switch (item.iconName) {
                  @case ('home') { <svg lucideHome class="nav-icon"></svg> }
                  @case ('terminal') { <svg lucideTerminal class="nav-icon"></svg> }
                  @case ('book') { <svg lucideBookOpen class="nav-icon"></svg> }
                  @case ('award') { <svg lucideAward class="nav-icon"></svg> }
                  @case ('history') { <svg lucideHistory class="nav-icon"></svg> }
                  @case ('settings') { <svg lucideSettings class="nav-icon"></svg> }
                  @case ('activity') { <svg lucideActivity class="nav-icon"></svg> }
                  @case ('users') { <svg lucideUsers class="nav-icon"></svg> }
                  @case ('layers') { <svg lucideLayers class="nav-icon"></svg> }
                  @case ('sliders') { <svg lucideSliders class="nav-icon"></svg> }
                  @case ('shield-alert') { <svg lucideShieldAlert class="nav-icon"></svg> }
                  @case ('file-text') { <svg lucideFileText class="nav-icon"></svg> }
                }
                <span class="nav-label">{{ item.label }}</span>
              </a>
            }
          </div>
        }
      </nav>
    </aside>
  `,
  styles: [`
    .sidebar {
      width: 240px;
      height: calc(100vh - 60px);
      background-color: var(--bg-surface, #FFFFFF);
      border-right: 1px solid var(--border-subtle, #E2E8F0);
      display: flex;
      flex-direction: column;
      transition: width var(--transition-normal, 200ms ease);
      overflow-x: hidden;
      user-select: none;
      position: sticky;
      top: 60px;

      &.collapsed {
        width: 64px;

        .section-title,
        .nav-label {
          display: none;
        }

        .nav-item {
          justify-content: center;
          padding: var(--space-2-5, 10px) 0;
        }
      }
    }

    .nav-menu {
      display: flex;
      flex-direction: column;
      height: 100%;
      padding: var(--space-4, 16px) var(--space-2-5, 10px);
      gap: var(--space-4, 16px);
    }

    .nav-section {
      display: flex;
      flex-direction: column;
      gap: var(--space-1, 4px);
    }

    .mt-auto {
      margin-top: auto;
    }

    .section-title {
      font-size: 10px;
      font-weight: 700;
      color: var(--text-muted, #94A3B8);
      letter-spacing: 0.05em;
      padding: var(--space-1, 4px) var(--space-3, 12px);
      margin-bottom: var(--space-1, 4px);
    }

    .divider {
      height: 1px;
      background-color: var(--border-subtle, #E2E8F0);
      margin: var(--space-2, 8px) var(--space-2, 8px);
    }

    .nav-item {
      display: flex;
      align-items: center;
      gap: var(--space-3, 12px);
      padding: var(--space-2, 8px) var(--space-3, 12px);
      border-radius: var(--radius-md, 6px);
      color: var(--text-secondary, #64748B);
      text-decoration: none;
      font-size: var(--font-size-sm, 14px);
      font-weight: 500;
      transition: all var(--transition-fast, 150ms ease);

      &:hover {
        background-color: var(--bg-canvas, #F6F7F9);
        color: var(--text-primary, #0F172A);
      }

      &.active {
        background-color: var(--tenant-primary-subtle, rgba(37, 99, 235, 0.08));
        color: var(--tenant-primary, #2563EB);
        font-weight: 600;

        .nav-icon {
          color: var(--tenant-primary, #2563EB);
        }
      }

      .nav-icon {
        width: 18px;
        height: 18px;
        flex-shrink: 0;
        color: var(--text-secondary, #64748B);
        transition: color var(--transition-fast, 150ms ease);
      }

      .nav-label {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
    }
  `]
})
export class SidebarComponent {
  collapsed = input<boolean>(false);
  sections = input<NavSection[]>(DEFAULT_STUDENT_SECTIONS);
}
