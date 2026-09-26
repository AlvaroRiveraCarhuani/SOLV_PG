import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TemplatePromoteModalComponent } from './template-promote-modal.component';
import { AdminTemplatesService, AdminTemplateItem } from '../../../services/admin-templates.service';
import { of, throwError } from 'rxjs';

describe('TemplatePromoteModalComponent', () => {
  let component: TemplatePromoteModalComponent;
  let fixture: ComponentFixture<TemplatePromoteModalComponent>;
  let templatesServiceMock: {
    getCategories: ReturnType<typeof vi.fn>;
    promoteToModel: ReturnType<typeof vi.fn>;
  };

  const mockTemplate: AdminTemplateItem = {
    id: 'tpl-10',
    name: 'Go 1.23 API',
    docker_image: 'golang:1.23-alpine',
    description: 'Plantilla de backend Go',
    category_id: 'cat-1',
    base_ram_mb: 512,
    status: 'approved',
    created_at: '2026-09-01T00:00:00Z'
  };

  beforeEach(async () => {
    templatesServiceMock = {
      getCategories: vi.fn().mockReturnValue(of([{ id: 'cat-1', name: 'Backend', order_index: 0 }])),
      promoteToModel: vi.fn().mockReturnValue(of({ id: 'mod-1' }))
    };

    await TestBed.configureTestingModule({
      imports: [TemplatePromoteModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: templatesServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplatePromoteModalComponent);
    component = fixture.componentInstance;
    component.template = mockTemplate;
  });

  it('debe crearse e inicializar formulario con datos de la plantilla', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
    expect(component.modelName()).toBe('Go 1.23 API');
    expect(component.categoryId()).toBe('cat-1');
    expect(component.description()).toBe('Plantilla de backend Go');
    expect(component.categories().length).toBe(1);
  });

  it('debe emitir promoted al promover exitosamente', () => {
    fixture.detectChanges();
    const promotedSpy = vi.fn();
    component.promoted.subscribe(promotedSpy);

    component.confirm();

    expect(templatesServiceMock.promoteToModel).toHaveBeenCalledWith('tpl-10', {
      name: 'Go 1.23 API',
      category_id: 'cat-1',
      description: 'Plantilla de backend Go'
    });
    expect(promotedSpy).toHaveBeenCalledWith({
      id: 'tpl-10',
      name: 'Go 1.23 API',
      category_id: 'cat-1',
      description: 'Plantilla de backend Go'
    });
  });

  it('debe capturar error si promoteToModel falla', () => {
    templatesServiceMock.promoteToModel.mockReturnValue(throwError(() => ({ error: { message: 'El modelo ya existe' } })));
    fixture.detectChanges();

    component.confirm();

    expect(component.isPromoting()).toBe(false);
    expect(component.errorMsg()).toBe('El modelo ya existe');
  });

  it('debe emitir closed al cerrar', () => {
    fixture.detectChanges();
    const closedSpy = vi.fn();
    component.closed.subscribe(closedSpy);

    component.closeModal();
    expect(closedSpy).toHaveBeenCalled();
  });
});
