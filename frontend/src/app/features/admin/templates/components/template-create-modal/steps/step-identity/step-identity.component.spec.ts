import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvStepIdentityComponent } from './step-identity.component';
import { TemplateCategory } from '../../../../../services/admin-templates.service';

describe('SolvStepIdentityComponent', () => {
  let component: SolvStepIdentityComponent;
  let fixture: ComponentFixture<SolvStepIdentityComponent>;

  const mockCategories: TemplateCategory[] = [
    { id: 'cat-1', tenant_id: 't-1', name: 'Algoritmos y Estructuras', sort_order: 1, created_at: '', updated_at: '' },
    { id: 'cat-2', tenant_id: 't-1', name: 'Desarrollo Web', sort_order: 2, created_at: '', updated_at: '' }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvStepIdentityComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepIdentityComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('categories', mockCategories);
    fixture.detectChanges();
  });

  it('no debe incluir enlaces externos como "Gestionar categorías"', () => {
    const text = fixture.nativeElement.textContent;
    expect(text).not.toContain('Gestionar categorías');
    const link = fixture.nativeElement.querySelector('.btn-manage-categories');
    expect(link).toBeNull();
  });

  it('debe emitir nameChange al modificar el input de nombre', () => {
    let emittedName = '';
    component.nameChange.subscribe((n) => (emittedName = n));

    const input = fixture.nativeElement.querySelector('#template-name-input');
    input.value = 'Nueva Plantilla Web';
    input.dispatchEvent(new Event('input'));

    expect(emittedName).toBe('Nueva Plantilla Web');
  });

  it('debe conmutar el modo de creación de plantilla al hacer clic en las puertas', () => {
    let mode = '';
    component.creationModeChange.subscribe((m) => (mode = m));

    const doorButtons = fixture.nativeElement.querySelectorAll('.door-chip');
    doorButtons[1].click(); // Desde modelo

    expect(mode).toBe('recipe');
  });

  it('debe permitir crear una categoría inline y emitir createCategory', () => {
    let newCategory = '';
    component.createCategory.subscribe((c) => (newCategory = c));

    // Simula selección de "__new__"
    component.onCategoryOptionSelected({
      id: '__new__',
      label: '＋ Crear nueva categoría...',
      value: '__new__'
    });
    fixture.detectChanges();

    expect(component.isCreatingCategoryInline()).toBe(true);

    component.inlineCategoryName.set('Bases de Datos Avanzadas');
    fixture.detectChanges();

    component.createCategoryInline();
    expect(newCategory).toBe('Bases de Datos Avanzadas');
    expect(component.isCreatingCategoryInline()).toBe(false);
  });
});
