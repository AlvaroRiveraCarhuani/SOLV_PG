import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StudioAiCopilotModalComponent } from './studio-ai-copilot-modal.component';
import { AiProposalView } from '../../template-studio.utils';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';

describe('StudioAiCopilotModalComponent', () => {
  let component: StudioAiCopilotModalComponent;
  let fixture: ComponentFixture<StudioAiCopilotModalComponent>;

  const mockProposal: AiProposalView = {
    runtimeChoices: [],
    webChoices: [],
    addonChoices: [],
    databaseChoices: [],
    runtime: {
      id: 'rt-python-312',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12',
      description: 'Python environment',
      icon_key: 'python',
      base_image: 'python:3.12-slim-bookworm',
      compatible_runtimes: ['python'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: 'python3 solution.py',
      tools_provided: ['python3', 'pip'],
      service_engine: 'none',
      ram_delta_mb: 0,
      is_system: true,
      is_active: true,
      sort_order: 1
    },
    runtimeSlug: 'python-3.12',
    web: null,
    webSlug: '',
    addons: [
      {
        id: 'ad-fastapi',
        slug: 'fastapi',
        layer: 'addon',
        sub_category: 'framework',
        name: 'FastAPI',
        description: 'FastAPI framework',
        icon_key: 'fastapi',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'pip',
        package_names: ['fastapi', 'uvicorn'],
        build_layer_snippet: 'RUN pip install fastapi uvicorn',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['fastapi'],
        service_engine: 'none',
        ram_delta_mb: 256,
        is_system: true,
        is_active: true,
        sort_order: 1
      }
    ],
    addonSlugs: ['fastapi'],
    database: null,
    databaseSlug: '',
    missing: ['pytest-custom'],
    ramMB: 1536,
    hasSetupScript: true,
    rationale: 'Arquitectura moderna para desarrollo de APIs REST en Python.',
    latencyMs: 820
  };

  const option = (id: string, name: string, layer: 'runtime' | 'addon' | 'database'): TemplateComponent => ({
    id,
    slug: id,
    layer,
    sub_category: layer === 'runtime' ? 'backend' : layer === 'addon' ? 'framework' : 'database',
    name,
    description: '',
    icon_key: '',
    base_image: '',
    compatible_runtimes: ['python'],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: layer === 'database' ? 'postgres' : 'none',
    ram_delta_mb: 0,
    is_system: true,
    is_active: true,
    sort_order: 0
  });

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StudioAiCopilotModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(StudioAiCopilotModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('should emit promptChange when typing in textarea', () => {
    const spy = vi.fn();
    component.promptChange.subscribe(spy);

    const textarea: HTMLTextAreaElement = fixture.nativeElement.querySelector('textarea');
    textarea.value = 'Nuevo prompt de arquitectura';
    textarea.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(spy).toHaveBeenCalledWith('Nuevo prompt de arquitectura');
  });

  it('should disable submit button when prompt length is under 3 characters', () => {
    fixture.componentRef.setInput('prompt', 'Py');
    fixture.detectChanges();

    const submitBtn: HTMLButtonElement | null = fixture.nativeElement.querySelector('.btn-ai-submit');
    expect(submitBtn?.disabled).toBe(true);
  });

  it('should enable submit button and emit suggest when clicked', () => {
    const spy = vi.fn();
    component.suggest.subscribe(spy);

    fixture.componentRef.setInput('prompt', 'Python con FastAPI y PostgreSQL');
    fixture.detectChanges();

    const submitBtn: HTMLButtonElement = fixture.nativeElement.querySelector('.btn-ai-submit');
    expect(submitBtn.disabled).toBe(false);

    submitBtn.click();
    expect(spy).toHaveBeenCalled();
  });

  it('should display telemetry counter when isSuggesting is true', () => {
    const cancelSpy = vi.fn();
    component.cancelSuggest.subscribe(cancelSpy);

    fixture.componentRef.setInput('isSuggesting', true);
    fixture.componentRef.setInput('elapsedSeconds', '4.2');
    fixture.detectChanges();

    const telemetry = fixture.nativeElement.querySelector('.ai-running-telemetry');
    expect(telemetry).toBeTruthy();
    expect(telemetry.textContent).toContain('4.2 s');

    const cancelBtn: HTMLButtonElement = fixture.nativeElement.querySelector('.btn-cancel-ai');
    cancelBtn.click();
    expect(cancelSpy).toHaveBeenCalled();
  });

  it('should render the specification sheet when proposalView is present', () => {
    fixture.componentRef.setInput('proposalView', {
      ...mockProposal,
      runtimeChoices: [option('rt-python-312', 'Python 3.12', 'runtime')],
      addonChoices: [option('ad-fastapi', 'FastAPI', 'addon')]
    });
    fixture.componentRef.setInput('constellation', {
      core: {
        role: 'lenguaje',
        name: 'Python 3.12',
        slug: 'python-312',
        iconKey: 'python',
        availability: 'available',
        isCore: true,
        ramDeltaMB: 0
      },
      satellites: [
        {
          role: 'addon',
          name: 'FastAPI',
          slug: 'fastapi',
          iconKey: 'fastapi',
          availability: 'available',
          isCore: false,
          ramDeltaMB: 256
        },
        {
          role: 'addon',
          name: 'pytest-custom',
          slug: 'pytest-custom',
          iconKey: '',
          availability: 'missing',
          isCore: false,
          ramDeltaMB: 0
        }
      ],
      ramMB: 1536,
      hasSetupScript: true
    });
    fixture.detectChanges();

    const review = fixture.nativeElement.querySelector('.proposal-review');
    expect(review).toBeTruthy();
    expect(review.textContent).toContain('Revisión de la propuesta');
    expect(review.textContent).toContain('Python 3.12');
    expect(review.textContent).toContain('FastAPI');
    expect(review.textContent).toContain('pytest-custom');
    expect(review.querySelector('.proposal-map-details')).toBeTruthy();
    expect(review.querySelector('select')).toBeTruthy();
    expect(review.querySelector('input[type="checkbox"]')).toBeTruthy();
  });

  it('should emit apply and discard when buttons in footer are clicked', () => {
    const applySpy = vi.fn();
    const discardSpy = vi.fn();
    component.apply.subscribe(applySpy);
    component.discard.subscribe(discardSpy);

    fixture.componentRef.setInput('proposalView', mockProposal);
    fixture.detectChanges();

    const buttons = fixture.nativeElement.querySelectorAll('footer button');
    const discardBtn = Array.from(buttons).find((button) => (button as HTMLButtonElement).textContent?.includes('Descartar')) as HTMLButtonElement;
    const applyBtn = Array.from(buttons).find((button) => (button as HTMLButtonElement).textContent?.includes('Aplicar')) as HTMLButtonElement;

    expect(discardBtn).toBeTruthy();
    expect(applyBtn).toBeTruthy();

    discardBtn.click();
    expect(discardSpy).toHaveBeenCalled();

    applyBtn.click();
    expect(applySpy).toHaveBeenCalled();
  });

  it('should display error banner when errorMessage is provided and not suggesting', () => {
    fixture.componentRef.setInput('errorMessage', 'El motor de IA superó el tiempo límite.');
    fixture.componentRef.setInput('isSuggesting', false);
    fixture.componentRef.setInput('proposalView', null);
    fixture.detectChanges();

    const banner = fixture.nativeElement.querySelector('.ai-error-banner');
    expect(banner).toBeTruthy();
    expect(banner.textContent).toContain('El motor de IA superó el tiempo límite.');
  });

  it('emite suggest al presionar Ctrl+Enter o Cmd+Enter si el prompt tiene al menos 3 caracteres', () => {
    const suggestSpy = vi.fn();
    component.suggest.subscribe(suggestSpy);

    fixture.componentRef.setInput('prompt', 'Python 3.12');
    fixture.componentRef.setInput('isSuggesting', false);

    const eventCtrl = new KeyboardEvent('keydown', { key: 'Enter', ctrlKey: true });
    const preventCtrl = vi.spyOn(eventCtrl, 'preventDefault');
    component.onPromptKeydown(eventCtrl);
    expect(preventCtrl).toHaveBeenCalled();
    expect(suggestSpy).toHaveBeenCalledTimes(1);

    const eventMeta = new KeyboardEvent('keydown', { key: 'Enter', metaKey: true });
    const preventMeta = vi.spyOn(eventMeta, 'preventDefault');
    component.onPromptKeydown(eventMeta);
    expect(preventMeta).toHaveBeenCalled();
    expect(suggestSpy).toHaveBeenCalledTimes(2);

    // No emite si está sugiriendo
    fixture.componentRef.setInput('isSuggesting', true);
    component.onPromptKeydown(eventCtrl);
    expect(suggestSpy).toHaveBeenCalledTimes(2);
  });
});
