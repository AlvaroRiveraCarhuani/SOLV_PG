import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';
import { ToastHostComponent } from '@shared/components/toast-host/toast-host.component';
import { HelpCenterComponent } from '@shared/help/help-center.component';
import { TutorialPanelComponent } from '@shared/help/tutorial-panel.component';
import { NavSection } from '@shared/components/sidebar/sidebar.model';

export const ADMIN_NAV_SECTIONS: NavSection[] = [
  {
    title: 'INFRAESTRUCTURA',
    items: [
      { label: 'Salud y Recursos', route: '/admin/dashboard', iconName: 'activity', exact: true },
      { label: 'Catálogo de Entornos', route: '/admin/plantillas', iconName: 'layers' },
      { label: 'Solicitudes de entornos', route: '/admin/solicitudes-entorno', iconName: 'inbox' },
      { label: 'Catálogo de Tecnologías', route: '/admin/tecnologias', iconName: 'boxes' },
      { label: 'Manual del administrador', route: '/admin/manual', iconName: 'book' }
    ]
  },
  {
    title: 'ACADÉMICO Y GESTIÓN',
    items: [
      { label: 'Categorías Académicas', route: '/admin/categorias', iconName: 'tag' },
      { label: 'Cursos', route: '/admin/cursos', iconName: 'book' },
      { label: 'Docentes', route: '/admin/docentes', iconName: 'users' },
      { label: 'Estudiantes', route: '/admin/estudiantes', iconName: 'graduation-cap' },
      { label: 'Configuración', route: '/admin/configuracion', iconName: 'sliders' }
    ]
  },
  {
    title: 'AUDITORÍA',
    items: [
      { label: 'Auditoría', route: '/admin/auditoria', iconName: 'shield-alert' }
    ]
  }
];

@Component({
  selector: 'admin-layout',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    TopbarComponent,
    SidebarComponent,
    ToastHostComponent,
    HelpCenterComponent,
    TutorialPanelComponent
  ],
  template: `
    <div class="shell-container">
      <topbar [showHelpButton]="true" (toggleSidebar)="toggleSidebar()" />
      <div class="shell-body">
        <sidebar 
          [collapsed]="isSidebarCollapsed()" 
          [sections]="adminSections" 
        />
        <main class="shell-content">
          <router-outlet />
        </main>
      </div>
      <help-center />
      <tutorial-panel />
      <toast-host />
    </div>
  `,
  styleUrl: './admin-layout.component.scss',
})
export class AdminLayoutComponent {
  isSidebarCollapsed = signal<boolean>(false);
  adminSections = ADMIN_NAV_SECTIONS;

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }
}
