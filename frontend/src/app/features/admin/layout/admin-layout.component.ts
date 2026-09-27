import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';
import { NavSection } from '@shared/components/sidebar/sidebar.model';

export const ADMIN_NAV_SECTIONS: NavSection[] = [
  {
    title: 'INFRAESTRUCTURA',
    items: [
      { label: 'Salud y Recursos', route: '/admin/dashboard', iconName: 'activity', exact: true },
      { label: 'Plantillas Docker', route: '/admin/plantillas', iconName: 'layers' },
      { label: 'Modelos y Categorías', route: '/admin/modelos-categorias', iconName: 'boxes' },
      { label: 'Manual del administrador', route: '/admin/manual', iconName: 'book' }
    ]
  },
  {
    title: 'ACADÉMICO Y GESTIÓN',
    items: [
      { label: 'Cursos', route: '/admin/cursos', iconName: 'book' },
      { label: 'Docentes', route: '/admin/docentes', iconName: 'users' },
      { label: 'Estudiantes', route: '/admin/estudiantes', iconName: 'graduation-cap' },
      { label: 'Configuración', route: '/admin/configuracion', iconName: 'sliders' }
    ]
  },
  {
    isFooter: true,
    items: [
      { label: 'Auditoría', route: '/admin/auditoria', iconName: 'shield-alert' }
    ]
  }
];

@Component({
  selector: 'admin-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, TopbarComponent, SidebarComponent],
  template: `
    <div class="shell-container">
      <topbar (toggleSidebar)="toggleSidebar()" />
      <div class="shell-body">
        <sidebar 
          [collapsed]="isSidebarCollapsed()" 
          [sections]="adminSections" 
        />
        <main class="shell-content">
          <router-outlet />
        </main>
      </div>
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
