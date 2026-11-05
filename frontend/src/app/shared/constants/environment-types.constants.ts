/**
 * Catálogo único de tipos de entorno.
 *
 * Los identificadores (`IDE_PERSISTENTE`, `JUEZ_EFIMERO`, `AMBOS`) son valores de
 * contrato: no se traducen ni se renombran. Lo que se muestra a la persona sale
 * siempre de aquí, a través de `environmentLabel`.
 *
 * Nombres visibles oficiales:
 *   IDE_PERSISTENTE -> Entorno de Desarrollo
 *   JUEZ_EFIMERO    -> Entorno de Evaluación
 *   AMBOS           -> Entorno de Desarrollo y Evaluación
 */
export type EnvironmentTypeKey = 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS';

/**
 * Nombre visible del motor que evalúa soluciones. No es una modalidad: en la
 * interfaz solo puede aparecer como título o en prosa, nunca como etiqueta de
 * modalidad ni como valor de un campo `environment_type`.
 */
export const EVALUATION_ENGINE_NAME = 'Juez Virtual';

export interface EnvironmentTypeMeta {
  key: EnvironmentTypeKey;
  label: string;
  short: string;
  description: string;
  badgeClass: string;
}

/**
 * Presentación resuelta de una modalidad. Un valor desconocido se presenta como
 * `variant: 'unknown'`, nunca como Desarrollo ni como Evaluación.
 */
export interface EnvironmentBadge {
  key: EnvironmentTypeKey | null;
  label: string;
  short: string;
  description: string;
  badgeClass: string;
  variant: 'ide' | 'judge' | 'dual' | 'unknown';
}

export const ENVIRONMENT_TYPES: Record<EnvironmentTypeKey, EnvironmentTypeMeta> = {
  IDE_PERSISTENTE: {
    key: 'IDE_PERSISTENTE',
    label: 'Entorno de Desarrollo',
    short: 'Desarrollo',
    description: 'Espacio de trabajo interactivo con VS Code para proyectos y prácticas continuas.',
    badgeClass: 'env-desarrollo',
  },
  JUEZ_EFIMERO: {
    key: 'JUEZ_EFIMERO',
    label: 'Entorno de Evaluación',
    short: 'Evaluación',
    description: 'Ejecución y corrección automática con casos de prueba para ejercicios específicos.',
    badgeClass: 'env-evaluacion',
  },
  AMBOS: {
    key: 'AMBOS',
    label: 'Entorno de Desarrollo y Evaluación',
    short: 'Desarrollo y Evaluación',
    description: 'La configuración sirve tanto para el trabajo en el entorno de desarrollo como para la corrección en el entorno de evaluación.',
    badgeClass: 'env-dual',
  },
};

/**
 * Alias históricos aceptados al leer datos ya guardados. No son tipos de
 * entorno: se resuelven al tipo que representan y no se propagan a etiquetas.
 */
export const ENVIRONMENT_TYPE_ALIASES: Readonly<Record<string, EnvironmentTypeKey>> = {
  JUEZ_VIRTUAL: 'JUEZ_EFIMERO',
  WORKSPACE: 'IDE_PERSISTENTE',
  IDE: 'IDE_PERSISTENTE',
  PROJECT: 'IDE_PERSISTENTE',
};

export const UNRECOGNIZED_ENVIRONMENT_BADGE: EnvironmentBadge = {
  key: null,
  label: 'Modalidad sin reconocer',
  short: 'Sin reconocer',
  description: 'El valor guardado no corresponde a ninguna modalidad conocida.',
  badgeClass: 'env-desconocido',
  variant: 'unknown',
};

const ENVIRONMENT_BADGE_VARIANTS: Record<EnvironmentTypeKey, 'ide' | 'judge' | 'dual'> = {
  IDE_PERSISTENTE: 'ide',
  JUEZ_EFIMERO: 'judge',
  AMBOS: 'dual',
};

export interface EnvironmentTypeOption {
  id: string;
  label: string;
  value: string;
}

export const ALL_ENVIRONMENT_TYPE_KEYS: readonly EnvironmentTypeKey[] = [
  'IDE_PERSISTENTE',
  'JUEZ_EFIMERO',
  'AMBOS',
];

/** Resuelve un valor de contrato a su tipo. Desconocido o vacío devuelve `null`. */
export function resolveEnvironmentType(value?: string | null): EnvironmentTypeKey | null {
  const key = (value ?? '').trim().toUpperCase();
  if (!key) return null;
  if (Object.prototype.hasOwnProperty.call(ENVIRONMENT_TYPES, key)) {
    return key as EnvironmentTypeKey;
  }
  return ENVIRONMENT_TYPE_ALIASES[key] ?? null;
}

/**
 * Devuelve la definición del tipo. Un valor desconocido no se presenta como
 * evaluación: devuelve `null` para que quien lo use decida qué mostrar.
 */
export function getEnvironmentMeta(value?: string | null): EnvironmentTypeMeta | null {
  const key = resolveEnvironmentType(value);
  return key ? ENVIRONMENT_TYPES[key] : null;
}

/** Compara un valor guardado con un tipo, resolviendo antes sus alias históricos. */
export function isEnvironmentType(value: string | null | undefined, key: EnvironmentTypeKey): boolean {
  return resolveEnvironmentType(value) === key;
}

/** Presentación de un valor guardado. Un valor desconocido devuelve el estado neutro. */
export function environmentBadge(value?: string | null): EnvironmentBadge {
  const meta = getEnvironmentMeta(value);
  if (!meta) return UNRECOGNIZED_ENVIRONMENT_BADGE;
  return { ...meta, variant: ENVIRONMENT_BADGE_VARIANTS[meta.key] };
}

/**
 * Opciones de filtro y selector para los tipos de entorno. Evita que cada
 * pantalla repita las etiquetas.
 */
export function environmentTypeOptions(options: {
  /** Tipos que ofrece el selector. Obligatorio: `AMBOS` solo aparece donde el dominio lo permite. */
  include: readonly EnvironmentTypeKey[];
  includeAll?: boolean;
  allLabel?: string;
  allValue?: string;
}): EnvironmentTypeOption[] {
  const entries: EnvironmentTypeOption[] = [];
  if (options.includeAll) {
    entries.push({
      id: options.allValue ?? 'all',
      label: options.allLabel ?? 'Todas las modalidades',
      value: options.allValue ?? 'all',
    });
  }
  for (const key of options.include) {
    const meta = ENVIRONMENT_TYPES[key];
    entries.push({ id: meta.key, label: meta.label, value: meta.key });
  }
  return entries;
}
