import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { StepIdentityComponent } from './step-identity.component';
import { TemplateCategory } from '../../../../../services/admin-templates.service';

describe('StepIdentityComponent', () => {
  let component: StepIdentityComponent;
  let fixture: ComponentFixture<StepIdentityComponent>;

  const mockCategories: TemplateCategory[] = [
    { id: 'cat-1', tenant_id: 't-1', name: 'Algoritmos y Estructuras', sort_order: 1, created_at: '', updated_at: '' },
    { id: 'cat-2', tenant_id: 't-1', name: 'Desarrollo Web', sort_order: 2, created_at: '', updated_at: '' }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StepIdentityComponent],
      providers: [provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(StepIdentityComponent);
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

  it('debe mostrar nombres de categoría reales en lugar de UUIDs en los encabezados de grupo', () => {
    const mockModels = [
      { id: 'm-1', category_id: 'cat-1', name: 'Python Básico', description: 'Intro', docker_image: 'python:3.12-slim', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 512, usage_count: 3, created_at: '' },
      { id: 'm-2', category_id: 'cat-2', name: 'Node.js Fullstack', description: 'Web', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 768, usage_count: 5, created_at: '' }
    ];

    fixture.componentRef.setInput('creationMode', 'recipe');
    fixture.componentRef.setInput('templateModels', mockModels);
    fixture.detectChanges();

    const badges = fixture.nativeElement.querySelectorAll('.discipline-badge');
    expect(badges.length).toBe(2);
    expect(badges[0].textContent.trim()).toBe('Algoritmos y Estructuras');
    expect(badges[1].textContent.trim()).toBe('Desarrollo Web');
    expect(fixture.nativeElement.textContent).not.toContain('cat-1');
  });

  it('debe filtrar combinando chips de categoría, búsqueda por texto y cota progresiva de 6', () => {
    const mockModels = [
      { id: 'm-1', category_id: 'cat-1', name: 'Python Básico', description: 'Intro a Python', docker_image: 'python:3.12-slim', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 512, usage_count: 3, created_at: '' },
      { id: 'm-2', category_id: 'cat-1', name: 'Python Avanzado', description: 'Django y FastAPI', docker_image: 'python:3.12-slim', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 1024, usage_count: 5, created_at: '' },
      { id: 'm-3', category_id: 'cat-1', name: 'Algoritmos C++', description: 'Estructuras', docker_image: 'gcc:13.2', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 512, usage_count: 1, created_at: '' },
      { id: 'm-4', category_id: 'cat-2', name: 'Node Express', description: 'Backend API', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 768, usage_count: 8, created_at: '' },
      { id: 'm-5', category_id: 'cat-2', name: 'Angular Web', description: 'Frontend SPA', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 768, usage_count: 2, created_at: '' },
      { id: 'm-6', category_id: 'cat-2', name: 'Vue Starter', description: 'Frontend Vite', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 512, usage_count: 4, created_at: '' },
      { id: 'm-7', category_id: 'cat-2', name: 'NestJS Microservices', description: 'Backend', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 1024, usage_count: 6, created_at: '' },
      { id: 'm-8', category_id: 'cat-2', name: 'NextJS SSR', description: 'Fullstack React', docker_image: 'node:20-alpine', target_environment: 'IDE_PERSISTENTE', base_ram_mb: 1024, usage_count: 7, created_at: '' }
    ];

    fixture.componentRef.setInput('creationMode', 'recipe');
    fixture.componentRef.setInput('templateModels', mockModels);
    fixture.detectChanges();

    // Cota inicial de 6
    expect(component.visibleModels().length).toBe(6);
    expect(component.hasMoreModels()).toBe(true);

    const cardsBefore = fixture.nativeElement.querySelectorAll('.model-card');
    expect(cardsBefore.length).toBe(6);

    // Botón "Mostrar más" expande de a 6
    const showMoreBtn = fixture.nativeElement.querySelector('.btn-show-more');
    expect(showMoreBtn).toBeTruthy();
    showMoreBtn.click();
    fixture.detectChanges();

    expect(component.visibleModels().length).toBe(8);
    expect(component.hasMoreModels()).toBe(false);

    // Filtro por chip de categoría "Desarrollo Web" (cat-2)
    component.selectCategoryFilter('cat-2');
    fixture.detectChanges();
    expect(component.filteredModels().length).toBe(5);

    // Combinación con búsqueda por texto
    component.onSearchChange('NestJS');
    fixture.detectChanges();
    expect(component.filteredModels().length).toBe(1);
    expect(component.filteredModels()[0].name).toBe('NestJS Microservices');
  });

  it('no debe ofrecer opción fantasma de categoría; vacío equivale a sin categoría', () => {
    const opts = component.categoryComboboxOptions();
    expect(opts.some(o => o.id === '__none__')).toBe(false);
    expect(opts.some(o => (o.label || '').includes('Sin categoría'))).toBe(false);
    expect(opts[opts.length - 1].id).toBe('__new__');

    fixture.componentRef.setInput('selectedCategoryId', null);
    fixture.detectChanges();
    expect(component.selectedCategoryLabel()).toBe('');
  });

  it('debe mostrar la línea informativa de descubribilidad de promoción con enlace al manual', () => {
    fixture.componentRef.setInput('creationMode', 'recipe');
    fixture.detectChanges();

    const banner = fixture.nativeElement.querySelector('.models-promotion-banner');
    expect(banner).toBeTruthy();
    expect(banner.textContent).toContain('Los modelos se originan a partir de plantillas aprobadas');

    const link = banner.querySelector('a.banner-link');
    expect(link).toBeTruthy();
    expect(link.getAttribute('href')).toBe('/admin/manual');
  });
});
