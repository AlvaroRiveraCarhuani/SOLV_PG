import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherTemplatesComponent, PublishedTemplate } from './teacher-templates.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { signal } from '@angular/core';
import { TeacherCourseSummary } from '../models/teacher.models';

describe('TeacherTemplatesComponent', () => {
  let component: TeacherTemplatesComponent;
  let fixture: ComponentFixture<TeacherTemplatesComponent>;
  let httpMock: HttpTestingController;
  let mockDashboardService: Partial<TeacherDashboardService>;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Estructuras de Datos',
      students_count: 20,
      active_now: 5,
      pending_review: 2,
      at_risk: 1
    }
  ];

  const mockTemplates: PublishedTemplate[] = [
    {
      id: 'tpl-1',
      name: 'Python Data Science Stack',
      description: 'Entorno completo con NumPy, Pandas y Jupyter.',
      docker_image: 'solv/python-datascience:v1.0.0',
      default_memory_mb: 1024,
      default_cpu_cores: 2,
      category: 'Programación',
      environment_type: 'IDE_PERSISTENTE',
      is_official: true,
      is_active: true
    },
    {
      id: 'tpl-2',
      name: 'PostgreSQL 16 Engine',
      description: 'Instancia SQL para laboratorios de bases de datos.',
      docker_image: 'solv/postgres:16-alpine',
      default_memory_mb: 512,
      default_cpu_cores: 1,
      category: 'Bases de Datos',
      environment_type: 'JUEZ_EFIMERO',
      is_official: false,
      is_active: true
    }
  ];

  beforeEach(async () => {
    mockDashboardService = {
      courses: signal<TeacherCourseSummary[]>(mockCourses)
    };

    await TestBed.configureTestingModule({
      imports: [TeacherTemplatesComponent, HttpClientTestingModule, RouterTestingModule],
      providers: [
        { provide: TeacherDashboardService, useValue: mockDashboardService }
      ]
    }).compileComponents();

    httpMock = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TeacherTemplatesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();

    const req = httpMock.expectOne('/api/v1/templates');
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockTemplates });
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should load templates on init and calculate KPIs correctly', () => {
    expect(component.templates().length).toBe(2);

    const stats = component.kpiStats();
    expect(stats.total).toBe(2);
    expect(stats.persistent).toBe(1);
    expect(stats.ephemeral).toBe(1);
    expect(stats.official).toBe(1);
  });

  it('should filter templates by category', () => {
    component.categoryFilter.set('Bases de Datos');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].name).toBe('PostgreSQL 16 Engine');
  });

  it('should filter templates by environment type', () => {
    component.envTypeFilter.set('IDE_PERSISTENTE');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].id).toBe('tpl-1');
  });

  it('should filter templates by search query', () => {
    component.searchTerm.set('datascience');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].id).toBe('tpl-1');
  });

  it('should open and close request modal', () => {
    component.openRequestModal();
    expect(component.isRequestModalOpen()).toBe(true);

    component.closeRequestModal();
    expect(component.isRequestModalOpen()).toBe(false);
  });

  it('should open and close exercise creation modal from template', () => {
    component.createLabFromTemplate(mockTemplates[0]);
    expect(component.isExerciseModalOpen()).toBe(true);
    expect(component.selectedTemplateForExercise()?.id).toBe('tpl-1');

    component.closeExerciseModal();
    expect(component.isExerciseModalOpen()).toBe(false);
    expect(component.selectedTemplateForExercise()).toBeNull();
  });
});
