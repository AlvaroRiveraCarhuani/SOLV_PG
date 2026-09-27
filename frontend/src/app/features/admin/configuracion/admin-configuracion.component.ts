import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantService } from '@core/services/tenant.service';
import { AdminConfigPeriodosComponent } from './tabs/admin-config-periodos.component';
import { AdminConfigIdentidadComponent } from './tabs/admin-config-identidad.component';
import { AdminConfigServidorComponent } from './tabs/admin-config-servidor.component';

export type ConfigTab = 'periodos' | 'identidad' | 'servidor';

interface ConfigTabDef {
  id: ConfigTab;
  label: string;
  description: string;
}

@Component({
  selector: 'admin-configuracion',
  standalone: true,
  imports: [
    CommonModule,
    AdminConfigPeriodosComponent,
    AdminConfigIdentidadComponent,
    AdminConfigServidorComponent
  ],
  templateUrl: './admin-configuracion.component.html',
  styleUrls: ['./admin-configuracion.component.scss']
})
export class AdminConfiguracionComponent {
  private readonly tenantService = inject(TenantService);

  readonly tenantName = computed<string>(
    () => this.tenantService.config()?.institution_name || 'la institución'
  );

  readonly tabs: ConfigTabDef[] = [
    { id: 'periodos', label: 'Períodos Académicos', description: 'Semestres, ciclos lectivos y archivado institucional' },
    { id: 'identidad', label: 'Identidad (White-Label)', description: 'Logo, nombre, color primario y tipografía de la universidad' },
    { id: 'servidor', label: 'Servidor y Respaldos', description: 'Políticas QoS, mantenimiento y estrategia de backups' }
  ];

  readonly activeTab = signal<ConfigTab>('periodos');

  readonly activeDef = computed<ConfigTabDef>(
    () => this.tabs.find((t) => t.id === this.activeTab()) ?? this.tabs[0]
  );

  selectTab(tab: ConfigTab): void {
    this.activeTab.set(tab);
  }
}
