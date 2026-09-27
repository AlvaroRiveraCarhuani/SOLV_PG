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
  LucideFileText,
  LucideGraduationCap,
  LucideBoxes
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
  selector: 'sidebar',
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
    LucideFileText,
    LucideGraduationCap,
    LucideBoxes
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
                  @case ('graduation-cap') { <svg lucideGraduationCap class="nav-icon"></svg> }
                  @case ('layers') { <svg lucideLayers class="nav-icon"></svg> }
                  @case ('boxes') { <svg lucideBoxes class="nav-icon"></svg> }
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
  styleUrl: './sidebar.component.scss',
})
export class SidebarComponent {
  collapsed = input<boolean>(false);
  sections = input<NavSection[]>(DEFAULT_STUDENT_SECTIONS);
}
