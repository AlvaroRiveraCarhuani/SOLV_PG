import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';
import { NavSection } from '@shared/components/sidebar/sidebar.model';

import { HotkeysModalComponent } from '@shared/components/hotkeys-modal/hotkeys-modal.component';

export const TEACHER_NAV_SECTIONS: NavSection[] = [
  {
    title: 'DOCENCIA',
    items: [
      { label: 'Panel de Control', route: '/teacher/dashboard', iconName: 'activity', exact: true },
      { label: 'Evaluaciones', route: '/teacher/evaluaciones', iconName: 'award' }
    ]
  }
];

@Component({
  selector: 'teacher-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, TopbarComponent, SidebarComponent, HotkeysModalComponent],
  template: `
    <div class="shell-container">
      <topbar (toggleSidebar)="toggleSidebar()" />
      <div class="shell-body">
        <sidebar 
          [collapsed]="isSidebarCollapsed()" 
          [sections]="teacherSections" 
        />
        <main class="shell-content">
          <router-outlet />
        </main>
      </div>
      <hotkeys-modal />
    </div>
  `,
  styleUrl: './teacher-layout.component.scss',
})
export class TeacherLayoutComponent {
  isSidebarCollapsed = signal<boolean>(false);
  teacherSections = TEACHER_NAV_SECTIONS;

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }
}
