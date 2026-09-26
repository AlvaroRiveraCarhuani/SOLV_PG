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
  styles: [`
    .shell-container {
      display: flex;
      flex-direction: column;
      min-height: 100vh;
      background-color: var(--bg-canvas, #F6F7F9);
    }

    .shell-body {
      display: flex;
      flex: 1;
      overflow: hidden;
    }

    .shell-content {
      flex: 1;
      height: calc(100vh - 60px);
      overflow-y: auto;
      padding: var(--space-6, 24px);
    }
  `]
})
export class StudentLayoutComponent {
  isSidebarCollapsed = signal<boolean>(false);

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }
}
