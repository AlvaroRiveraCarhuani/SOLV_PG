import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

/**
 * Pestaña Registro de Auditoría (wireframe AUDIT_LOGS.md, ADR-027).
 * Stub de andamiaje: la tabla enriquecida y el drawer timeline llegan en F2.
 */
@Component({
  selector: 'admin-auditoria-registro',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="stub-panel">
      <h2>Registro de Auditoría y Trazabilidad</h2>
      <p>Tabla cronológica enriquecida con drawer de timeline por actor.</p>
    </div>
  `,
  styleUrl: './admin-auditoria-registro.component.scss'
})
export class AdminAuditoriaRegistroComponent {}
