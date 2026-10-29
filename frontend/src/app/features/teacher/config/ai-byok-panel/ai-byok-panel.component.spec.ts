import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of, throwError } from 'rxjs';
import { AiByokPanelComponent } from './ai-byok-panel.component';
import {
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../../services/teacher-ai-copilot.service';

const institutionalOnly: AIUserOverrideDTO = {
  enabled: false,
  provider: 'openrouter',
  provider_label: 'OpenRouter',
  model: '',
  base_url: '',
  has_api_key: false,
  api_key_masked: '',
  updated_at: '',
  institutional_enabled: true,
  institutional_provider_label: 'Google Gemini',
  institutional_model: 'gemini-2.5-flash'
};

const withPersonalKey: AIUserOverrideDTO = {
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

describe('AiByokPanelComponent', () => {
  let component: AiByokPanelComponent;
  let fixture: ComponentFixture<AiByokPanelComponent>;
  let mockService: {
    getMySettings: ReturnType<typeof vi.fn>;
    updateMySettings: ReturnType<typeof vi.fn>;
    testMyConnection: ReturnType<typeof vi.fn>;
    listMyModels: ReturnType<typeof vi.fn>;
    resolveError: ReturnType<typeof vi.fn>;
  };

  const buildService = (settings: AIUserOverrideDTO) => ({
    getMySettings: vi.fn().mockReturnValue(of(settings)),
    updateMySettings: vi.fn().mockReturnValue(of(withPersonalKey)),
    testMyConnection: vi
      .fn()
      .mockReturnValue(of({ ok: true, message: 'Conexión OK (120ms)', latency_ms: 120 })),
    listMyModels: vi.fn().mockReturnValue(of(['modelo-a', 'modelo-b:free'])),
    resolveError: vi.fn().mockReturnValue('Error de prueba')
  });

  const setup = async (settings: AIUserOverrideDTO = institutionalOnly) => {
    mockService = buildService(settings);

    await TestBed.configureTestingModule({
      imports: [AiByokPanelComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: TeacherAiCopilotService, useValue: mockService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AiByokPanelComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  it('carga la configuración personal del docente al iniciar', async () => {
    await setup(withPersonalKey);

    expect(mockService.getMySettings).toHaveBeenCalled();
    expect(component.enabled()).toBe(true);
    expect(component.hasSavedKey()).toBe(true);
    expect(component.model()).toBe('meta-llama/llama-3.3-70b-instruct:free');
  });

  it('sin clave guardada arranca inactivo y sin acción de borrado disponible', async () => {
    await setup();

    expect(component.enabled()).toBe(false);
    expect(component.hasSavedKey()).toBe(false);
    expect(component.institutionalAvailable()).toBe(true);
  });

  it('guarda la preferencia activada junto con el proveedor y el modelo elegido', async () => {
    await setup();

    component.onToggleEnabled(true);
    component.onProviderChange('gemini');
    component.model.set('gemini-2.5-flash');
    component.apiKey.set('AIzaSy-clave');
    component.save();

    expect(mockService.updateMySettings).toHaveBeenCalledWith({
      enabled: true,
      provider: 'gemini',
      model: 'gemini-2.5-flash',
      base_url: undefined,
      api_key: 'AIzaSy-clave'
    });
    expect(component.apiKey()).toBe('');
    expect(component.feedbackOk()).toBe(true);
  });

  it('no guarda sin modelo y avisa al docente', async () => {
    await setup();

    component.model.set('   ');
    component.save();

    expect(mockService.updateMySettings).not.toHaveBeenCalled();
    expect(component.feedbackOk()).toBe(false);
    expect(component.feedbackMessage()).toContain('modelo');
  });

  it('quitar la clave vuelve al motor institucional sin fallback automático', async () => {
    await setup(withPersonalKey);
    mockService.updateMySettings.mockReturnValue(of({ ...withPersonalKey, enabled: false, has_api_key: false }));

    component.clearKey();

    expect(mockService.updateMySettings).toHaveBeenCalledWith({
      enabled: false,
      provider: 'openrouter',
      model: 'meta-llama/llama-3.3-70b-instruct:free',
      clear_api_key: true
    });
    expect(component.enabled()).toBe(false);
    expect(component.feedbackMessage()).toContain('institucional');
  });

  it('propaga el error del servicio al probar la conexión sin romper el formulario', async () => {
    await setup();
    mockService.testMyConnection.mockReturnValue(throwError(() => new Error('boom')));

    component.model.set('gemini-2.5-flash');
    component.testConnection();

    expect(component.testing()).toBe(false);
    expect(component.feedbackOk()).toBe(false);
    expect(component.feedbackMessage()).toBe('Error de prueba');
  });

  it('lista modelos y filtra los gratuitos cuando se pide solo gratis', async () => {
    await setup();

    component.fetchModels();

    expect(mockService.listMyModels).toHaveBeenCalledWith({
      provider: 'openrouter',
      api_key: undefined,
      base_url: undefined
    });
    expect(component.availableModels()).toEqual(['modelo-a', 'modelo-b:free']);

    component.modelFilter.set('free');
    expect(component.filteredModels()).toEqual(['modelo-b:free']);
  });
});
