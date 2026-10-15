export interface ErrorLocation {
  file?: string;
  line?: number;
  functionName?: string;
  rawLocation?: string;
}

export interface ErrorDiagnostic {
  hasError: boolean;
  what: string;
  where?: ErrorLocation;
  why: string;
  rawTrace: string;
  isFormatDiscrepancy?: boolean;
}

/**
 * Detecta si la diferencia entre la salida obtenida y la esperada radica
 * exclusivamente en terminadores de línea (\r\n vs \n) o espacios en blanco residuales.
 */
export function detectWhitespaceDiscrepancy(actual: string, expected: string): boolean {
  if (actual === expected) return false;
  const normalize = (s: string) => s.replace(/\r\n/g, '\n').trim();
  return normalize(actual) === normalize(expected);
}

/**
 * Parsea el flujo de error (stderr) para estructurar el diagnóstico bajo el patrón What-Where-Why.
 */
export function parseErrorDiagnostic(stderr: string, language?: string): ErrorDiagnostic {
  const raw = (stderr || '').trim();
  if (!raw) {
    return {
      hasError: false,
      what: '',
      why: '',
      rawTrace: ''
    };
  }

  const lang = (language || '').toLowerCase();

  // 1. Detección de señales del sistema operativo (C/C++ / Linux POSIX)
  if (raw.includes('SIGSEGV') || raw.toLowerCase().includes('segmentation fault')) {
    return {
      hasError: true,
      what: 'Violación de segmento (SIGSEGV)',
      where: extractLocation(raw),
      why: 'El programa intentó acceder a una dirección de memoria no asignada o desreferenciar un puntero nulo. Revisa los límites de los arreglos o punteros.',
      rawTrace: raw
    };
  }

  if (raw.includes('SIGFPE') || raw.toLowerCase().includes('floating point exception')) {
    return {
      hasError: true,
      what: 'Excepción de punto flotante (SIGFPE)',
      where: extractLocation(raw),
      why: 'Ocurrió una operación aritmética prohibida, usualmente una división entre cero o un módulo por cero.',
      rawTrace: raw
    };
  }

  if (raw.includes('SIGXCPU') || raw.toLowerCase().includes('time limit exceeded')) {
    return {
      hasError: true,
      what: 'Límite de tiempo excedido',
      where: extractLocation(raw),
      why: 'El proceso tardó más del tiempo máximo permitido de CPU. Verifica la existencia de bucles infinitos o la complejidad asintótica del algoritmo.',
      rawTrace: raw
    };
  }

  // 2. Python Exceptions
  if (lang.includes('py') || raw.includes('Traceback (most recent call last)')) {
    const pyLocation = extractPythonLocation(raw);

    if (raw.includes('ZeroDivisionError')) {
      return {
        hasError: true,
        what: 'División entre cero (ZeroDivisionError)',
        where: pyLocation,
        why: 'El código intentó realizar una división por cero. Asegúrate de que el divisor nunca tome valor 0 (revisa casos de borde o condiciones iniciales).',
        rawTrace: raw
      };
    }
    if (raw.includes('IndexError')) {
      return {
        hasError: true,
        what: 'Índice fuera de rango (IndexError)',
        where: pyLocation,
        why: 'Se intentó acceder a una posición que no existe en la lista o colección. Comprueba los límites de los bucles y si la lista puede estar vacía.',
        rawTrace: raw
      };
    }
    if (raw.includes('KeyError')) {
      return {
        hasError: true,
        what: 'Clave no encontrada (KeyError)',
        where: pyLocation,
        why: 'Se consultó una clave inexistente en un diccionario. Puedes usar .get(clave) o comprobar previamente con "clave in dict".',
        rawTrace: raw
      };
    }
    if (raw.includes('RecursionError')) {
      return {
        hasError: true,
        what: 'Límite de recursión superado (RecursionError)',
        where: pyLocation,
        why: 'El algoritmo excedió la profundidad máxima de llamadas recursivas. Verifica la condición de corte (caso base) y que los parámetros converjan a él.',
        rawTrace: raw
      };
    }
    if (raw.includes('TypeError')) {
      return {
        hasError: true,
        what: 'Incompatibilidad de tipos (TypeError)',
        where: pyLocation,
        why: 'Se aplicó una operación o función sobre tipos de datos no compatibles (por ejemplo sumar texto con números enteros).',
        rawTrace: raw
      };
    }
    if (raw.includes('ValueError')) {
      return {
        hasError: true,
        what: 'Valor inadecuado (ValueError)',
        where: pyLocation,
        why: 'La función recibió un dato con el tipo correcto pero con un valor que no puede procesar (por ejemplo convertir letras a int).',
        rawTrace: raw
      };
    }
    if (raw.includes('AttributeError')) {
      return {
        hasError: true,
        what: 'Atributo no encontrado (AttributeError)',
        where: pyLocation,
        why: 'Se invocó un método o propiedad sobre un objeto que no lo posee, comúnmente una variable que contiene None.',
        rawTrace: raw
      };
    }
    if (raw.includes('SyntaxError') || raw.includes('IndentationError')) {
      return {
        hasError: true,
        what: 'Error sintáctico o de indentación',
        where: pyLocation,
        why: 'El código presenta una estructura no válida para el intérprete. Revisa dos puntos (:), paréntesis sin cerrar o la sangría de los bloques.',
        rawTrace: raw
      };
    }
  }

  // 3. Java Exceptions
  if (lang.includes('java') || raw.includes('Exception in thread')) {
    const javaLocation = extractJavaLocation(raw);

    if (raw.includes('NullPointerException')) {
      return {
        hasError: true,
        what: 'Puntero nulo (NullPointerException)',
        where: javaLocation,
        why: 'Se intentó invocar un método o acceder a un atributo en un objeto que vale null. Inicializa la referencia o valida previamente.',
        rawTrace: raw
      };
    }
    if (raw.includes('ArrayIndexOutOfBoundsException') || raw.includes('IndexOutOfBoundsException')) {
      return {
        hasError: true,
        what: 'Índice fuera de límites en arreglo',
        where: javaLocation,
        why: 'Se intentó acceder a una posición menor a 0 o mayor o igual al tamaño del arreglo.',
        rawTrace: raw
      };
    }
    if (raw.includes('StackOverflowError')) {
      return {
        hasError: true,
        what: 'Desbordamiento de pila (StackOverflowError)',
        where: javaLocation,
        why: 'Demasiadas llamadas anidadas sin retorno. Verifica el caso base de tu función recursiva.',
        rawTrace: raw
      };
    }
  }

  // Fallback genérico para otros errores
  return {
    hasError: true,
    what: 'Error en tiempo de ejecución',
    where: extractLocation(raw),
    why: 'El programa abortó la ejecución con errores en la salida estándar de error.',
    rawTrace: raw
  };
}

