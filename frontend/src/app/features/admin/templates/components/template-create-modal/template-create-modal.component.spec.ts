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
    createCategory: (dto: { name: string }) => of({ id: 'cat-new-1', name: dto.name, description: '', sort_order: 1, is_active: true, created_at: '', updated_at: '' }),
    getTemplates: () => of([]),
    getLocalImages: () => of({ images: [], usage_map: {} }),
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

  describe('Accesibilidad y Navegación por Teclado (ARIA APG)', () => {
    it('debe soportar roving tabindex en las tarjetas de propósito', () => {
      expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');
      component.selectTargetEnvironment('JUEZ_EFIMERO');
      expect(component.targetEnvironment()).toBe('JUEZ_EFIMERO');
    });

    it('las flechas deben alternar el propósito seleccionado (ambos pares horizontal y vertical)', () => {
      // 1. Horizontal: ArrowRight / ArrowLeft
      const eventRight = new KeyboardEvent('keydown', { key: 'ArrowRight' });
      component.onPurposeKeydown(eventRight, 'IDE_PERSISTENTE');
      expect(component.targetEnvironment()).toBe('JUEZ_EFIMERO');

      const eventLeft = new KeyboardEvent('keydown', { key: 'ArrowLeft' });
      component.onPurposeKeydown(eventLeft, 'JUEZ_EFIMERO');
      expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');

      // 2. Vertical: ArrowDown / ArrowUp
      const eventDown = new KeyboardEvent('keydown', { key: 'ArrowDown' });
      component.onPurposeKeydown(eventDown, 'IDE_PERSISTENTE');
      expect(component.targetEnvironment()).toBe('JUEZ_EFIMERO');

      const eventUp = new KeyboardEvent('keydown', { key: 'ArrowUp' });
      component.onPurposeKeydown(eventUp, 'JUEZ_EFIMERO');
      expect(component.targetEnvironment()).toBe('IDE_PERSISTENTE');
    });

    it('la tecla Enter en propósito debe avanzar a la siguiente sección (identity)', () => {
      expect(component.activeSection()).toBe('purpose');
      const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' });
      component.onPurposeKeydown(enterEvent, 'IDE_PERSISTENTE');
      expect(component.activeSection()).toBe('identity');
    });

    it('la tecla Escape debe cerrar drawer, popover o combobox antes de cerrar el modal', () => {
      let closedEmitted = false;
      component.closed.subscribe(() => { closedEmitted = true; });

      // 1. Con drawer abierto, Escape cierra el drawer
      component.isHelpDrawerOpen.set(true);
      component.onEscape();
      expect(component.isHelpDrawerOpen()).toBe(false);
      expect(closedEmitted).toBe(false);

      // 2. Con popover abierto, Escape cierra el popover
      component.showImagePopover.set(true);
      component.onEscape();
      expect(component.showImagePopover()).toBe(false);
      expect(closedEmitted).toBe(false);

      // 3. Con combobox abierto, Escape cierra el combobox
      component.isDropdownOpen.set(true);
      component.onEscape();
      expect(component.isDropdownOpen()).toBe(false);
      expect(closedEmitted).toBe(false);

      // 4. Con creación inline de categoría abierta, Escape cancela
      component.isCreatingCategoryInline.set(true);
      component.onEscape();
      expect(component.isCreatingCategoryInline()).toBe(false);
      expect(closedEmitted).toBe(false);

      // 5. Sin elementos secundarios abiertos, Escape cierra el modal
      component.onEscape();
      expect(closedEmitted).toBe(true);
    });
  });

  describe('Combobox de Imágenes con Grupos, Badges Separados y Conteo Real de Uso', () => {
    it('debe ordenar el grupo local por usage_count descendente y luego size_mb ascendente', () => {
      component.localImages.set([
        { repo_tag: 'nginx:alpine', size_mb: 90, is_official: true, has_latest_tag: false, created_at: '', usage_count: 1 },
        { repo_tag: 'python:3.12-custom', size_mb: 250, is_official: false, has_latest_tag: false, created_at: '', usage_count: 5 },
        { repo_tag: 'node:20-alpine', size_mb: 180, is_official: true, has_latest_tag: false, created_at: '', usage_count: 5 }
      ]);
      component.dockerImage.set('');

      const localGroup = component.groupLocalImages();
      expect(localGroup.length).toBe(3);
      expect(localGroup[0].repoTag).toBe('node:20-alpine');
      expect(localGroup[0].usageCount).toBe(5);
      expect(localGroup[1].repoTag).toBe('python:3.12-custom');
      expect(localGroup[1].usageCount).toBe(5);
      expect(localGroup[2].repoTag).toBe('nginx:alpine');
      expect(localGroup[2].usageCount).toBe(1);
    });

    it('debe enriquecer el grupo curado con usage_map y ordenarlo por uso descendente', () => {
      component.usageMap.set({
        'python:3.12-slim-bookworm': 10,
        'golang:1.22-bookworm': 3
      });
      component.dockerImage.set('');

      const curatedGroup = component.groupCuratedImages();
      expect(curatedGroup.length).toBeGreaterThan(0);
      expect(curatedGroup[0].repoTag).toBe('python:3.12-slim-bookworm');
      expect(curatedGroup[0].usageCount).toBe(10);
    });

    it('la búsqueda en el combobox debe filtrar por subcadena en ambos grupos', () => {
      component.localImages.set([
        { repo_tag: 'gcc:13.2-custom', size_mb: 150, is_official: false, has_latest_tag: false, created_at: '', usage_count: 0 }
      ]);
      component.dockerImage.set('gcc');

      const local = component.groupLocalImages();
      const curated = component.groupCuratedImages();

      expect(local.every(i => i.repoTag.toLowerCase().includes('gcc'))).toBe(true);
      expect(curated.every(i => i.repoTag.toLowerCase().includes('gcc'))).toBe(true);
    });

    it('onComboboxKeydown debe navegar cíclicamente y seleccionar con Enter', () => {
      component.localImages.set([
        { repo_tag: 'python:3.12-local', size_mb: 100, is_official: true, has_latest_tag: false, created_at: '' }
      ]);
      component.openCombobox();
      expect(component.isDropdownOpen()).toBe(true);

      const downEvent = new KeyboardEvent('keydown', { key: 'ArrowDown' });
      component.onComboboxKeydown(downEvent);
      expect(component.activeComboboxIndex()).toBe(0);

      const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' });
      component.onComboboxKeydown(enterEvent);
      expect(component.dockerImage()).toBe('python:3.12-local');
      expect(component.isDropdownOpen()).toBe(false);
    });

    it('chips sugeridos de herramientas deben variar según el propósito', () => {
      component.targetEnvironment.set('IDE_PERSISTENTE');
      expect(component.suggestedToolChips()).toEqual(['python3', 'node', 'gcc']);

      component.targetEnvironment.set('JUEZ_EFIMERO');
      expect(component.suggestedToolChips()).toEqual(['gcc', 'python3', 'javac']);

      component.toolsDeclared.set('');
      component.addToolDeclared('gcc');
      expect(component.toolsDeclared()).toBe('gcc');

      component.addToolDeclared('gcc');
      expect(component.toolsDeclared()).toBe('gcc');
    });
  });

  describe('Disclosure Progresivo de Categoría', () => {
    it('isCreatingCategoryInline debe estar en false por defecto', () => {
      expect(component.isCreatingCategoryInline()).toBe(false);
    });

    it('seleccionar __new__ debe abrir la fila inline y guardar la categoría previa', () => {
      component.selectedCategoryId.set('cat-existente');
      component.onCategorySelect('__new__');

      expect(component.isCreatingCategoryInline()).toBe(true);
      expect(component.previousCategoryId()).toBe('cat-existente');
      expect(component.inlineCategoryName()).toBe('');
    });

    it('cancelInlineCategory debe restaurar la categoría previa y ocultar la fila inline', () => {
      component.selectedCategoryId.set('cat-existente');
      component.onCategorySelect('__new__');
      component.inlineCategoryName.set('Nombre Descartado');

      component.cancelInlineCategory();

      expect(component.isCreatingCategoryInline()).toBe(false);
      expect(component.selectedCategoryId()).toBe('cat-existente');
      expect(component.inlineCategoryName()).toBe('');
    });

    it('createCategoryInline debe crear la categoría, seleccionarla y colapsar la fila inline', () => {
      component.onCategorySelect('__new__');
      component.inlineCategoryName.set('Inteligencia Artificial');

      component.createCategoryInline();

      expect(component.isCreatingCategoryInline()).toBe(false);
      expect(component.selectedCategoryId()).toBe('cat-new-1');
      expect(component.categories().some(c => c.name === 'Inteligencia Artificial')).toBe(true);
    });
  });

  describe('Drawer Contextual por Paso y Enlace al Manual', () => {
    it('toggleHelpDrawer debe alternar la visibilidad y establecer el paso activo', () => {
      expect(component.isHelpDrawerOpen()).toBe(false);

      component.toggleHelpDrawer('purpose');
      expect(component.isHelpDrawerOpen()).toBe(true);
      expect(component.activeDrawerStep()).toBe('purpose');

      component.toggleHelpDrawer('purpose');
      expect(component.isHelpDrawerOpen()).toBe(false);

      component.toggleHelpDrawer('image');
      expect(component.isHelpDrawerOpen()).toBe(true);
      expect(component.activeDrawerStep()).toBe('image');
    });

    it('closeHelpDrawer debe cerrar el drawer', () => {
      component.isHelpDrawerOpen.set(true);
      component.closeHelpDrawer();
      expect(component.isHelpDrawerOpen()).toBe(false);
    });
  });

  describe('Regla stale de Smoke Test e Invalidación', () => {
    const mockSuccessJob: any = {
      id: 'job-ok-1',
      image: 'python:3.12-slim-bookworm',
      tools: ['python3'],
      status: 'success',
      progress: { percent: 100 },
      result: { tools: [{ name: 'python3', present: true }], duration_ms: 100 }
    };

    it('debe marcar isEnvTestStale como true si cambia la imagen, RAM, herramientas, script o entrypoint tras la prueba', () => {
      component.dockerImage.set('python:3.12-slim-bookworm');
      component.baseRamMB.set(1024);
      component.toolsDeclared.set('python3, pip');
      component.setupScript.set('echo "ready"');
      component.entrypoint.set('python3 main.py');
      component.onEnvTestCompleted(mockSuccessJob);

      expect(component.isEnvTestStale()).toBe(false);

      // 1. Cambiar imagen invalida la prueba
      component.dockerImage.set('golang:1.22-alpine');
      expect(component.isEnvTestStale()).toBe(true);
      expect(component.footerActionState()).toBe('publish_disabled');
      expect(component.publishDisabledReason()).toContain('Prueba obsoleta');

      // Restaurar imagen
      component.dockerImage.set('python:3.12-slim-bookworm');
      expect(component.isEnvTestStale()).toBe(false);

      // 2. Cambiar RAM invalida
      component.baseRamMB.set(2048);
      expect(component.isEnvTestStale()).toBe(true);
      component.baseRamMB.set(1024);
      expect(component.isEnvTestStale()).toBe(false);

      // 3. Cambiar herramientas invalida
      component.toolsDeclared.set('python3, pip, pytest');
      expect(component.isEnvTestStale()).toBe(true);
      component.toolsDeclared.set('python3, pip');
      expect(component.isEnvTestStale()).toBe(false);

      // 4. Cambiar script de preparación invalida
      component.setupScript.set('pip install -r req.txt');
      expect(component.isEnvTestStale()).toBe(true);
      component.setupScript.set('echo "ready"');
      expect(component.isEnvTestStale()).toBe(false);

      // 5. Cambiar entrypoint invalida
      component.entrypoint.set('python3 test.py');
      expect(component.isEnvTestStale()).toBe(true);
      component.entrypoint.set('python3 main.py');
      expect(component.isEnvTestStale()).toBe(false);
    });
  });

  describe('Autoguardado silencioso y Footer contextual', () => {
    it('debe actualizar autosaveStatus y lastSavedTime tras guardar borrador', () => {
      component.name.set('Plantilla Test');
      component.dockerImage.set('python:3.12-slim-bookworm');
      component.saveDraft();

      expect(component.autosaveStatus()).toBe('saved');
      expect(component.lastSavedTime()).toBeTruthy();
      expect(component.autosaveIndicator()).toContain('Borrador guardado');
    });

    it('openPublishDialog debe guardar borrador implícitamente antes de abrir diálogo', () => {
      let saved = false;
      const originalSave = (component as any).saveDraftToStorage.bind(component);
      (component as any).saveDraftToStorage = () => {
        saved = true;
        originalSave();
      };

      component.isDraftSaved.set(true);
      component.activeEnvTestJob.set({ id: 'j1', status: 'success' } as any);
      component.dockerImage.set('python:3.12-slim-bookworm');
      component.lastTestedImage.set('python:3.12-slim-bookworm');
      component.lastTestedRam.set(component.baseRamMB());

      component.openPublishDialog();
      expect(saved).toBe(true);
      expect(component.showPublishDialog()).toBe(true);
    });
  });
});


