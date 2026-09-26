import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ModelNewModalComponent } from './model-new-modal.component';
import { AdminTemplatesService, AdminTemplateItem, TemplateCategory } from '../../../../../services/admin-templates.service';
import { of, throwError } from 'rxjs';

describe('ModelNewModalComponent', () => {
  let component: ModelNewModalComponent;
  let fixture: ComponentFixture<ModelNewModalComponent>;
  let templatesServiceMock: {
    promoteToModel: ReturnType<typeof vi.fn>;
  };

  const mockApprovedTemplates: AdminTemplateItem[] = [
    {
      id: 'tpl-1',
      name: 'Python Data Science Base',
      docker_image: 'python:3.12-slim',
      description: 'Stack estándar con pandas y numpy',
      category_id: 'cat-1',
      base_ram_mb: 1024,
      status: 'approved',
      created_at: '2026-09-01T00:00:00Z'
    }
  ];

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
      promoteToModel: vi.fn().mockReturnValue(of({ id: 'm-new-1', name: 'Python Data Science Base' }))
    };

    await TestBed.configureTestingModule({
      imports: [ModelNewModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: templatesServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ModelNewModalComponent);
    component = fixture.componentInstance;
    component.approvedTemplates = mockApprovedTemplates;
    component.categories = mockCategories;
    fixture.detectChanges();
  });

  it('debe inicializarse correctamente y computar opciones de plantillas', () => {
    expect(component).toBeTruthy();
    expect(component.templateOptions().length).toBe(1);
    expect(component.templateOptions()[0].label).toBe('Python Data Science Base');
  });

  it('debe autocompletar campos al seleccionar una plantilla aprobada', () => {
    component.onTemplateSelected({ id: 'tpl-1', label: 'Python Data Science Base', value: 'tpl-1' });
    expect(component.selectedTemplateId()).toBe('tpl-1');
    expect(component.modelTitle()).toBe('Python Data Science Base');
    expect(component.modelCategoryId()).toBe('cat-1');
    expect(component.modelDescription()).toBe('Stack estándar con pandas y numpy');
  });

  it('debe mostrar error si se intenta confirmar sin plantilla seleccionada', () => {
    component.selectedTemplateId.set('');
    component.modelTitle.set('Test');
    component.onConfirm();

    expect(templatesServiceMock.promoteToModel).not.toHaveBeenCalled();
    expect(component.errorMessage()).toContain('plantilla aprobada');
  });

  it('debe emitir promoted cuando la promoción al servicio es exitosa', () => {
    const promotedSpy = vi.spyOn(component.promoted, 'emit');
    component.selectedTemplateId.set('tpl-1');
    component.modelTitle.set('Python Oficial');
    component.modelCategoryId.set('cat-1');
    component.modelDescription.set('Desc');

    component.onConfirm();

    expect(templatesServiceMock.promoteToModel).toHaveBeenCalledWith('tpl-1', {
      name: 'Python Oficial',
      category_id: 'cat-1',
      description: 'Desc'
    });
    expect(promotedSpy).toHaveBeenCalledWith({ id: 'm-new-1', name: 'Python Oficial' });
  });

  it('debe manejar error de promoción del servicio', () => {
    templatesServiceMock.promoteToModel.mockReturnValueOnce(
      throwError(() => ({ error: { message: 'Fallo al promover' } }))
    );

    component.selectedTemplateId.set('tpl-1');
    component.modelTitle.set('Python Oficial');
    component.onConfirm();

    expect(component.isSubmitting()).toBe(false);
    expect(component.errorMessage()).toBe('Fallo al promover');
  });
});
