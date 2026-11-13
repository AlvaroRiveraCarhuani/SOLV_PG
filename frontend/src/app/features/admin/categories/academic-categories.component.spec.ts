import '@angular/localize/init';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { AcademicCategoriesComponent, ACADEMIC_CATEGORIES_HELP_KEY } from './academic-categories.component';
import { AdminTemplatesService, TemplateCategory } from '../services/admin-templates.service';
import { ContextHelpService } from '@shared/help/context-help.service';

describe('AcademicCategoriesComponent', () => {
  let component: AcademicCategoriesComponent;
  let fixture: ComponentFixture<AcademicCategoriesComponent>;
  let compiled: HTMLElement;
  let getCategoriesSpy: ReturnType<typeof vi.fn>;
  let reorderCategoriesSpy: ReturnType<typeof vi.fn>;
  let helpService: ContextHelpService;

  const mockCategories: TemplateCategory[] = [
    {
      id: 'c-1',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      name: 'Ciencia de Datos & IA',
      description: 'Modelos para análisis de datos',
      sort_order: 10,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    },
    {
      id: 'c-2',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      name: 'Desarrollo Web & Cloud',
      description: 'Modelos web',
      sort_order: 20,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    }
  ];

  beforeEach(async () => {
    getCategoriesSpy = vi.fn().mockReturnValue(of(mockCategories));
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
            getModels: vi.fn().mockReturnValue(of([])),
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
    expect(component.categories().length).toBe(2);
    expect(component.totalCategoriesCount()).toBe(2);
    expect(component.activeCategoriesCount()).toBe(2);
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
});
