import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { TeacherAiSettingsComponent } from './teacher-ai-settings.component';
import {
  AIAvailabilityStatus,
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../services/teacher-ai-copilot.service';

const personalSettings: AIUserOverrideDTO = {
  enabled: true,
  provider: 'openrouter',
  provider_label: 'OpenRouter',
  model: 'meta-llama/llama-3.3-70b-instruct:free',
  base_url: '',
  has_api_key: true,
  api_key_masked: 'sk-or-•••••••1234',
  updated_at: '2026-10-25T12:00:00Z',
  institutional_enabled: true,
  institutional_provider_label: 'Google Gemini',
  institutional_model: 'gemini-2.5-flash'
};

const institutionalStatus: AIAvailabilityStatus = {
  enabled: true,
  provider: 'gemini',
  model: 'gemini-2.5-flash',
  using_personal_key: false,
  institutional_enabled: true
};

describe('TeacherAiSettingsComponent', () => {
  let component: TeacherAiSettingsComponent;
  let fixture: ComponentFixture<TeacherAiSettingsComponent>;
  let mockService: {
    getStatus: ReturnType<typeof vi.fn>;
    getMySettings: ReturnType<typeof vi.fn>;
    resolveError: ReturnType<typeof vi.fn>;
  };

  const setup = async (
    status: AIAvailabilityStatus,
    settings: AIUserOverrideDTO
  ): Promise<void> => {
    mockService = {
      getStatus: vi.fn().mockReturnValue(of(status)),
      getMySettings: vi.fn().mockReturnValue(of(settings)),
      resolveError: vi.fn().mockReturnValue('Error')
    };

    await TestBed.configureTestingModule({
      imports: [TeacherAiSettingsComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: TeacherAiCopilotService, useValue: mockService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherAiSettingsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  it('refleja el motor institucional cuando la institución lo tiene habilitado', async () => {
    await setup(institutionalStatus, { ...personalSettings, enabled: false, has_api_key: false });

    expect(component.institutionalEnabled()).toBe(true);
    expect(component.institutionalProvider()).toBe('Google Gemini');
    expect(component.institutionalModel()).toBe('gemini-2.5-flash');
    expect(component.usingPersonalKey()).toBe(false);
  });

  it('informa que el docente opera con su clave personal', async () => {
    await setup({ ...institutionalStatus, using_personal_key: true }, personalSettings);

    expect(component.usingPersonalKey()).toBe(true);
    expect(component.institutionalBadgeLabel()).toBe('EN PAUSA');
    expect(fixture.nativeElement.textContent).toContain('EN PAUSA');
    expect(fixture.nativeElement.textContent).toContain('tu clave personal tiene prioridad');
  });

  it('muestra HABILITADO cuando el motor institucional es el que se usa', async () => {
    await setup(institutionalStatus, { ...personalSettings, enabled: false, has_api_key: false });

    expect(component.institutionalBadgeLabel()).toBe('HABILITADO');
    expect(fixture.nativeElement.textContent).not.toContain('EN PAUSA');
  });

  it('advierte cuando la institución no habilitó la IA', async () => {
    await setup(
      { enabled: false, provider: '', model: '', using_personal_key: false, institutional_enabled: false },
      { ...personalSettings, enabled: false, has_api_key: false, institutional_enabled: false }
    );

    expect(component.institutionalEnabled()).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('no tiene la IA habilitada');
  });
});
