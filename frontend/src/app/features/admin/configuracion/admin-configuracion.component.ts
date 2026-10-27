import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantService } from '@core/services/tenant.service';
import { AdminConfigPeriodosComponent } from './tabs/admin-config-periodos.component';
import { AdminConfigIdentidadComponent } from './tabs/admin-config-identidad.component';
import { AdminConfigServidorComponent } from './tabs/admin-config-servidor.component';
import { AdminConfigIaComponent } from './tabs/admin-config-ia.component';
import { AdminConfigAdministradoresComponent } from './tabs/admin-config-administradores.component';

export type ConfigTab = 'periodos' | 'identidad' | 'servidor' | 'ia' | 'administradores';

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
    AdminConfigServidorComponent,
    AdminConfigIaComponent,
    AdminConfigAdministradoresComponent
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
    { id: 'servidor', label: 'Servidor y Respaldos', description: 'Políticas QoS, mantenimiento y estrategia de backups' },
    { id: 'ia', label: 'Inteligencia Artificial', description: 'Copiloto institucional para plantillas y creación de ejercicios del Juez' },
    { id: 'administradores', label: 'Administradores', description: 'Gestión de credenciales con privilegios elevados y operadores del sistema' }
  ];

  readonly activeTab = signal<ConfigTab>('periodos');

  readonly activeDef = computed<ConfigTabDef>(
    () => this.tabs.find((t) => t.id === this.activeTab()) ?? this.tabs[0]
  );

  selectTab(tab: ConfigTab): void {
    this.activeTab.set(tab);
  }
}
