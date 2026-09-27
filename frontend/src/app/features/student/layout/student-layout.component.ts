import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';

@Component({
  selector: 'student-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, TopbarComponent, SidebarComponent],
  template: `
    <div class="shell-container">
      <topbar (toggleSidebar)="toggleSidebar()" />
      <div class="shell-body">
        <sidebar [collapsed]="isSidebarCollapsed()" />
        <main class="shell-content">
          <router-outlet />
        </main>
      </div>
    </div>
  `,
  styleUrl: './student-layout.component.scss',
})
export class StudentLayoutComponent {
  isSidebarCollapsed = signal<boolean>(false);

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }
}
