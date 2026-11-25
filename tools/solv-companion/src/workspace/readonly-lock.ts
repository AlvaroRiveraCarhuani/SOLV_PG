import { constants as fsConstants, promises as fs } from 'node:fs';

/** Patrones aplicados en User Settings para activar solo lectura nativa en OpenVSCode. */
export const SOLV_READONLY_PATTERNS: Readonly<Record<string, boolean>> = {
  '/home/workspace/**': true,
  '**/*': true
};

const MANAGED_KEYS = Object.keys(SOLV_READONLY_PATTERNS);

export interface ReadonlyIncludeDecision {
  changed: boolean;
  nextValue: Record<string, boolean> | undefined;
  nextOwned: boolean;
}

/**
 * Comprueba si un directorio permite escritura en el sistema de archivos.
 * Si el volumen esta montado en modo :ro devuelve false (EROFS/EACCES/EPERM).
 */
export async function isDirectoryWritable(dirPath: string): Promise<boolean> {
  try {
    await fs.access(dirPath, fsConstants.W_OK);
    return true;
  } catch (error) {
    const code = (error as NodeJS.ErrnoException | undefined)?.code;
    if (code === 'EROFS' || code === 'EACCES' || code === 'EPERM') {
      return false;
    }
    return true;
  }
}

function isExactSolvReadonlyMap(value: Record<string, boolean> | undefined): boolean {
  if (!value) {
    return false;
  }
  const keys = Object.keys(value);
  if (keys.length !== MANAGED_KEYS.length) {
    return false;
  }
  return MANAGED_KEYS.every(key => value[key] === true);
}

/**
 * Calcula de forma idempotente el valor destino de `files.readonlyInclude`
 * sin sobreescribir patrones definidos por el usuario fuera de SOLV.
 */
export function computeReadonlyIncludeDecision(
  shouldLock: boolean,
  currentGlobal: Record<string, boolean> | undefined,
  ownedBySolv: boolean
): ReadonlyIncludeDecision {
  if (shouldLock) {
    const alreadyCovered =
      currentGlobal !== undefined && MANAGED_KEYS.every(key => currentGlobal[key] === true);
    if (alreadyCovered) {
      return {
        changed: false,
        nextValue: currentGlobal,
        nextOwned: ownedBySolv
      };
    }
    return {
      changed: true,
      nextValue: { ...(currentGlobal ?? {}), ...SOLV_READONLY_PATTERNS },
      nextOwned: true
    };
  }

  if (!ownedBySolv && !isExactSolvReadonlyMap(currentGlobal)) {
    return {
      changed: false,
      nextValue: currentGlobal,
      nextOwned: false
    };
  }

  if (!currentGlobal) {
    return {
      changed: false,
      nextValue: undefined,
      nextOwned: false
    };
  }

  const remaining: Record<string, boolean> = { ...currentGlobal };
  let removedAny = false;
  for (const key of MANAGED_KEYS) {
    if (key in remaining) {
      delete remaining[key];
      removedAny = true;
    }
  }

  const nextValue = Object.keys(remaining).length > 0 ? remaining : undefined;
  return {
    changed: removedAny,
    nextValue,
    nextOwned: false
  };
}
