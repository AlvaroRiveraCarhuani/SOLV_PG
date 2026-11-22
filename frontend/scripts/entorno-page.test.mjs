import { readFileSync } from 'node:fs';

import { JSDOM } from 'jsdom';
import { describe, expect, it } from 'vitest';

const HTML = readFileSync(new URL('../public/entorno.html', import.meta.url), 'utf8');
const SCRIPT = readFileSync(new URL('../public/entorno.js', import.meta.url), 'utf8');

const ESPERA_MS = 90_000;
const EJERCICIO = '/student/exercises/08d80113-cc02-44cb-91fa-08e671453675';

/**
 * Abre la página con el query indicado y corre su script con el temporizador
 * capturado, para no esperar el minuto y medio real.
 */
function abrir(consulta = '') {
  const dom = new JSDOM(HTML, {
    url: `https://solv.test/entorno.html${consulta}`,
    runScripts: 'outside-only',
  });
  const { window } = dom;

  let temporizador = null;
  window.setTimeout = (fn, ms) => {
    temporizador = { fn, ms };
    return 1;
  };

  window.eval(SCRIPT);

  return {
    document: window.document,
    espera: () => temporizador,
    vencerEspera: () => temporizador.fn(),
  };
}

describe('pestaña de preparación del entorno', () => {
  it('carga su script y tiene los elementos que usa', () => {
    expect(HTML).toContain('src="/entorno.js"');
    const { document } = abrir();
    for (const id of ['rueda', 'titulo', 'detalle', 'volver', 'volver-enlace']) {
      expect(document.getElementById(id), `falta #${id}`).not.toBeNull();
    }
  });

  it('vuelve a la ruta del ejercicio que recibe en volver', () => {
    const { document } = abrir(`?volver=${encodeURIComponent(EJERCICIO)}`);
    expect(document.getElementById('volver-enlace').getAttribute('href')).toBe(EJERCICIO);
  });

  it('no acepta destinos fuera del sitio y cae en /student', () => {
    for (const parametro of ['', 'https://ejemplo.com', '//ejemplo.com', '/\\ejemplo.com']) {
      const consulta = parametro ? `?volver=${encodeURIComponent(parametro)}` : '';
      const { document } = abrir(consulta);
      expect(document.getElementById('volver-enlace').getAttribute('href'), parametro).toBe('/student');
    }
  });

  it('gira sin ofrecer el enlace mientras no vence la espera', () => {
    const { document, espera } = abrir(`?volver=${encodeURIComponent(EJERCICIO)}`);
    expect(espera().ms).toBe(ESPERA_MS);
    expect(document.getElementById('rueda').hidden).toBe(false);
    expect(document.getElementById('volver').hidden).toBe(true);
    expect(document.getElementById('titulo').textContent).toBe('Preparando tu entorno');
  });

  it('al vencer la espera muestra el aviso y el enlace de vuelta', () => {
    const { document, vencerEspera } = abrir(`?volver=${encodeURIComponent(EJERCICIO)}`);

    vencerEspera();

    expect(document.getElementById('rueda').hidden).toBe(true);
    expect(document.getElementById('volver').hidden).toBe(false);
    expect(document.getElementById('titulo').textContent).toBe('El entorno no terminó de abrirse');
    expect(document.getElementById('detalle').textContent).toContain('arranque');
    expect(document.getElementById('volver-enlace').getAttribute('href')).toBe(EJERCICIO);
  });
});
