import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AdminEnvironmentRequestsComponent } from './admin-environment-requests.component';
import { EnvironmentRequestService } from '@features/teacher/services/environment-request.service';
import { AdminTemplatesService } from '../services/admin-templates.service';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { EnvironmentRequest } from '@features/teacher/templates/environment-request.models';
import { AdminTemplateItem } from '../models/admin-templates.models';

describe('AdminEnvironmentRequestsComponent', () => {
  let component: AdminEnvironmentRequestsComponent;
  let fixture: ComponentFixture<AdminEnvironmentRequestsComponent>;
  let requestServiceMock: {
    listForReview: ReturnType<typeof vi.fn>;
    getAdminRequest: ReturnType<typeof vi.fn>;
    review: ReturnType<typeof vi.fn>;
  };
  let templatesServiceMock: {
    getTemplates: ReturnType<typeof vi.fn>;
  };
  let helpServiceMock: {
    registerScreen: ReturnType<typeof vi.fn>;
    registerGuides: ReturnType<typeof vi.fn>;
    unregisterGuides: ReturnType<typeof vi.fn>;
    unregisterScreen: ReturnType<typeof vi.fn>;
  };

  const mockRequests: EnvironmentRequest[] = [
    {
      id: 'req-1',
      tenant_id: 'ten-1',
      requested_by: 'user-1',
      requester_name: 'Dr. Alan Turing',
      title: 'Taller de Algoritmos Cuánticos',
      description: 'Prácticas de simulación con Qiskit.',
      teacher_text: 'Requerimos Python con librerías cuánticas.',
      uses: ['practice'],
      target_environment: 'IDE_PERSISTENTE',
      components: [
        { id: 'comp-1', slug: 'python', name: 'Python 3.12', layer: 'runtime', icon_key: 'python' }
      ],
      missing_components: [],
      subjects: [{ subject_name: 'Computación Cuántica', added_at: '2026-11-20T00:00:00Z' }],
      stack_signature: 'sig-1',
      status: 'EN_REVISION',
      review_note: '',
      created_at: '2026-11-20T10:00:00Z',
      updated_at: '2026-11-20T10:00:00Z'
    },
    {
      id: 'req-2',
      tenant_id: 'ten-1',
      requested_by: 'user-2',
      requester_name: 'Dra. Ada Lovelace',
      title: 'Laboratorio MATLAB y Simulación',
      description: 'Modelado matemático de fluidos.',
      teacher_text: 'Usaremos MATLAB exclusivamente.',
      uses: ['practice', 'grade'],
      target_environment: 'AMBOS',
      components: [],
      missing_components: [
        { name: 'MATLAB', layer: 'runtime', reason: 'Software con licencia no en catálogo' }
      ],
      subjects: [
        { subject_name: 'Física I', added_at: '2026-11-21T00:00:00Z' },
        { subject_name: 'Cálculo Avanzado', added_at: '2026-11-21T00:00:00Z' }
      ],
      stack_signature: 'sig-2',
      status: 'NECESITA_CAMBIO',
      review_note: 'Por favor justifique si puede utilizarse GNU Octave.',
      reviewed_at: '2026-11-22T08:30:00Z',
      created_at: '2026-11-21T11:00:00Z',
      updated_at: '2026-11-22T08:30:00Z'
    },
    {
      id: 'req-3',
      tenant_id: 'ten-1',
      requested_by: 'user-3',
      requester_name: 'Ing. Grace Hopper',
      title: 'Desarrollo Backend con Go',
      description: 'Microservicios con PostgreSQL.',
      teacher_text: '',
      uses: ['grade'],
      target_environment: 'JUEZ_EFIMERO',
      components: [
        { id: 'comp-2', slug: 'go', name: 'Go 1.26', layer: 'runtime', icon_key: 'go' },
        { id: 'comp-3', slug: 'postgres', name: 'PostgreSQL 18', layer: 'database', icon_key: 'postgres' }
      ],
      missing_components: [],
      subjects: [{ subject_name: 'Sistemas Distribuidos', added_at: '2026-11-22T00:00:00Z' }],
      stack_signature: 'sig-3',
      status: 'APROBADA',
      review_note: 'Aprobada con plantilla oficial.',
      resolved_template_id: 'tpl-go',
      resolved_template_name: 'Go Backend Base',
      reviewed_at: '2026-11-22T14:00:00Z',
      created_at: '2026-11-22T09:00:00Z',
      updated_at: '2026-11-22T14:00:00Z'
    },
    {
      id: 'req-4',
      tenant_id: 'ten-1',
      requested_by: 'user-4',
      requester_name: 'Prof. Donald Knuth',
      title: 'TeX y Tipografía',
      description: 'Edición de libros.',
      teacher_text: '',
      uses: ['practice'],
      target_environment: 'IDE_PERSISTENTE',
      components: [],
      missing_components: [],
      subjects: [],
      stack_signature: 'sig-4',
      status: 'NO_VIABLE',
      review_note: 'Servidor sin capacidad para procesamiento LaTeX masivo.',
      reviewed_at: '2026-11-22T16:00:00Z',
      created_at: '2026-11-22T10:00:00Z',
      updated_at: '2026-11-22T16:00:00Z'
    }
  ];

  const mockTemplates: AdminTemplateItem[] = [
    {
      id: 'tpl-1',
      name: 'Python Standard Lab',
      docker_image: 'python:3.12-slim',
      base_ram_mb: 512,
      status: 'approved',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-01T00:00:00Z'
    },
    {
      id: 'tpl-go',
      name: 'Go Backend Base',
      docker_image: 'golang:1.26',
      base_ram_mb: 512,
      status: 'approved',
      target_environment: 'AMBOS',
      created_at: '2026-09-01T00:00:00Z'
    }
  ];

  beforeEach(async () => {
    requestServiceMock = {
      listForReview: vi.fn().mockReturnValue(of(mockRequests)),
      // La ficha completa aporta la capacidad medida en el servidor actual.
      getAdminRequest: vi.fn().mockImplementation((id: string) =>
        of(mockRequests.find(r => r.id === id) ?? mockRequests[0])
      ),
      review: vi.fn().mockImplementation((id, payload) =>
        of({
          ...mockRequests.find(r => r.id === id)!,
          status: payload.status,
          review_note: payload.review_note,
          resolved_template_id: payload.resolved_template_id
        })
      )
    };

    templatesServiceMock = {
      getTemplates: vi.fn().mockReturnValue(of(mockTemplates))
    };

    helpServiceMock = {
      registerScreen: vi.fn(),
      registerGuides: vi.fn(),
      unregisterGuides: vi.fn(),
      unregisterScreen: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [AdminEnvironmentRequestsComponent],
      providers: [
        provideRouter([]),
        { provide: EnvironmentRequestService, useValue: requestServiceMock },
        { provide: AdminTemplatesService, useValue: templatesServiceMock },
        { provide: ContextHelpService, useValue: helpServiceMock },
        { provide: ProductTourService, useValue: { start: vi.fn() } },
        { provide: InteractiveTutorialService, useValue: { start: vi.fn() } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminEnvironmentRequestsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializar y renderizar las 5 pestañas en <status-tabs>', () => {
    const tabs = component.tabs();
    expect(tabs.length).toBe(5);
    expect(tabs.map(t => t.id)).toEqual(['pending', 'changes', 'approved', 'rejected', 'all']);
    expect(tabs.map(t => t.label)).toEqual([
      'En revisión',
      'Necesita cambio',
      'Aprobadas',
      'No viables',
      'Todas'
    ]);
    expect(tabs[0].count).toBe(1); // req-1
    expect(tabs[1].count).toBe(1); // req-2
    expect(tabs[2].count).toBe(1); // req-3
    expect(tabs[3].count).toBe(1); // req-4
    expect(tabs[4].count).toBe(4); // total
  });

  it('debe mostrar la columna ESTADO únicamente en la pestaña Todas', () => {
    // Por defecto está en 'pending'
    expect(component.activeTab()).toBe('pending');
    let compiled = fixture.nativeElement as HTMLElement;
    let headers = Array.from(compiled.querySelectorAll('.data-table th')).map(th => th.textContent?.trim());
    expect(headers).not.toContain('ESTADO');
    expect(headers).not.toContain('Estado');

    // Cambiar a 'all'
    component.onTabChange('all');
    fixture.detectChanges();

    compiled = fixture.nativeElement as HTMLElement;
    headers = Array.from(compiled.querySelectorAll('.data-table th')).map(th => th.textContent?.trim());
    expect(headers.some(h => h?.toUpperCase() === 'ESTADO')).toBe(true);
  });

  it('elimina USO DECLARADO de la tabla (no existe en el DOM)', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const textContent = compiled.textContent || '';
    expect(textContent).not.toContain('USO DECLARADO');
    expect(textContent).not.toContain('Uso declarado');
  });

  it('muestra directamente los chips faltantes sin el mensaje contradictorio cuando components está vacío pero missing_components tiene elementos', () => {
    component.onTabChange('changes'); // req-2 tiene missing_components: ['MATLAB'] y components: []
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const textContent = compiled.textContent || '';
    expect(textContent).toContain('MATLAB');
    expect(textContent).not.toContain('Sin piezas: el docente pidió solo con texto');
    expect(textContent).not.toContain('Sin tecnologías especificadas');
  });

  it('muestra Sin tecnologías especificadas únicamente si ambos arrays están vacíos', () => {
    component.onTabChange('rejected'); // req-4 tiene ambos vacíos
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const textContent = compiled.textContent || '';
    expect(textContent).toContain('Sin tecnologías especificadas');
  });

  it('abre el Drawer en EN_REVISION y permite responder', () => {
    const pendingReq = mockRequests[0];
    component.openDrawer(pendingReq);
    expect(component.isDrawerOpen()).toBe(true);
    expect(component.selected()?.id).toBe('req-1');
    expect(component.canReview(pendingReq)).toBe(true);
    expect(component.actionButtonLabel(pendingReq)).toBe('Responder');
  });

  it('permite actualizar la resolución en NECESITA_CAMBIO y muestra la nota anterior', () => {
    const changesReq = mockRequests[1];
    component.openDrawer(changesReq);
    expect(component.isDrawerOpen()).toBe(true);
    expect(component.canReview(changesReq)).toBe(true);
    expect(component.actionButtonLabel(changesReq)).toBe('Actualizar');

    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Ajuste solicitado anteriormente');
    expect(compiled.textContent).toContain('Por favor justifique si puede utilizarse GNU Octave');
  });

  it('traduce la capa de las tecnologías reconocidas y no repite el nombre del docente en el subtítulo', () => {
    component.openDrawer(mockRequests[0]);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const header = compiled.querySelector('.drawer-header');
    expect(header?.querySelector('.drawer-subtitle')).toBeNull();
    expect(header?.textContent).not.toContain('Dr. Alan Turing');

    const teacher = compiled.querySelector('.drawer-header-meta .meta-teacher')?.textContent?.trim();
    expect(teacher).toBe('Dr. Alan Turing');

    expect(component.layerLabel('runtime')).toBe('Lenguaje principal');
    expect(component.layerLabel('web_platform')).toBe('Interfaz web');
    expect(component.layerLabel('addon')).toBe('Complemento');
    expect(component.layerLabel('database')).toBe('Base de datos');

    const layers = Array.from(compiled.querySelectorAll('.tech-item-layer')).map(el =>
      el.textContent?.trim()
    );
    expect(layers).toContain('Lenguaje principal');
    expect(layers).not.toContain('runtime');
  });

  it('enlaza al Constructor con el identificador de la solicitud en la misma pestaña', () => {
    component.openDrawer(mockRequests[0]);
    fixture.detectChanges();

    const link = (fixture.nativeElement as HTMLElement).querySelector('.constructor-link');
    expect(link?.getAttribute('href')).toContain('/admin/plantillas/nueva');
    expect(link?.getAttribute('href')).toContain('requestId=req-1');
    expect(link?.getAttribute('target')).toBeNull();
  });

  it('omite la nota adicional cuando el docente no escribió nada', () => {
    component.openDrawer(mockRequests[2]); // teacher_text vacío
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent || '';
    expect(text).not.toContain('Nota adicional del docente');
  });

  it('omite el bloque Qué deben lograr los estudiantes cuando description viene vacía', () => {
    component.openDrawer({ ...mockRequests[2], description: '' });
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent || '';
    expect(text).not.toContain('Qué deben lograr los estudiantes');
  });

  it('exige al menos 5 caracteres al pedir ajustes (requestChanges) y al rechazar (reject)', () => {
    const req = mockRequests[0];
    component.openDrawer(req);

    // Nota vacía
    component.reviewNote.set('');
    component.requestChanges();
    expect(component.error()).toContain('mínimo 5 caracteres');
    expect(requestServiceMock.review).not.toHaveBeenCalled();

    // Nota corta (3 caracteres)
    component.reviewNote.set('abc');
    component.reject();
    expect(component.error()).toContain('mínimo 5 caracteres');
    expect(requestServiceMock.review).not.toHaveBeenCalled();

    // Nota válida
    component.reviewNote.set('Falta especificar versión de Python requerida.');
    component.requestChanges();
    expect(requestServiceMock.review).toHaveBeenCalledWith('req-1', {
      status: 'NECESITA_CAMBIO',
      review_note: 'Falta especificar versión de Python requerida.'
    });
  });
});
