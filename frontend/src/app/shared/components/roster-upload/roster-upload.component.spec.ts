import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { RosterUploadComponent } from './roster-upload.component';
import { ParsedRosterResult } from '@core/models/admin.model';

const SAMPLE_ROSTER: ParsedRosterResult = {
  total_detected: 2,
  valid_emails: ['estudiante1@uab.edu.bo', 'estudiante2@uab.edu.bo'],
  duplicate_emails: [],
  invalid_emails: []
};

describe('RosterUploadComponent', () => {
  let component: RosterUploadComponent;
  let fixture: ComponentFixture<RosterUploadComponent>;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RosterUploadComponent],
      providers: [provideHttpClientTesting()]
    }).compileComponents();

    fixture = TestBed.createComponent(RosterUploadComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    fixture.componentRef.setInput('parseEndpoint', '/api/v1/admin/students/parse-roster');
    fixture.detectChanges();
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('rechaza formatos no admitidos', () => {
    const bad = new File(['x'], 'lista.txt', { type: 'text/plain' });
    component.processFile(bad);
    expect(component.parseError()).toContain('Formato no admitido');
    expect(component.parsedRoster()).toBeNull();
  });

  it('envía el archivo al endpoint y emite el resultado parseado', () => {
    let emitted: ParsedRosterResult | null = null;
    component.parsed.subscribe(res => { emitted = res; });

    const file = new File(['email\nestudiante1@uab.edu.bo'], 'roster.csv', { type: 'text/csv' });
    component.processFile(file);

    const req = httpMock.expectOne('/api/v1/admin/students/parse-roster');
    expect(req.request.method).toBe('POST');
    expect(req.request.body instanceof FormData).toBe(true);
    req.flush(SAMPLE_ROSTER);

    expect(component.selectedFile()).toBe(file);
    expect(component.hasValid()).toBe(true);
    expect(component.parsedRoster()?.valid_emails.length).toBe(2);
    expect((emitted as ParsedRosterResult | null)?.valid_emails).toEqual(SAMPLE_ROSTER.valid_emails);
  });

  it('muestra error cuando el backend falla', () => {
    const file = new File(['email\na@uab.edu.bo'], 'roster.csv', { type: 'text/csv' });
    component.processFile(file);

    const req = httpMock.expectOne('/api/v1/admin/students/parse-roster');
    req.flush({ message: 'boom' }, { status: 500, statusText: 'Server Error' });

    expect(component.parsedRoster()).toBeNull();
    expect(component.parseError()).toContain('No se pudo leer el archivo');
  });

  it('reset limpia el estado', () => {
    component.parsedRoster.set(SAMPLE_ROSTER);
    component.reset();
    expect(component.parsedRoster()).toBeNull();
    expect(component.selectedFile()).toBeNull();
    expect(component.parseError()).toBeNull();
  });
});
