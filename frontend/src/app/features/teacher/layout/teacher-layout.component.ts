import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';
import { NavSection } from '@shared/components/sidebar/sidebar.model';

import { HotkeysModalComponent } from '@shared/components/hotkeys-modal/hotkeys-modal.component';
import { HelpCenterComponent } from '@shared/help/help-center.component';
import { TutorialPanelComponent } from '@shared/help/tutorial-panel.component';

export const TEACHER_NAV_SECTIONS: NavSection[] = [
  {
    title: 'HOY',
    items: [
      { label: 'Inicio', route: '/teacher/dashboard', iconName: 'home', exact: true }
    ]
  },
  {
    title: 'DOCENCIA',
    items: [
      { label: 'Aula', route: '/teacher/aula', iconName: 'terminal' },
      { label: 'Por calificar', route: '/teacher/por-calificar', iconName: 'file-text' }
    ]
  },
  {
    title: 'CONTENIDOS',
    items: [
      { label: 'Banco de Ejercicios', route: '/teacher/exercises', iconName: 'boxes' },
      { label: 'Catálogo de Entornos', route: '/teacher/templates', iconName: 'layers' }
    ]
  },
  {
    title: 'CUENTA',
    items: [
      { label: 'Mi Inteligencia Artificial', route: '/teacher/configuracion', iconName: 'sparkles' }
    ]
  }
];

@Component({
  selector: 'teacher-layout',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    TopbarComponent,
    SidebarComponent,
    HotkeysModalComponent,
    HelpCenterComponent,
    TutorialPanelComponent
  ],
  template: `
    <div class="shell-container">
      <topbar [showHelpButton]="true" (toggleSidebar)="toggleSidebar()" />
      <div class="shell-body">
        <sidebar 
          [collapsed]="isSidebarCollapsed()" 
          [sections]="teacherSections" 
        />
        <main class="shell-content">
          <router-outlet />
        </main>
      </div>
      <help-center [manualLink]="null" />
      <tutorial-panel />
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
