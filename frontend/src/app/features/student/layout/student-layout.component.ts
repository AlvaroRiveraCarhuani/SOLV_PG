import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { TopbarComponent } from '@shared/components/topbar/topbar.component';
import { SidebarComponent } from '@shared/components/sidebar/sidebar.component';
import { ClassPresenceService } from '../services/class-presence.service';

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
export class StudentLayoutComponent implements OnInit, OnDestroy {
  private readonly presence = inject(ClassPresenceService);

  isSidebarCollapsed = signal<boolean>(false);

  ngOnInit(): void {
    // La presencia en el aula sale de esta señal: sin ella el docente no puede
    // saber quién falta, solo quién está inscrito.
    this.presence.start();
  }

  ngOnDestroy(): void {
    this.presence.stop();
  }

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }
}
