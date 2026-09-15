import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TechnicalIncident } from '@core/models/admin.model';
import { LucideAlertTriangle, LucideRotateCcw, LucideFileText, LucideInfo } from '@lucide/angular';

@Component({
  selector: 'solv-incidents-panel',
  standalone: true,
  imports: [CommonModule, LucideAlertTriangle, LucideRotateCcw, LucideFileText, LucideInfo],
  template: `
    <div class="panel-container">
      <div class="panel-header">
        <div class="header-title-group">
          <svg lucideAlertTriangle class="header-icon-danger"></svg>
          <h3 class="panel-title">Incidencias de Infraestructura</h3>
        </div>
        <span class="incident-badge">{{ incidents().length }} alertas</span>
      </div>

      <div class="panel-body">
        <!-- Lista de Incidencias OOM / Fallos -->
        <div class="incidents-list">
          @for (inc of incidents(); track inc.id) {
            <div class="incident-card" [class.oom-card]="inc.type === 'oom_killed'">
              <div class="card-top">
                <div class="tag-status">
                  <span class="dot-danger"></span>
                  <span class="tag-text">OOM Killed (Exit code 137)</span>
                </div>
                <span class="incident-time">{{ inc.timestamp }}</span>
              </div>

              <div class="card-content">
                <span class="student-info">{{ inc.student_name }}</span>
                <span class="course-sub">{{ inc.course_name }} &bull; {{ inc.workspace_id }}</span>
                <p class="incident-desc">{{ inc.description }}</p>
              </div>

              <div class="card-actions">
                <button 
                  class="btn-log" 
                  (click)="viewLogs.emit(inc.workspace_id)"
                  title="Inspeccionar traza del error">
                  <svg lucideFileText class="btn-icon"></svg>
                  <span>Ver Logs</span>
                </button>
                <button 
                  class="btn-restart" 
                  (click)="restartWorkspace.emit(inc.workspace_id)"
                  title="Restablecer contenedor del estudiante">
                  <svg lucideRotateCcw class="btn-icon"></svg>
                  <span>Reiniciar</span>
                </button>
              </div>
            </div>
          } @empty {
            <div class="empty-incidents">
              <span class="empty-icon">✓</span>
              <span class="empty-text">Sin incidencias técnicas en este momento. Todos los contenedores operan dentro de sus cuotas.</span>
            </div>
          }
        </div>

        <!-- Widget Contextual de Políticas QoS del Host -->
        <div class="qos-widget">
          <div class="qos-header">
            <svg lucideInfo class="qos-icon"></svg>
            <span class="qos-title">Umbral de Recursos y Políticas QoS</span>
          </div>
          <p class="qos-text">
            El servicio QoS del host supervisa la inactividad de red y consola. Si la ocupación de RAM excede el 85%, se pausarán instancias inactivas tras 15 minutos.
          </p>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .panel-container {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      display: flex;
      flex-direction: column;
      overflow: hidden;
      height: 100%;
    }

    .panel-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--space-4, 16px) var(--space-5, 20px);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
    }

    .header-title-group {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
    }

    .header-icon-danger {
      width: 16px;
      height: 16px;
      color: #DC2626;
    }

    .panel-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .incident-badge {
      font-size: 11px;
      font-weight: 600;
      color: #991B1B;
      background-color: #FEE2E2;
      padding: 2px 8px;
      border-radius: var(--radius-full, 9999px);
    }

    .panel-body {
      padding: var(--space-4, 16px);
      display: flex;
      flex-direction: column;
      gap: var(--space-4, 16px);
      overflow-y: auto;
    }

    .incidents-list {
      display: flex;
      flex-direction: column;
      gap: var(--space-3, 12px);
    }

    .incident-card {
      border: 1px solid #FECACA;
      background-color: #FFFDFD;
      border-radius: var(--radius-md, 6px);
      padding: var(--space-3, 12px);
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .tag-status {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      font-weight: 600;
      color: #B91C1C;
    }

    .dot-danger {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background-color: #DC2626;
    }

    .incident-time {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .card-content {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .student-info {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .course-sub {
      font-size: 11px;
      color: var(--text-secondary, #64748B);
    }

    .incident-desc {
      font-size: 12px;
      color: #475569;
      margin: 4px 0 0 0;
      line-height: 1.4;
    }

    .card-actions {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      margin-top: 4px;
    }

    .btn-log, .btn-restart {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 500;
      cursor: pointer;
      transition: all 150ms ease;

      .btn-icon {
        width: 12px;
        height: 12px;
      }
    }

    .btn-log {
      background-color: #FFFFFF;
      border: 1px solid var(--border-subtle, #CBD5E1);
      color: #334155;

      &:hover {
        background-color: #F8FAFC;
        border-color: #94A3B8;
      }
    }

    .btn-restart {
      background-color: #FEE2E2;
      border: 1px solid #FCA5A5;
      color: #991B1B;

      &:hover {
        background-color: #FCA5A5;
        color: #7F1D1D;
      }
    }

    .qos-widget {
      background-color: #F8FAFC;
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-md, 6px);
      padding: var(--space-3, 12px);
      display: flex;
      flex-direction: column;
      gap: 6px;
    }

    .qos-header {
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .qos-icon {
      width: 14px;
      height: 14px;
      color: var(--text-secondary, #475569);
    }

    .qos-title {
      font-size: 11px;
      font-weight: 700;
      color: var(--text-secondary, #475569);
      letter-spacing: 0.04em;
    }

    .qos-text {
      font-size: 12px;
      color: var(--text-muted, #64748B);
      margin: 0;
      line-height: 1.4;
    }

    .empty-incidents {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px 12px;
      text-align: center;
      gap: 8px;
    }

    .empty-icon {
      width: 28px;
      height: 28px;
      border-radius: 50%;
      background-color: #DCFCE7;
      color: #16A34A;
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
    }

    .empty-text {
      font-size: 12px;
      color: var(--text-secondary, #475569);
      line-height: 1.4;
    }
  `]
})
export class IncidentsPanelComponent {
  incidents = input.required<TechnicalIncident[]>();
  restartWorkspace = output<string>();
  viewLogs = output<string>();
}
