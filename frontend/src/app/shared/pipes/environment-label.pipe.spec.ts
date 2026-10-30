import { EnvironmentLabelPipe } from './environment-label.pipe';

describe('EnvironmentLabelPipe', () => {
  const pipe = new EnvironmentLabelPipe();

  it('transforms IDE_PERSISTENTE to full canonical label', () => {
    expect(pipe.transform('IDE_PERSISTENTE')).toBe('Entorno de Desarrollo');
    expect(pipe.transform('workspace')).toBe('Entorno de Desarrollo');
    expect(pipe.transform('ide')).toBe('Entorno de Desarrollo');
  });

  it('transforms JUEZ_EFIMERO to full canonical label', () => {
    expect(pipe.transform('JUEZ_EFIMERO')).toBe('Entorno de Evaluación');
    expect(pipe.transform('JUEZ_VIRTUAL')).toBe('Entorno de Evaluación');
    expect(pipe.transform('')).toBe('Entorno de Evaluación');
    expect(pipe.transform(null)).toBe('Entorno de Evaluación');
  });

  it('returns short format correctly', () => {
    expect(pipe.transform('IDE_PERSISTENTE', 'short')).toBe('Desarrollo');
    expect(pipe.transform('JUEZ_EFIMERO', 'short')).toBe('Evaluación');
  });

  it('returns description format correctly', () => {
    expect(pipe.transform('IDE_PERSISTENTE', 'description')).toContain('Espacio de trabajo interactivo');
    expect(pipe.transform('JUEZ_EFIMERO', 'description')).toContain('casos de prueba');
  });

  it('returns badgeClass format correctly', () => {
    expect(pipe.transform('IDE_PERSISTENTE', 'badgeClass')).toBe('env-desarrollo');
    expect(pipe.transform('JUEZ_EFIMERO', 'badgeClass')).toBe('env-evaluacion');
  });
});
