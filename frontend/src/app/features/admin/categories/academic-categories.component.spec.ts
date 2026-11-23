import '@angular/localize/init';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { AcademicCategoriesComponent, ACADEMIC_CATEGORIES_HELP_KEY } from './academic-categories.component';
import { AdminTemplatesService, TemplateCategory, TemplateModelItem } from '../services/admin-templates.service';
import { ContextHelpService } from '@shared/help/context-help.service';

describe('AcademicCategoriesComponent', () => {
  let component: AcademicCategoriesComponent;
  let fixture: ComponentFixture<AcademicCategoriesComponent>;
  let compiled: HTMLElement;
  let getCategoriesSpy: ReturnType<typeof vi.fn>;
  let getModelsSpy: ReturnType<typeof vi.fn>;
  let reorderCategoriesSpy: ReturnType<typeof vi.fn>;
  let helpService: ContextHelpService;

  const mockCategories: TemplateCategory[] = [
    {
      id: 'c-1',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      name: 'Ciencia de Datos & IA',
      description: 'Entornos para análisis de datos',
      sort_order: 10,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    },
    {
      id: 'c-2',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      name: 'Desarrollo Web & Cloud',
      description: '',
      sort_order: 20,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    }
  ];

  const mockModels: TemplateModelItem[] = [
    {
      id: 'm-1',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      name: 'Python Data Science',
      category_id: 'c-1',
      description: 'Stack de ciencia de datos',
      docker_image: 'python:3.12-slim-bookworm',
      target_environment: 'IDE_PERSISTENTE',
      base_ram_mb: 1024,
      usage_count: 2,
      tools: ['python3', 'pip'],
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    } as TemplateModelItem
  ];

  beforeEach(async () => {
    getCategoriesSpy = vi.fn().mockReturnValue(of(mockCategories));
    getModelsSpy = vi.fn().mockReturnValue(of(mockModels));
    reorderCategoriesSpy = vi.fn().mockReturnValue(of(void 0));

    await TestBed.configureTestingModule({
      imports: [AcademicCategoriesComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: AdminTemplatesService,
          useValue: {
            getCategories: getCategoriesSpy,
            getModels: getModelsSpy,
            createCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
            updateCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
            deleteCategory: vi.fn().mockReturnValue(of(void 0)),
            reorderCategories: reorderCategoriesSpy
          }
        }
      ]
    }).compileComponents();

    helpService = TestBed.inject(ContextHelpService);
    fixture = TestBed.createComponent(AcademicCategoriesComponent);
    component = fixture.componentInstance;
    compiled = fixture.nativeElement;
    fixture.detectChanges();
  });

  it('debe inicializar el componente y cargar categorías', () => {
    expect(component).toBeTruthy();
    expect(getCategoriesSpy).toHaveBeenCalled();
    expect(getModelsSpy).toHaveBeenCalled();
    expect(component.categories().length).toBe(2);
    expect(component.totalCategoriesCount()).toBe(2);
    expect(component.activeCategoriesCount()).toBe(2);
    expect(compiled.querySelector('.categories-table')).toBeTruthy();
  });

  it('debe registrar y desregistrar la pantalla en el servicio de ayuda', () => {
    const regSpy = vi.spyOn(helpService, 'registerScreen');
    const unregSpy = vi.spyOn(helpService, 'unregisterScreen');

    component.ngOnInit();
    expect(regSpy).toHaveBeenCalledWith(ACADEMIC_CATEGORIES_HELP_KEY);

    component.ngOnDestroy();
    expect(unregSpy).toHaveBeenCalledWith(ACADEMIC_CATEGORIES_HELP_KEY);
  });

  it('debe abrir y cerrar el modal de creación de categoría', () => {
    component.openCreateCategory();
    expect(component.showCategoryModal()).toBe(true);
    expect(component.editingCategory()).toBeNull();

    component.closeCategoryModal();
    expect(component.showCategoryModal()).toBe(false);
  });

  it('debe abrir el modal en modo edición con la categoría seleccionada', () => {
    const cat = mockCategories[0];
    component.openEditCategory(cat);

    expect(component.showCategoryModal()).toBe(true);
    expect(component.editingCategory()).toEqual(cat);
  });

  it('debe reordenar categorías y llamar a reorderCategories', () => {
    component.moveCategoryDown(0);

    expect(reorderCategoriesSpy).toHaveBeenCalledWith([
      { id: 'c-2', sort_order: 10 },
      { id: 'c-1', sort_order: 20 }
    ]);
  });

  it('debe filtrar categorías por nombre o descripción', () => {
    component.onSearchChange('web');
    fixture.detectChanges();

    expect(component.isFiltered()).toBe(true);
    expect(component.filteredCategories().length).toBe(1);
    expect(component.filteredCategories()[0].name).toBe('Desarrollo Web & Cloud');
  });

  it('debe mostrar guion tenue si la categoría no tiene descripción', () => {
    const emptyDescCell = compiled.querySelectorAll('td.col-desc')[1];
    expect(emptyDescCell.textContent?.trim()).toBe('—');
  });

  it('debe deshabilitar el botón de eliminar si la categoría tiene entornos asociados', () => {
    expect(component.getCategoryModelCount('c-1')).toBe(1);
    const deleteButtons = compiled.querySelectorAll('.btn-delete');
    const firstDeleteBtn = deleteButtons[0] as HTMLButtonElement;
    expect(firstDeleteBtn.disabled).toBe(true);
    expect(firstDeleteBtn.getAttribute('title')).toContain('No se puede eliminar');
  });

  it('debe habilitar el botón de eliminar si la categoría tiene 0 entornos asociados', () => {
    expect(component.getCategoryModelCount('c-2')).toBe(0);
    const deleteButtons = compiled.querySelectorAll('.btn-delete');
    const secondDeleteBtn = deleteButtons[1] as HTMLButtonElement;
    expect(secondDeleteBtn.disabled).toBe(false);
  });

  it('debe abrir edición al hacer clic en la fila fuera de controles', () => {
    const editSpy = vi.spyOn(component, 'openEditCategory');
    const row = compiled.querySelector('tbody tr.category-row') as HTMLElement;
    const nameCell = row.querySelector('.col-name') as HTMLElement;
    nameCell.click();

    expect(editSpy).toHaveBeenCalledWith(mockCategories[0]);
  });

  it('debe abrir el drawer de entornos al hacer clic en el chip de entornos', () => {
    const chip = compiled.querySelector('.env-chip') as HTMLButtonElement;
    expect(chip).toBeTruthy();
    chip.click();
    fixture.detectChanges();

    expect(component.selectedCategoryForDrawer()).toEqual(mockCategories[0]);
    expect(component.categoryModelsForDrawer().length).toBe(1);
    expect(component.categoryModelsForDrawer()[0].name).toBe('Python Data Science');

    const drawerEl = compiled.querySelector('drawer');
    expect(drawerEl).toBeTruthy();
  });

  it('debe cerrar el drawer y abrir el modal de edición al seleccionar editar categoría desde el drawer', () => {
    component.openCategoryDrawer(mockCategories[0]);
    fixture.detectChanges();

    expect(component.selectedCategoryForDrawer()).toEqual(mockCategories[0]);

    component.editCategoryFromDrawer();
    fixture.detectChanges();

    expect(component.selectedCategoryForDrawer()).toBeNull();
    expect(component.showCategoryModal()).toBe(true);
    expect(component.editingCategory()).toEqual(mockCategories[0]);
  });
});
