import {
  ALL_ENVIRONMENT_TYPE_KEYS,
  ENVIRONMENT_TYPES,
  EVALUATION_ENGINE_NAME,
  environmentBadge,
  environmentTypeOptions,
  isEnvironmentType,
  resolveEnvironmentType,
} from './environment-types.constants';

describe('catálogo de tipos de entorno', () => {
  it('resuelve alias históricos al tipo que representan', () => {
    expect(resolveEnvironmentType('JUEZ_VIRTUAL')).toBe('JUEZ_EFIMERO');
    expect(resolveEnvironmentType('workspace')).toBe('IDE_PERSISTENTE');
    expect(resolveEnvironmentType('JUEZ_EFIMERO')).toBe('JUEZ_EFIMERO');
    expect(resolveEnvironmentType('OTRO_VALOR')).toBeNull();
    expect(resolveEnvironmentType(null)).toBeNull();
  });

  it('compara valores con alias mediante isEnvironmentType', () => {
    expect(isEnvironmentType('JUEZ_VIRTUAL', 'JUEZ_EFIMERO')).toBe(true);
    expect(isEnvironmentType('AMBOS', 'JUEZ_EFIMERO')).toBe(false);
    expect(isEnvironmentType('IDE_PERSISTENTE', 'IDE_PERSISTENTE')).toBe(true);
    expect(isEnvironmentType(undefined, 'IDE_PERSISTENTE')).toBe(false);
  });

  it('presenta un valor desconocido como estado neutro, no como una modalidad', () => {
    const badge = environmentBadge('OTRO_VALOR');
    expect(badge.variant).toBe('unknown');
    expect(badge.key).toBeNull();
    expect(badge.label).not.toContain('Desarrollo');
    expect(badge.label).not.toContain('Evaluación');
    expect(environmentBadge(undefined).variant).toBe('unknown');
  });

  it('presenta cada tipo con su etiqueta, texto corto y variante', () => {
    expect(environmentBadge('IDE_PERSISTENTE')).toMatchObject({
      label: 'Entorno de Desarrollo',
      short: 'Desarrollo',
      variant: 'ide',
    });
    expect(environmentBadge('JUEZ_VIRTUAL')).toMatchObject({
      label: 'Entorno de Evaluación',
      variant: 'judge',
    });
    expect(environmentBadge('AMBOS')).toMatchObject({
      label: 'Entorno de Desarrollo y Evaluación',
      variant: 'dual',
    });
  });

  it('ofrece solo los tipos que el selector pide', () => {
    const exerciseOptions = environmentTypeOptions({ include: ['IDE_PERSISTENTE', 'JUEZ_EFIMERO'] });
    expect(exerciseOptions.map(option => option.value)).toEqual(['IDE_PERSISTENTE', 'JUEZ_EFIMERO']);
    expect(exerciseOptions.some(option => option.value === 'AMBOS')).toBe(false);

    const libraryOptions = environmentTypeOptions({ include: ALL_ENVIRONMENT_TYPE_KEYS, includeAll: true });
    expect(libraryOptions.map(option => option.value)).toEqual(['all', 'IDE_PERSISTENTE', 'JUEZ_EFIMERO', 'AMBOS']);
  });

  it('mantiene el nombre del motor fuera de las etiquetas de modalidad', () => {
    expect(EVALUATION_ENGINE_NAME).toBe('Juez Virtual');
    expect(Object.values(ENVIRONMENT_TYPES).map(meta => meta.label)).not.toContain(EVALUATION_ENGINE_NAME);
  });
});
