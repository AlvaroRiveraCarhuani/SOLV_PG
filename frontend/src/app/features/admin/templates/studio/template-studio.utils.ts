export interface DockerfileDiagnostic {
  severity: 'error' | 'warning';
  message: string;
}

export interface VerificationParameters {
  targetEnvironment: string;
  dockerImage: string;
  dockerfileContent: string;
  setupScript: string;
  entrypoint: string;
  toolsDeclared: string;
  timeoutMS: number;
  ramMB: number;
}

/**
 * Analiza estáticamente el contenido de un Dockerfile en busca de conformidades
 * con las reglas institucionales de gobernanza de SOLV.
 */
export function analyzeDockerfile(content: string): DockerfileDiagnostic[] {
  const trimmed = (content || '').trim();
  if (!trimmed) return [];

  const diagnostics: DockerfileDiagnostic[] = [];
  const activeLines = trimmed
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.length > 0 && !l.startsWith('#'));

  const fromLines = activeLines.filter(l => l.toUpperCase().startsWith('FROM '));
  if (fromLines.length === 0) {
    diagnostics.push({
      severity: 'error',
      message: 'El Dockerfile debe comenzar con una instrucción FROM válida (ej: FROM python:3.12-slim-bookworm).'
    });
  } else {
    for (const fl of fromLines) {
      const tokens = fl.split(/\s+/).filter(Boolean);
      const img = tokens[1] || '';
      if (!img || img.endsWith(':latest') || img.includes(':latest@') || (!img.includes(':') && !img.includes('@'))) {
        diagnostics.push({
          severity: 'error',
          message: 'El uso de :latest o imágenes sin versión fija está prohibido por reproducibilidad académica.'
        });
        break;
      }
    }
  }

  for (const line of activeLines) {
    if (line.includes('apt-get install')) {
      const hasY = /(?:^|\s)(-y|--assume-yes)\b/.test(line);
      if (!hasY) {
        diagnostics.push({
          severity: 'warning',
          message: 'Se detectó apt-get install sin el parámetro -y; la construcción desatendida podría detenerse esperando confirmación.'
        });
        break;
      }
    }
    if (/\bsudo\b/.test(line)) {
      diagnostics.push({
        severity: 'warning',
        message: 'Se detectó el comando "sudo". Los contenedores de aprovisionamiento se ejecutan con permisos administrativos.'
      });
      break;
    }
  }

  return diagnostics;
}

/**
 * Hashing determinista FNV-1a de 32 bits formateado como string hexadecimal de 8 a 12 caracteres.
 */
export function fnv1aHex(input: string): string {
  let h1 = 0x811c9dc5;
  let h2 = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    const ch = input.charCodeAt(i);
    h1 ^= ch;
    h1 = Math.imul(h1, 0x01000193);
    h2 ^= ch ^ (i & 0xff);
    h2 = Math.imul(h2, 0x01000193);
  }
  const p1 = (h1 >>> 0).toString(16).padStart(8, '0');
  const p2 = (h2 >>> 0).toString(16).padStart(8, '0');
  return (p1 + p2).slice(0, 12);
}

/**
 * Calcula una huella digital determinista única sobre los parámetros reales de construcción y ejecución del entorno.
 * Permite detectar cualquier cambio posterior a la prueba de verificación.
 */
export function computeVerificationFingerprint(params: VerificationParameters): string {
  const normalizedTools = (params.toolsDeclared || '')
    .split(',')
    .map(t => t.trim())
    .filter(Boolean)
    .join(',');

  const canonical = [
    (params.targetEnvironment || '').trim(),
    (params.dockerImage || '').trim(),
    (params.dockerfileContent || '').trim(),
    (params.setupScript || '').trim(),
    (params.entrypoint || '').trim(),
    normalizedTools,
    params.timeoutMS ?? 0,
    params.ramMB ?? 0
  ].join(':::');

  return fnv1aHex(canonical);
}
