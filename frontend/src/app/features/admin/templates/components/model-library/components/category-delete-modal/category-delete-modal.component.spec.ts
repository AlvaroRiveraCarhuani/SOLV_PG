import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CategoryDeleteModalComponent } from './category-delete-modal.component';
import { AdminTemplatesService, TemplateCategory } from '../../../../../services/admin-templates.service';
import { of, throwError } from 'rxjs';

describe('CategoryDeleteModalComponent', () => {
  let component: CategoryDeleteModalComponent;
  let fixture: ComponentFixture<CategoryDeleteModalComponent>;
  let templatesServiceMock: {
    deleteCategory: ReturnType<typeof vi.fn>;
  };

  const mockCategory: TemplateCategory = {
    id: 'c-1',
    name: 'Sistemas Distribuidos',
    sort_order: 1,
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z'
  };

  beforeEach(async () => {
    templatesServiceMock = {
      deleteCategory: vi.fn().mockReturnValue(of({}))
    };

    await TestBed.configureTestingModule({
      imports: [CategoryDeleteModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: templatesServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CategoryDeleteModalComponent);
    component = fixture.componentInstance;
    component.category = mockCategory;
    fixture.detectChanges();
  });

  it('debe inicializarse con el nombre de la categoría a eliminar', () => {
    expect(component).toBeTruthy();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Sistemas Distribuidos');
  });

  it('debe emitir deleted cuando el borrado en el servicio es exitoso', () => {
    const deletedSpy = vi.spyOn(component.deleted, 'emit');
    component.onConfirm();

    expect(templatesServiceMock.deleteCategory).toHaveBeenCalledWith('c-1');
    expect(deletedSpy).toHaveBeenCalledWith(mockCategory);
  });

  it('debe mostrar mensaje amigable si la categoría está en uso (status 409)', () => {
    templatesServiceMock.deleteCategory.mockReturnValueOnce(
      throwError(() => ({ status: 409 }))
    );

    component.onConfirm();

    expect(component.isSubmitting()).toBe(false);
    expect(component.errorMessage()).toContain('tiene modelos o plantillas asociadas');
  });
});
