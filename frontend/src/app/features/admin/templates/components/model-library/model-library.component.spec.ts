import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { provideRouter } from '@angular/router';
import { ModelLibraryComponent } from './model-library.component';
import { AdminTemplatesService, TemplateModelItem, TemplateCategory, AdminTemplateItem } from '../../../services/admin-templates.service';

describe('ModelLibraryComponent Spec', () => {
  let component: ModelLibraryComponent;
  let fixture: any;

  const mockModels: TemplateModelItem[] = [
    {
      id: 'm-1',
      name: 'Python 3 Ciencia de Datos',
      title: 'Python 3 Ciencia de Datos',
      description: 'Stack estándar con pandas, numpy y jupyter',
      category_id: 'c-1',
      category_name: 'Ciencia de Datos & IA',
      docker_image: 'python:3.12-slim-bookworm',
      target_environment: 'IDE_PERSISTENTE',
      base_ram_mb: 1024,
      usage_count: 5,
      is_active: true,
      sort_order: 1,
      source_template_id: 't-source-uuid-1234',
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    },
    {
      id: 'm-2',
      name: 'Node.js 20 Fullstack',
      title: 'Node.js 20 Fullstack',
      description: 'Entorno web con express y typescript',
      category_id: 'c-2',
      category_name: 'Desarrollo Web & Cloud',
      docker_image: 'node:20-bookworm-slim',
      target_environment: 'IDE_PERSISTENTE',
      base_ram_mb: 768,
      usage_count: 0,
      is_active: false,
      sort_order: 2,
      source_template_id: null,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    }
  ];

  const mockCategories: TemplateCategory[] = [
    {
      id: 'c-1',
      name: 'Ciencia de Datos & IA',
      description: 'Modelos para machine learning y analítica',
      sort_order: 1,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    },
    {
      id: 'c-2',
      name: 'Desarrollo Web & Cloud',
      description: 'Frameworks web modernos',
      sort_order: 2,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    }
  ];

  const mockApprovedTemplates: Partial<AdminTemplateItem>[] = [
    {
      id: 't-1',
      name: 'Plantilla Python Base',
      description: 'Plantilla aprobada de Python para entornos',
      docker_image: 'python:3.12-slim-bookworm',
      target_environment: 'IDE_PERSISTENTE',
      category_id: 'c-1',
      status: 'approved'
    }
  ];

  let getModelsSpy: any;
  let getCategoriesSpy: any;
  let deactivateModelSpy: any;
  let reorderCategoriesSpy: any;
  let deleteCategorySpy: any;
  let promoteToModelSpy: any;

  beforeEach(() => {
    getModelsSpy = vi.fn().mockReturnValue(of(mockModels));
    getCategoriesSpy = vi.fn().mockReturnValue(of(mockCategories));
    deactivateModelSpy = vi.fn().mockReturnValue(of(void 0));
    reorderCategoriesSpy = vi.fn().mockReturnValue(of(void 0));
    deleteCategorySpy = vi.fn().mockReturnValue(of(void 0));
    promoteToModelSpy = vi.fn().mockReturnValue(of(mockModels[0]));

    const mockService = {
      getModels: getModelsSpy,
      getCategories: getCategoriesSpy,
      deactivateModel: deactivateModelSpy,
      reactivateModel: vi.fn().mockReturnValue(of(void 0)),
      updateModel: vi.fn().mockReturnValue(of(mockModels[0])),
      createCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
      updateCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
      deleteCategory: deleteCategorySpy,
      reorderCategories: reorderCategoriesSpy,
      getTemplates: vi.fn().mockReturnValue(of(mockApprovedTemplates)),
      promoteToModel: promoteToModelSpy
    };

    TestBed.configureTestingModule({
      imports: [ModelLibraryComponent],
      providers: [
        provideRouter([]),
        { provide: AdminTemplatesService, useValue: mockService }
      ]
    });

    fixture = TestBed.createComponent(ModelLibraryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializar el componente y cargar modelos desde el servicio', () => {
    expect(component).toBeTruthy();
    expect(getModelsSpy).toHaveBeenCalledWith(undefined, undefined, true);
    expect(component.models().length).toBe(2);
  });

  it('debe calcular correctamente los KPIs del catálogo institucional', () => {
    expect(component.totalModelsCount()).toBe(2);
    expect(component.activeModelsCount()).toBe(1);
    expect(component.ideModelsCount()).toBe(2);
    expect(component.judgeModelsCount()).toBe(0);
    expect(component.categoriesCount()).toBe(2);
  });

  it('debe permitir alternar entre modo tabla y modo tarjetas', () => {
    expect(component.viewMode()).toBe('table');
    component.viewMode.set('cards');
    expect(component.viewMode()).toBe('cards');
  });

  it('debe mostrar el usage_count real en la grilla de modelos', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const usageBadges = compiled.querySelectorAll('.usage-badge');
    expect(usageBadges.length).toBe(2);

    // Primer modelo tiene 5 plantillas
    expect(usageBadges[0].textContent).toContain('5');
    expect(usageBadges[0].textContent).toContain('plantillas');

    // Segundo modelo tiene 0 plantillas
    expect(usageBadges[1].textContent).toContain('0');
  });

  it('debe calcular la cantidad de modelos asociados a cada categoría', () => {
    expect(component.getCategoryModelCount('c-1')).toBe(1);
    expect(component.getCategoryModelCount('c-2')).toBe(1);
    expect(component.getCategoryModelCount('c-inexistente')).toBe(0);
  });

  it('debe filtrar modelos por propósito cuando se modifica filterPurpose', () => {
    component.filterPurpose.set('IDE_PERSISTENTE');
    expect(component.filteredModels().length).toBe(2);

    component.filterPurpose.set('JUEZ_VIRTUAL');
    expect(component.filteredModels().length).toBe(0);
  });

  it('debe llamar a deactivateModel al alternar estado de un modelo activo', () => {
    component.toggleModelActive(mockModels[0]);
    expect(deactivateModelSpy).toHaveBeenCalledWith('m-1');
  });

  it('debe reordenar categorías y llamar a reorderCategories', () => {
    component.moveCategoryDown(0);
    expect(reorderCategoriesSpy).toHaveBeenCalled();
  });

  it('debe abrir el modal de nuevo modelo y promover plantilla seleccionada', () => {
    component.openNewModelModal();
    expect(component.showNewModelModal()).toBe(true);

    component.onSelectTemplateToPromote('t-1');
    expect(component.newModelTitle()).toBe('Plantilla Python Base');
    expect(component.newModelCategoryId()).toBe('c-1');

    component.confirmPromoteToModel();
    expect(promoteToModelSpy).toHaveBeenCalledWith('t-1', {
      name: 'Plantilla Python Base',
      category_id: 'c-1',
      description: 'Plantilla aprobada de Python para entornos'
    });
    expect(component.showNewModelModal()).toBe(false);
  });
});
