export type EnvironmentTypeKey = 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | string;

export interface EnvironmentTypeMeta {
  key: string;
  label: string;
  short: string;
  description: string;
  badgeClass: string;
}

export const ENVIRONMENT_TYPES: Record<string, EnvironmentTypeMeta> = {
  IDE_PERSISTENTE: {
    key: 'IDE_PERSISTENTE',
    label: 'Entorno de Desarrollo',
    short: 'Desarrollo',
    description: 'Espacio de trabajo interactivo con VS Code para proyectos y prácticas continuas.',
    badgeClass: 'env-desarrollo'
  },
  JUEZ_EFIMERO: {
    key: 'JUEZ_EFIMERO',
    label: 'Entorno de Evaluación',
    short: 'Evaluación',
    description: 'Ejecución y corrección automática con casos de prueba para ejercicios específicos.',
    badgeClass: 'env-evaluacion'
  },
  JUEZ_VIRTUAL: {
    key: 'JUEZ_EFIMERO',
    label: 'Entorno de Evaluación',
    short: 'Evaluación',
    description: 'Ejecución y corrección automática con casos de prueba para ejercicios específicos.',
    badgeClass: 'env-evaluacion'
  }
};

export function getEnvironmentMeta(type?: string | null): EnvironmentTypeMeta {
  const key = (type || '').trim().toUpperCase();
  if (key === 'IDE_PERSISTENTE' || key === 'WORKSPACE' || key === 'IDE' || key === 'PROJECT') {
    return ENVIRONMENT_TYPES['IDE_PERSISTENTE'];
  }
  return ENVIRONMENT_TYPES['JUEZ_EFIMERO'];
}
