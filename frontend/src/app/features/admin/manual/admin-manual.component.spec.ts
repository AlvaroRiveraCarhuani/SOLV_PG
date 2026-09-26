import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { AdminManualComponent } from './admin-manual.component';

describe('AdminManualComponent', () => {
  let component: AdminManualComponent;
  let fixture: ComponentFixture<AdminManualComponent>;
  let httpMock: HttpTestingController;

  const mockMarkdown = `
# Manual de Administración del Sistema — SOLV

Guía operativa para administradores.

## 1. Visión General e Identidad del Sistema

### 1.1 Propósito y Alcance
Contenido de la sección de propósito.

## 2. Entornos de Laboratorio

Contenido de entornos.

\`\`\`bash
docker ps --filter "label=solv.managed=true"
\`\`\`

> [!NOTE]
> Esta es una nota informativa importante.
`;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminManualComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([])
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminManualComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('debe crearse correctamente', () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/v1/admin/manual');
    req.flush(mockMarkdown);
    expect(component).toBeTruthy();
  });

  it('debe cargar y parsear el markdown de la API correctamente', () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/v1/admin/manual');
    req.flush(mockMarkdown);
    fixture.detectChanges();

    expect(component.loading()).toBe(false);
    expect(component.error()).toBeNull();
    expect(component.rawMarkdown()).toBe(mockMarkdown);
    expect(component.tocItems().length).toBeGreaterThan(0);
  });

  it('debe recurrir al respaldo estático si la API principal falla', () => {
    fixture.detectChanges();
    const req1 = httpMock.expectOne('/api/v1/admin/manual');
    req1.error(new ProgressEvent('error'));

    const req2 = httpMock.expectOne('/docs/MANUAL_ADMIN.md');
    req2.flush(mockMarkdown);

    expect(component.loading()).toBe(false);
    expect(component.rawMarkdown()).toBe(mockMarkdown);
  });

  it('debe mostrar error si tanto la API como el respaldo estático fallan', () => {
    fixture.detectChanges();
    const req1 = httpMock.expectOne('/api/v1/admin/manual');
    req1.error(new ProgressEvent('error'));

    const req2 = httpMock.expectOne('/docs/MANUAL_ADMIN.md');
    req2.error(new ProgressEvent('error'));

    expect(component.loading()).toBe(false);
    expect(component.error()).toBeTruthy();
  });

  it('debe filtrar secciones al buscar por término', () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/v1/admin/manual');
    req.flush(mockMarkdown);

    component.searchQuery.set('docker');
    fixture.detectChanges();

    const filtered = component.filteredMarkdown();
    expect(filtered).toContain('docker ps');
    expect(filtered).not.toContain('Propósito y Alcance');
  });

  it('debe limpiar la búsqueda al llamar clearSearch', () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/v1/admin/manual');
    req.flush(mockMarkdown);

    component.searchQuery.set('entornos');
    expect(component.searchQuery()).toBe('entornos');

    component.clearSearch();
    expect(component.searchQuery()).toBe('');
  });
});
