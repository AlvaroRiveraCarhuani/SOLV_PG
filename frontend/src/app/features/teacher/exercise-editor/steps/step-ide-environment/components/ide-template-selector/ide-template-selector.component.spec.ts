import { ComponentFixture, TestBed } from '@angular/core/testing';
import { IdeTemplateSelectorComponent } from './ide-template-selector.component';
import { ApprovedTemplate } from '../../step-ide-environment.component';

describe('IdeTemplateSelectorComponent', () => {
  let component: IdeTemplateSelectorComponent;
  let fixture: ComponentFixture<IdeTemplateSelectorComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [IdeTemplateSelectorComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(IdeTemplateSelectorComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('muestra el stack completo con logos y etiquetas en chips, y deduplica base de datos si hay satélite activo', () => {
    const tpl: ApprovedTemplate = {
      id: '8ec592bf-4856-4059-b03f-e3b15efca849',
      name: 'Entorno Full-Stack JS',
      description: 'Node.js, Next.js y PostgreSQL para aplicaciones web',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        services: [
          { category: 'database', engine: 'postgres', role: 'primary_database' }
        ]
      }
    };

    // Stack chips: incluye Node.js y Next.js con sus nombres e iconos,
    // y excluye PostgreSQL de los chips genéricos porque ya tiene badge satélite dedicado
    const badges = component.getTechBadges(tpl);
    expect(badges.some(b => b.label === 'Node.js' && b.icon === 'node')).toBe(true);
    expect(badges.some(b => b.label === 'Next.js' && b.icon === 'nextjs')).toBe(true);
    expect(badges.some(b => b.icon === 'postgres')).toBe(false);

    expect(component.hasDatabaseService(tpl)).toBe(true);
    expect(component.databaseEngine(tpl)).toBe('PostgreSQL');
  });

  it('muestra todos los componentes del stack en entornos de Data Science', () => {
    const tpl: ApprovedTemplate = {
      id: 'b0000000-0000-0000-0000-000000000001',
      name: 'Python 3.12 Data Science & IA',
      description: 'Python con Pandas y NumPy para ciencia de datos',
      docker_image: 'gitpod/openvscode-server:1.96.0',
      base_ram_mb: 512,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved'
    };

    const badges = component.getTechBadges(tpl);
    expect(badges.some(b => b.label === 'Python' && b.icon === 'python')).toBe(true);
    expect(badges.some(b => b.label === 'Pandas')).toBe(true);
    expect(badges.some(b => b.label === 'NumPy')).toBe(true);
  });

  it('muestra el chip de runtime para entornos monomotores básicos', () => {
    const tpl: ApprovedTemplate = {
      id: 'b0000000-0000-0000-0000-000000000007',
      name: 'PHP 7.4 Legacy Monolith',
      description: 'Entorno base PHP',
      docker_image: 'php:7.4-apache',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved'
    };

    const badges = component.getTechBadges(tpl);
    expect(badges.length).toBe(1);
    expect(badges[0].label).toBe('PHP');
    expect(badges[0].icon).toBe('php');
  });

  it('detecta base de datos satélite activa tanto en services[] como en database.enabled', () => {
    const tplModernServices: ApprovedTemplate = {
      id: 'tpl-1',
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        services: [
          { category: 'database', engine: 'postgres', role: 'primary_database' }
        ]
      }
    };

    expect(component.hasDatabaseService(tplModernServices)).toBe(true);
    expect(component.databaseEngine(tplModernServices)).toBe('PostgreSQL');

    const tplHistorical: ApprovedTemplate = {
      id: 'tpl-2',
      name: 'SQL Lab',
      docker_image: 'postgres:16',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        database: {
          enabled: true,
          engine: 'MySQL'
        }
      }
    };

    expect(component.hasDatabaseService(tplHistorical)).toBe(true);
    expect(component.databaseEngine(tplHistorical)).toBe('MySQL');
  });

  it('emite templateSelected al seleccionar una opción del combobox', () => {
    let emitted = '';
    component.templateSelected.subscribe(id => (emitted = id));

    component.onTemplateOptionSelected({ id: 'tpl-1', label: 'Node', value: 'tpl-1' });
    expect(emitted).toBe('tpl-1');
  });
});