function extractPythonLocation(stderr: string): ErrorLocation | undefined {
  // Patrón: File "...", line 12, in <module>
  const regex = /File "([^"]+)", line (\d+)(?:, in ([^\n]+))?/g;
  let match: RegExpExecArray | null;
  let lastMatch: RegExpExecArray | null = null;

  while ((match = regex.exec(stderr)) !== null) {
    lastMatch = match;
  }

  if (lastMatch) {
    const file = lastMatch[1].split('/').pop();
    const line = parseInt(lastMatch[2], 10);
    const functionName = lastMatch[3] ? lastMatch[3].trim() : undefined;
    return {
      file,
      line: isNaN(line) ? undefined : line,
      functionName,
      rawLocation: `Línea ${line}${functionName ? ` en ${functionName}` : ''}`
    };
  }

  return undefined;
}

function extractJavaLocation(stderr: string): ErrorLocation | undefined {
  // Patrón: at Main.calcular(Main.java:15)
  const regex = /at\s+([a-zA-Z0-9_$.]+)\(([a-zA-Z0-9_]+\.java):(\d+)\)/;
  const match = regex.exec(stderr);
  if (match) {
    const functionName = match[1];
    const file = match[2];
    const line = parseInt(match[3], 10);
    return {
      file,
      line: isNaN(line) ? undefined : line,
      functionName,
      rawLocation: `Línea ${line} (${file})`
    };
  }
  return undefined;
}

function extractLocation(stderr: string): ErrorLocation | undefined {
  const lineMatch = /(?:line|línea|linea)\s*(\d+)/i.exec(stderr);
  if (lineMatch) {
    const line = parseInt(lineMatch[1], 10);
    return {
      line: isNaN(line) ? undefined : line,
      rawLocation: `Línea ${line}`
    };
  }
  return undefined;
}
