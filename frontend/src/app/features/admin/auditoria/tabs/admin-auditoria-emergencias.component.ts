import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

/**
 * Pestaña Emergencias (ADR-032).
 * Stub de andamiaje: el centro de control con doble confirmación llega en F4.
 */
@Component({
  selector: 'admin-auditoria-emergencias',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="stub-panel">
      <h2>Centro de Control de Emergencias</h2>
      <p>Acciones operativas clasificadas por impacto con doble confirmación tipada.</p>
    </div>
  `,
  styleUrl: './admin-auditoria-emergencias.component.scss'
})
export class AdminAuditoriaEmergenciasComponent {}
