import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TechnicalIncident } from '@core/models/admin.model';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { LucideAlertTriangle, LucideCheckCircle, LucideFileText, LucideInfo } from '@lucide/angular';

@Component({
  selector: 'incidents-panel',
  standalone: true,
  imports: [CommonModule, MachineDataDirective, LucideAlertTriangle, LucideCheckCircle, LucideFileText, LucideInfo],
  template: `
    <div class="panel-container">
      <div class="panel-header">
        <div class="header-title-group">
          <svg lucideAlertTriangle class="header-icon-danger"></svg>
          <h3 class="panel-title">Incidencias de Infraestructura</h3>
        </div>
        <span class="incident-badge" [class.zero]="incidents().length === 0">
          {{ incidents().length }} {{ incidents().length === 1 ? 'alerta' : 'alertas' }}
        </span>
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
                <span class="incident-time" machineData>{{ inc.timestamp }}</span>
              </div>

              <div class="card-content">
                <span class="student-info">{{ inc.student_name }}</span>
                <span class="course-sub">{{ inc.course_name }} &bull; <span machineData>{{ inc.workspace_id }}</span></span>
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
                  class="btn-resolve" 
                  (click)="resolveIncident.emit(inc.workspace_id)"
                  title="Restablecer contadores OOM y liberar recursos del contenedor">
                  <svg lucideCheckCircle class="btn-icon"></svg>
                  <span>Resolver</span>
                </button>
              </div>
            </div>
          } @empty {
            <div class="empty-incidents">
              <span class="empty-icon">✓</span>
              <span class="empty-title">Sin incidencias técnicas</span>
              <span class="empty-text">Todos los contenedores operan dentro de los límites de memoria y cuotas del sistema.</span>
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
  styleUrl: './incidents-panel.component.scss',
})
export class IncidentsPanelComponent {
  incidents = input.required<TechnicalIncident[]>();
  resolveIncident = output<string>();
  restartWorkspace = output<string>();
  viewLogs = output<string>();
}
