import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantService } from '@core/services/tenant.service';
import { AdminAuditoriaRegistroComponent } from './tabs/admin-auditoria-registro.component';
import { AdminAuditoriaEmergenciasComponent } from './tabs/admin-auditoria-emergencias.component';

export type AuditoriaTab = 'registro' | 'emergencias';

interface AuditoriaTabDef {
  id: AuditoriaTab;
  label: string;
  description: string;
}

@Component({
  selector: 'admin-auditoria',
  standalone: true,
  imports: [CommonModule, AdminAuditoriaRegistroComponent, AdminAuditoriaEmergenciasComponent],
  templateUrl: './admin-auditoria.component.html',
  styleUrls: ['./admin-auditoria.component.scss']
})
export class AdminAuditoriaComponent {
  private readonly tenantService = inject(TenantService);

  readonly tenantName = computed<string>(
    () => this.tenantService.config()?.institution_name || 'la institución'
  );

  readonly tabs: AuditoriaTabDef[] = [
    { id: 'registro', label: 'Registro de Auditoría', description: 'Trazabilidad cronológica de eventos de la institución' },
    { id: 'emergencias', label: 'Emergencias', description: 'Centro de control operativo con doble confirmación y registro obligatorio' }
  ];

  readonly activeTab = signal<AuditoriaTab>('registro');

  readonly activeDef = computed<AuditoriaTabDef>(
    () => this.tabs.find((t) => t.id === this.activeTab()) ?? this.tabs[0]
  );

  selectTab(tab: AuditoriaTab): void {
    this.activeTab.set(tab);
  }
}
