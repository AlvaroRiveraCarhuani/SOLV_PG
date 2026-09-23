import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { TemplateCreateModalComponent } from './template-create-modal.component';
import { AdminTemplatesService, TemplateModelItem, CreateOfficialTemplateDTO } from '../../../services/admin-templates.service';

describe('TemplateCreateModalComponent Unit Tests', () => {
  let component: TemplateCreateModalComponent;

  const mockModel: TemplateModelItem = {
    id: 'm-python-algo',
    tenant_id: 'tenant-test',
    category_id: 'cat-comp',
    category_name: 'Competición & Algoritmos',
    name: 'Python 3 Juez Algorítmico',
    description: 'Entorno de prueba rápida para juez',
    docker_image: 'python:3.12-slim-bookworm',
    target_environment: 'JUEZ_EFIMERO',
    base_ram_mb: 256,
    entrypoint: 'python3 solution.py',
    timeout_ms: 2500,
    sample_input: '5 10',
    usage_count: 3,
    created_at: '2026-09-22T00:00:00Z',
    updated_at: '2026-09-22T00:00:00Z'
  };

  const mockTemplatesService = {
    getModels: () => of([]),
    getCategories: () => of([]),
    getTemplates: () => of([]),
    getLocalImages: () => of([]),
    getRuntimeCapabilities: () => of(null),
    getAvailableSatelliteServices: () => [],
    getDraft: () => of(null),
    saveDraft: () => of({} as any),
    deleteDraft: () => of(void 0),
    verifyImage: () => of({
      image_ref: 'python:3.12-slim-bookworm',
      is_local: false,
      exists: true,
      is_official: true,
      origin_type: 'official',
      architecture_compatible: true,
      host_arch: 'amd64',
      supported_platforms: ['linux/amd64'],
      size_bytes: 120000000,
      size_formatted: '120 MB',
      estimated_uncompressed_mb: 120,
      host_disk_free_gb: 50,
      host_disk_total_gb: 100,
      storage_status: 'OK',
      storage_message: '',
      cached: false,
      verified_at: new Date().toISOString()
    })
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        { provide: AdminTemplatesService, useValue: mockTemplatesService }
      ]
    });

    const fixture = TestBed.createComponent(TemplateCreateModalComponent);
    component = fixture.componentInstance;
  });

  it('debe iniciar con el paso purpose y estado en blanco', () => {
    expect(component.activeSection()).toBe('purpose');
    expect(component.creationMode()).toBe('blank');
    expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');
  });

  it('models debe ser una señal dinamica y no un array estatico hardcodeado', () => {
    expect(typeof component.models).toBe('function');
    expect(Array.isArray(component.models())).toBe(true);
    expect(component.models().length).toBe(0);
  });

  it('applyModel debe copiar todos los atributos tecnicos incluyendo categoria, modelo y sample_input', () => {
    component.applyModel(mockModel);

    expect(component.recipeUsed()).toBe('m-python-algo');
    expect(component.selectedModelId()).toBe('m-python-algo');
    expect(component.selectedCategoryId()).toBe('cat-comp');
    expect(component.name()).toBe('Python 3 Juez Algorítmico');
    expect(component.dockerImage()).toBe('python:3.12-slim-bookworm');
    expect(component.baseRamMB()).toBe(256);
    expect(component.entrypoint()).toBe('python3 solution.py');
    expect(component.timeoutMS()).toBe(2500);
    expect(component.sampleInput()).toBe('5 10');
    expect(component.activeSection()).toBe('image');
  });

  it('debe cambiar correctamente entre propositos actualizando la memoria base', () => {
    component.selectTargetEnvironment('JUEZ_EFIMERO');
    expect(component.targetEnvironment()).toBe('JUEZ_EFIMERO');
    expect(component.baseRamMB()).toBe(256);

    component.selectTargetEnvironment('IDE_PERSISTENTE');
    expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');
    expect(component.baseRamMB()).toBe(512);
  });

  it('confirmCreate debe emitir el payload con category_id, model_id y sample_input', () => {
    let emittedPayload: CreateOfficialTemplateDTO | null = null;
    component.created.subscribe((payload: CreateOfficialTemplateDTO) => {
      emittedPayload = payload;
    });

    component.applyModel(mockModel);
    component.targetEnvironment.set('JUEZ_EFIMERO');

    component.confirmCreate();

    expect(emittedPayload).not.toBeNull();
    expect(emittedPayload!.name).toBe('Python 3 Juez Algorítmico');
    expect(emittedPayload!.docker_image).toBe('python:3.12-slim-bookworm');
    expect(emittedPayload!.base_ram_mb).toBe(256);
    expect(emittedPayload!.target_environment).toBe('JUEZ_EFIMERO');
    expect(emittedPayload!.category_id).toBe('cat-comp');
    expect(emittedPayload!.model_id).toBe('m-python-algo');
    expect(emittedPayload!.sample_input).toBe('5 10');
    expect(emittedPayload!.entrypoint).toBe('python3 solution.py');
    expect(emittedPayload!.timeout_ms).toBe(2500);
  });

  it('matriz de reseteo de proposito: con formulario sucio exige confirmacion y al confirmar resetea recursos, satelites y entrypoint', () => {
    // 1. Ensuciar formulario con servicios y nombre
    component.name.set('Plantilla En Progreso');
    component.selectedServices.set([{ category: 'database', engine: 'postgres', version: '16' }]);
    expect(component.isFormDirty()).toBe(true);

    // 2. Intentar cambiar a JUEZ_EFIMERO con formulario dirty -> no aplica directo, activa modal
    component.selectTargetEnvironment('JUEZ_EFIMERO');
    expect(component.showPurposeConfirmDialog()).toBe(true);
    expect(component.pendingPurposeChange()).toBe('JUEZ_EFIMERO');
    expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');

    // 3. Confirmar cambio de propósito
    component.confirmPurposeChange();
    expect(component.showPurposeConfirmDialog()).toBe(false);
    expect(component.targetEnvironment()).toBe('JUEZ_EFIMERO');
    expect(component.baseRamMB()).toBe(256);
    // Para Juez, los servicios satélite se resetean a vacío
    expect(component.selectedServices().length).toBe(0);
    expect(component.entrypoint()).toBe('');
  });

  it('entrypoint obligatorio en Juez: bloquea guardado de borrador y publicacion si esta vacio', () => {
    component.targetEnvironment.set('JUEZ_EFIMERO');
    component.name.set('Juez Sin Entrypoint');
    component.dockerImage.set('python:3.12-slim-bookworm');
    component.baseRamMB.set(256);
    component.entrypoint.set(''); // Vacío
    component.isDraftSaved.set(true);

    // executionStatus debe marcar 'pending'
    expect(component.executionStatus()).toBe('pending');
    // canSaveDraft debe ser false
    expect(component.canSaveDraft()).toBe(false);
    // publishDisabledReason debe advertir sobre comando requerido
    expect(component.publishDisabledReason()).toContain('Comando de ejecución requerido para juez');
    expect(component.footerActionState()).toBe('publish_disabled');

    // Al definir el entrypoint pasa a estar completo y permite guardar borrador
    component.entrypoint.set('python3 solution.py');
    expect(component.executionStatus()).toBe('complete');
    expect(component.canSaveDraft()).toBe(true);
    expect(component.publishDisabledReason()).not.toContain('Comando de ejecución requerido para juez');
  });

  it('default de categoria desde modelo: al aplicar un modelo se precarga su category_id por defecto', () => {
    expect(component.selectedCategoryId()).toBeNull();

    component.applyModel(mockModel);

    // Debe heredar cat-comp directamente del modelo
    expect(component.selectedCategoryId()).toBe('cat-comp');
    expect(component.selectedModelId()).toBe('m-python-algo');
  });

  it('al abrir el modal con borrador existente en backend, el formulario se pre-carga', () => {
    const mockDraft = {
      id: 'draft-123',
      user_id: 'user-abc',
      form_data: {
        name: 'Plantilla desde Borrador',
        dockerImage: 'node:20-alpine',
        description: 'Borrador precargado',
        baseRamMB: 1024,
        targetEnvironment: 'IDE_PERSISTENTE'
      },
      updated_at: '2026-09-23T00:00:00Z'
    };

    const serviceWithDraft = {
      ...mockTemplatesService,
      getDraft: () => of(mockDraft)
    };

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [
        { provide: AdminTemplatesService, useValue: serviceWithDraft }
      ]
    });

    const fixture = TestBed.createComponent(TemplateCreateModalComponent);
    const comp = fixture.componentInstance;
    comp.ngOnInit();

    expect(comp.name()).toBe('Plantilla desde Borrador');
    expect(comp.dockerImage()).toBe('node:20-alpine');
    expect(comp.description()).toBe('Borrador precargado');
    expect(comp.baseRamMB()).toBe(1024);
    expect(comp.hasDraftToResume()).toBe(true);
  });
});

