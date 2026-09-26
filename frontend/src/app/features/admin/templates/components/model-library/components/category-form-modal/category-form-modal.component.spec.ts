import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CategoryFormModalComponent } from './category-form-modal.component';
import { AdminTemplatesService, TemplateCategory } from '../../../../../services/admin-templates.service';
import { of, throwError } from 'rxjs';

describe('CategoryFormModalComponent', () => {
  let component: CategoryFormModalComponent;
  let fixture: ComponentFixture<CategoryFormModalComponent>;
  let templatesServiceMock: {
    createCategory: ReturnType<typeof vi.fn>;
    updateCategory: ReturnType<typeof vi.fn>;
  };

  const mockCategory: TemplateCategory = {
    id: 'c-1',
    name: 'Sistemas Distribuidos',
    description: 'Kubernetes y microservicios',
    sort_order: 1,
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z'
  };

  beforeEach(async () => {
    templatesServiceMock = {
      createCategory: vi.fn().mockReturnValue(of({ id: 'c-new', name: 'Nueva Cat' })),
      updateCategory: vi.fn().mockReturnValue(of({ ...mockCategory, name: 'Editada' }))
    };

    await TestBed.configureTestingModule({
      imports: [CategoryFormModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: templatesServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CategoryFormModalComponent);
    component = fixture.componentInstance;
  });

  it('debe crear una nueva categoría cuando category es null', () => {
    component.category = null;
    fixture.detectChanges();

    const savedSpy = vi.spyOn(component.saved, 'emit');
    component.name.set('Bases de Datos');
    component.description.set('Relacionales y NoSQL');
    component.onSave();

    expect(templatesServiceMock.createCategory).toHaveBeenCalledWith({
      name: 'Bases de Datos',
      description: 'Relacionales y NoSQL'
    });
    expect(savedSpy).toHaveBeenCalledWith({ id: 'c-new', name: 'Bases de Datos' });
  });

  it('debe actualizar la categoría existente cuando category está provista', () => {
    component.category = mockCategory;
    component.ngOnInit();
    fixture.detectChanges();

    expect(component.name()).toBe('Sistemas Distribuidos');
    expect(component.description()).toBe('Kubernetes y microservicios');

    const savedSpy = vi.spyOn(component.saved, 'emit');
    component.name.set('Sistemas Cloud');
    component.onSave();

    expect(templatesServiceMock.updateCategory).toHaveBeenCalledWith('c-1', {
      name: 'Sistemas Cloud',
      description: 'Kubernetes y microservicios'
    });
    expect(savedSpy).toHaveBeenCalledWith({ id: 'c-1', name: 'Sistemas Cloud' });
  });

  it('debe requerir el nombre de la categoría', () => {
    component.name.set('   ');
    component.onSave();
    expect(templatesServiceMock.createCategory).not.toHaveBeenCalled();
    expect(component.errorMessage()).toContain('nombre de la categoría es obligatorio');
  });

  it('debe capturar errores de creación del backend', () => {
    templatesServiceMock.createCategory.mockReturnValueOnce(
      throwError(() => ({ message: 'Nombre duplicado' }))
    );

    component.name.set('Redundante');
    component.onSave();

    expect(component.isSubmitting()).toBe(false);
    expect(component.errorMessage()).toBe('Nombre duplicado');
  });
});
