import { TestBed } from '@angular/core/testing';
import { AiFlowDrafts } from './ai-flow-drafts';

describe('AiFlowDrafts', () => {
  function create(): AiFlowDrafts {
    TestBed.configureTestingModule({});
    return TestBed.inject(AiFlowDrafts);
  }

  it('devuelve vacío un flujo que todavía no guardó nada', () => {
    expect(create().get('suggest-component')).toBeNull();
  });

  it('guarda y recupera el borrador de un flujo', () => {
    const drafts = create();
    drafts.save('generate-exercise', { prompt: 'Árboles binarios' });

    expect(drafts.get<{ prompt: string }>('generate-exercise')).toEqual({
      prompt: 'Árboles binarios'
    });
  });

  it('separa los borradores por clave de flujo', () => {
    const drafts = create();
    drafts.save('generate-exercise', { prompt: 'Árboles' });
    drafts.save('suggest-component', { prompt: 'FastAPI' });

    expect(drafts.get<{ prompt: string }>('generate-exercise')?.prompt).toBe('Árboles');
    expect(drafts.get<{ prompt: string }>('suggest-component')?.prompt).toBe('FastAPI');
  });

  it('clear libera solo el flujo indicado', () => {
    const drafts = create();
    drafts.save('generate-exercise', { prompt: 'Árboles' });
    drafts.save('suggest-component', { prompt: 'FastAPI' });

    drafts.clear('generate-exercise');

    expect(drafts.get('generate-exercise')).toBeNull();
    expect(drafts.get<{ prompt: string }>('suggest-component')?.prompt).toBe('FastAPI');
  });
});
