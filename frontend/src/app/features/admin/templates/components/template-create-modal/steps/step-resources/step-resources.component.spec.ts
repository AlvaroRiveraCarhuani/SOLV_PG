import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvStepResourcesComponent } from './step-resources.component';
import { AvailableSatelliteService } from '../../../../../services/admin-templates.service';

describe('SolvStepResourcesComponent', () => {
  let component: SolvStepResourcesComponent;
  let fixture: ComponentFixture<SolvStepResourcesComponent>;

  const mockServices: AvailableSatelliteService[] = [
    {
      category: 'database',
      engine: 'postgres',
      label: 'PostgreSQL',
      version: '16',
      description: 'PostgreSQL aislada',
      envVar: 'DATABASE_URL',
      isAvailable: true
    },
    {
      category: 'database',
      engine: 'mysql',
      label: 'MySQL',
      version: '8.4',
      description: 'MySQL aislada',
      envVar: 'DATABASE_URL',
      isAvailable: true
    },
    {
      category: 'database',
      engine: 'mongodb',
      label: 'MongoDB',
      version: '7.0',
      description: 'MongoDB NoSQL',
      envVar: 'MONGODB_URI',
      isAvailable: false
    },
    {
      category: 'cache',
      engine: 'redis',
      label: 'Redis',
      version: '7.2',
      description: 'Redis en memoria',
      envVar: 'REDIS_URL',
      isAvailable: true
    }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvStepResourcesComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepResourcesComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.componentRef.setInput('baseRamMB', 1024);
    fixture.componentRef.setInput('availableServices', mockServices);
    fixture.detectChanges();
  });

  it('debe crearse correctamente con presets de IDE y memoria inicial', () => {
    expect(component).toBeTruthy();
    const activePreset = fixture.nativeElement.querySelector('.btn-ram-preset.active');
    expect(activePreset).toBeTruthy();
    expect(activePreset.textContent).toContain('1 GB');
  });

  it('debe emitir baseRamMBChange al seleccionar otro preset de RAM', () => {
    let emitted = 0;
    component.baseRamMBChange.subscribe(v => emitted = v);

    component.setRam(2048);
    expect(emitted).toBe(2048);
  });

  it('debe mostrar bases de datos adicionales diferenciando disponibles de no disponibles', () => {
    const cards = fixture.nativeElement.querySelectorAll('.service-card');
    expect(cards.length).toBe(4);

    // Postgres disponible
    const postgresCard = cards[0];
    expect(postgresCard.classList.contains('disabled')).toBe(false);
    expect(postgresCard.querySelector('.badge-optional')).toBeTruthy();

    // MongoDB no disponible
    const mongoCard = cards[2];
    expect(mongoCard.classList.contains('disabled')).toBe(true);
    const unavailableBadge = mongoCard.querySelector('.badge-unavailable');
    expect(unavailableBadge).toBeTruthy();
    expect(unavailableBadge.textContent).toContain('No disponible');
    expect(mongoCard.getAttribute('title')).toContain('No disponible');
  });

  it('no debe permitir seleccionar un servicio no disponible al hacer clic', () => {
    let emittedServices: any[] | null = null;
    component.selectedServicesChange.subscribe(s => emittedServices = s);

    component.toggleService(mockServices[2]); // MongoDB isAvailable: false
    expect(emittedServices).toBeNull();
  });

  it('debe permitir alternar la selección de un servicio disponible', () => {
    let emittedServices: any[] | null = null;
    component.selectedServicesChange.subscribe(s => emittedServices = s);

    component.toggleService(mockServices[0]); // PostgreSQL isAvailable: true
    expect(emittedServices).toBeTruthy();
    expect(emittedServices!.length).toBe(1);
    expect(emittedServices![0].engine).toBe('postgres');
  });

  it('selección única relacional: elegir MySQL deselecciona PostgreSQL manteniendo servicios no relacionales', () => {
    // Estado inicial: PostgreSQL y Redis seleccionados
    fixture.componentRef.setInput('selectedServices', [
      { category: 'database', engine: 'postgres', version: '16' },
      { category: 'cache', engine: 'redis', version: '7.2' }
    ]);
    fixture.detectChanges();

    let emittedServices: any[] = [];
    component.selectedServicesChange.subscribe(s => emittedServices = s);

    // Usuario hace clic en MySQL (mockServices[1])
    component.toggleService(mockServices[1]);

    expect(emittedServices.length).toBe(2);
    const engines = emittedServices.map(s => s.engine);
    expect(engines).toContain('mysql');
    expect(engines).toContain('redis');
    expect(engines).not.toContain('postgres');
  });

  it('spec de barra con satélites: los segmentos suman el base declarado', () => {
    const declaredBaseRam = 1024;
    fixture.componentRef.setInput('baseRamMB', declaredBaseRam);
    fixture.componentRef.setInput('runtimeCapabilities', {
      editor_base_mb: 210,
      host_memory: { total_ram_mb: 8192, available_ram_mb: 4096 }
    } as any);
    fixture.componentRef.setInput('selectedServices', [
      { category: 'database', engine: 'postgres', version: '16' }
    ]);
    fixture.detectChanges();

    const profile = component.resourceProfilePreview();
    const editorSegment = profile.editorBase; // 210
    const satSegments = profile.satelliteSegments; // [ { engine: 'postgres', baseMB: 128 } ]
    const satTotal = satSegments.reduce((sum, s) => sum + s.baseMB, 0); // 128
    const usableSegment = profile.usable; // 1024 - 210 - 128 = 686

    expect(satSegments.length).toBe(1);
    expect(satSegments[0].baseMB).toBe(128);
    expect(editorSegment + satTotal + usableSegment).toBe(declaredBaseRam);

    // Verificación en el DOM de la barra
    const barSegments = fixture.nativeElement.querySelectorAll('.memory-bar .bar-segment');
    expect(barSegments.length).toBe(3); // editor + postgres + libre
    expect(barSegments[1].classList.contains('bar-satellite')).toBe(true);
    expect(barSegments[1].textContent).toContain('PostgreSQL (128 MB)');
  });

  it('la capacidad de alumnos descuenta el consumo base de satélites seleccionados', () => {
    const availableHostRam = 4096;
    fixture.componentRef.setInput('runtimeCapabilities', {
      host_memory: { available_ram_mb: availableHostRam }
    } as any);
    fixture.componentRef.setInput('baseRamMB', 1024);

    // Sin satélites: 4096 / 1024 = 4 alumnos
    fixture.componentRef.setInput('selectedServices', []);
    fixture.detectChanges();
    const capacityWithoutSatellites = component.resourceProfilePreview().estimatedCapacity;
    expect(capacityWithoutSatellites).toBe(4);

    // Con PostgreSQL (+128 MB): total por alumno = 1024 + 128 = 1152 MB -> 4096 / 1152 = 3 alumnos
    fixture.componentRef.setInput('selectedServices', [
      { category: 'database', engine: 'postgres', version: '16' }
    ]);
    fixture.detectChanges();
    const capacityWithSatellites = component.resourceProfilePreview().estimatedCapacity;
    expect(capacityWithSatellites).toBe(3);
    expect(capacityWithSatellites).toBeLessThan(capacityWithoutSatellites);
  });

  it('el badge de piso indica el piso y la alerta muestra hecho, causa y acción', () => {
    fixture.componentRef.setInput('runtimeCapabilities', {
      editor_base_mb: 210,
      host_memory: { available_ram_mb: 4096 }
    } as any);
    fixture.componentRef.setInput('selectedServices', [
      { category: 'database', engine: 'postgres', version: '16' }
    ]);
    // Piso = 210 + 128 = 338 MB. Asignamos 256 MB (bajo el piso)
    fixture.componentRef.setInput('baseRamMB', 256);
    fixture.detectChanges();

    expect(component.minimumFloorMB()).toBe(338);
    expect(component.isRamBelowFloor()).toBe(true);

    const badgeFloor = fixture.nativeElement.querySelector('.badge-floor');
    expect(badgeFloor.textContent).toContain('338 MB');

    const alert = fixture.nativeElement.querySelector('.validation-message-alert.alert-danger');
    expect(alert).toBeTruthy();
    expect(alert.textContent).toContain('Hecho:');
    expect(alert.textContent).toContain('Causa:');
    expect(alert.textContent).toContain('Acción:');
  });

  it('debe emitir helpRequested al presionar el botón de ayuda contextual', () => {
    let emitted = false;
    component.helpRequested.subscribe(() => emitted = true);

    const helpBtn = fixture.nativeElement.querySelector('.btn-step-help');
    expect(helpBtn).toBeTruthy();
    helpBtn.click();

    expect(emitted).toBe(true);
  });

  it('debe detectar cuando la RAM excede el techo del host y activar bar-overflow', () => {
    fixture.componentRef.setInput('runtimeCapabilities', {
      max_allowed_ram_mb: 2048,
      host_memory: { total_ram_mb: 4096, available_ram_mb: 2048 }
    } as any);
    fixture.componentRef.setInput('baseRamMB', 4096);
    fixture.detectChanges();

    expect(component.isRamExceedingHost()).toBe(true);
    expect(component.ramExcessMB()).toBe(2048);

    const overflowBar = fixture.nativeElement.querySelector('.memory-bar.bar-overflow');
    expect(overflowBar).toBeTruthy();
    expect(overflowBar.textContent).toContain('Excede la capacidad del host');
  });

  it('debe usar pluralización correcta para 1 alumno y múltiples alumnos', () => {
    // 1 alumno
    fixture.componentRef.setInput('runtimeCapabilities', {
      host_memory: { available_ram_mb: 1024 }
    } as any);
    fixture.componentRef.setInput('baseRamMB', 1024);
    fixture.detectChanges();
    expect(component.capacityPluralLabel()).toContain('~1 alumno simultáneo');

    // Múltiples alumnos
    fixture.componentRef.setInput('runtimeCapabilities', {
      host_memory: { available_ram_mb: 4096 }
    } as any);
    fixture.detectChanges();
    expect(component.capacityPluralLabel()).toContain('alumnos simultáneos');
  });
});
