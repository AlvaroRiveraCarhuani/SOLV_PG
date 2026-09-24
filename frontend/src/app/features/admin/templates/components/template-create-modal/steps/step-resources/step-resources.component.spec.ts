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
      engine: 'mongodb',
      label: 'MongoDB',
      version: '7.0',
      description: 'MongoDB NoSQL',
      envVar: 'MONGODB_URI',
      isAvailable: false
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

  it('debe mostrar servicios satélite diferenciando disponibles de no disponibles', () => {
    const cards = fixture.nativeElement.querySelectorAll('.service-card');
    expect(cards.length).toBe(2);

    // Postgres disponible
    const postgresCard = cards[0];
    expect(postgresCard.classList.contains('disabled')).toBe(false);
    expect(postgresCard.querySelector('.badge-optional')).toBeTruthy();

    // MongoDB no disponible
    const mongoCard = cards[1];
    expect(mongoCard.classList.contains('disabled')).toBe(true);
    const unavailableBadge = mongoCard.querySelector('.badge-unavailable');
    expect(unavailableBadge).toBeTruthy();
    expect(unavailableBadge.textContent).toContain('No disponible');
    expect(mongoCard.getAttribute('title')).toContain('No disponible');
  });

  it('no debe permitir seleccionar un servicio no disponible al hacer clic', () => {
    let emittedServices: any[] | null = null;
    component.selectedServicesChange.subscribe(s => emittedServices = s);

    component.toggleService(mockServices[1]); // MongoDB isAvailable: false
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
    expect(overflowBar.textContent).toContain('Excede capacidad del host');
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
