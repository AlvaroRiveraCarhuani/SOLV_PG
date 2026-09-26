import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ModelEditModalComponent } from './model-edit-modal.component';
import { AdminTemplatesService, TemplateModelItem, TemplateCategory } from '../../../../../services/admin-templates.service';
import { of, throwError } from 'rxjs';

describe('ModelEditModalComponent', () => {
  let component: ModelEditModalComponent;
  let fixture: ComponentFixture<ModelEditModalComponent>;
  let templatesServiceMock: {
    updateModel: ReturnType<typeof vi.fn>;
  };

  const mockModel: TemplateModelItem = {
    id: 'm-1',
    name: 'Python Data Science Base',
    title: 'Python Data Science Base',
    description: 'Entorno pedagógico',
    category_id: 'cat-1',
    docker_image: 'python:3.12-slim',
    target_environment: 'IDE_PERSISTENTE',
    base_ram_mb: 1024,
    usage_count: 2,
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z'
  };

  const mockCategories: TemplateCategory[] = [
    {
      id: 'cat-1',
      name: 'Ciencia de Datos',
      sort_order: 1,
      created_at: '2026-09-01T00:00:00Z',
      updated_at: '2026-09-01T00:00:00Z'
    }
  ];

  beforeEach(async () => {
    templatesServiceMock = {
      updateModel: vi.fn().mockReturnValue(of({ ...mockModel, title: 'Nuevo Título' }))
    };

    await TestBed.configureTestingModule({
      imports: [ModelEditModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: templatesServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ModelEditModalComponent);
    component = fixture.componentInstance;
    component.model = mockModel;
    component.categories = mockCategories;
    fixture.detectChanges();
  });

  it('debe inicializarse con los datos del modelo provisto', () => {
    expect(component).toBeTruthy();
    expect(component.title()).toBe('Python Data Science Base');
    expect(component.categoryId()).toBe('cat-1');
    expect(component.description()).toBe('Entorno pedagógico');
  });

  it('debe validar que título y categoría sean obligatorios', () => {
    component.title.set('');
    component.onSave();
    expect(templatesServiceMock.updateModel).not.toHaveBeenCalled();
    expect(component.errorMessage()).toContain('título del modelo es obligatorio');

    component.title.set('Título Válido');
    component.categoryId.set('');
    component.onSave();
    expect(templatesServiceMock.updateModel).not.toHaveBeenCalled();
    expect(component.errorMessage()).toContain('categoría académica');
  });

  it('debe actualizar el modelo y emitir saved cuando los datos son válidos', () => {
    const savedSpy = vi.spyOn(component.saved, 'emit');
    component.title.set('Python Data Science Pro');
    component.categoryId.set('cat-1');
    component.description.set('Actualizado');

    component.onSave();

    expect(templatesServiceMock.updateModel).toHaveBeenCalledWith('m-1', {
      title: 'Python Data Science Pro',
      category_id: 'cat-1',
      description: 'Actualizado'
    });
    expect(savedSpy).toHaveBeenCalledWith({ id: 'm-1', title: 'Python Data Science Pro' });
  });

  it('debe gestionar errores al guardar', () => {
    templatesServiceMock.updateModel.mockReturnValueOnce(
      throwError(() => ({ message: 'Error de servidor' }))
    );

    component.title.set('Nuevo');
    component.categoryId.set('cat-1');
    component.onSave();

    expect(component.isSubmitting()).toBe(false);
    expect(component.errorMessage()).toBe('Error de servidor');
  });
});
