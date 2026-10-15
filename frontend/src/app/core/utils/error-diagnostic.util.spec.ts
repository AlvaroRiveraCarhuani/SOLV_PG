import { describe, it, expect } from 'vitest';
import { parseErrorDiagnostic, detectWhitespaceDiscrepancy } from './error-diagnostic.util';

describe('error-diagnostic.util', () => {
  describe('detectWhitespaceDiscrepancy', () => {
    it('debe detectar discrepancia cuando solo difieren en saltos de línea Windows CRLF vs POSIX LF', () => {
      const actual = 'hola mundo\r\n';
      const expected = 'hola mundo\n';
      expect(detectWhitespaceDiscrepancy(actual, expected)).toBe(true);
    });

    it('debe detectar discrepancia cuando solo hay espacios en blanco al final', () => {
      const actual = '42  ';
      const expected = '42';
      expect(detectWhitespaceDiscrepancy(actual, expected)).toBe(true);
    });

    it('debe retornar false si las salidas son idénticas', () => {
      expect(detectWhitespaceDiscrepancy('100', '100')).toBe(false);
    });

    it('debe retornar false si la lógica del contenido difiere', () => {
      expect(detectWhitespaceDiscrepancy('100', '101')).toBe(false);
    });
  });

  describe('parseErrorDiagnostic', () => {
    it('debe procesar un ZeroDivisionError en Python extrayendo línea y explicación', () => {
      const stderr = `Traceback (most recent call last):
  File "solution.py", line 14, in division
    resultado = a / b
ZeroDivisionError: division by zero`;

      const diag = parseErrorDiagnostic(stderr, 'python');
      expect(diag.hasError).toBe(true);
      expect(diag.what).toContain('División entre cero');
      expect(diag.where?.line).toBe(14);
      expect(diag.where?.file).toBe('solution.py');
      expect(diag.why).toContain('divisor nunca tome valor 0');
    });

    it('debe procesar un IndexError en Python', () => {
      const stderr = `Traceback (most recent call last):
  File "sol.py", line 8, in buscar
    elem = lista[10]
IndexError: list index out of range`;

      const diag = parseErrorDiagnostic(stderr, 'python');
      expect(diag.hasError).toBe(true);
      expect(diag.what).toContain('Índice fuera de rango');
      expect(diag.where?.line).toBe(8);
      expect(diag.why).toContain('posición que no existe');
    });

    it('debe procesar una señal SIGSEGV de C++', () => {
      const stderr = `Command terminated by signal 11 (SIGSEGV)`;

      const diag = parseErrorDiagnostic(stderr, 'cpp');
      expect(diag.hasError).toBe(true);
      expect(diag.what).toContain('Violación de segmento (SIGSEGV)');
      expect(diag.why).toContain('puntero nulo');
    });

    it('debe procesar NullPointerException en Java', () => {
      const stderr = `Exception in thread "main" java.lang.NullPointerException
        at Main.calcular(Main.java:23)`;

      const diag = parseErrorDiagnostic(stderr, 'java');
      expect(diag.hasError).toBe(true);
      expect(diag.what).toContain('Puntero nulo (NullPointerException)');
      expect(diag.where?.line).toBe(23);
      expect(diag.where?.file).toBe('Main.java');
    });

    it('debe retornar hasError=false para cadenas vacías', () => {
      const diag = parseErrorDiagnostic('');
      expect(diag.hasError).toBe(false);
    });
  });
});
