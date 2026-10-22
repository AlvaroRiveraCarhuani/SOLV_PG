/**
 * Helpers de API para los specs E2E del Juez Virtual.
 *
 * El backend aplica un límite global por IP (100 req/min con burst 20). Una
 * suite multi-rol que prepara materia, módulos, ejercicios y entregas en
 * paralelo puede agotar ese bucket y recibir 429 sin que el producto tenga un
 * fallo. Estos helpers reintentan ante un 429 en lugar de romper el spec por
 * infraestructura.
 */

export function authHeaders(token) {
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  };
}

const RATE_LIMIT_RETRY_DELAY_MS = 2000;
const RATE_LIMIT_MAX_ATTEMPTS = 10;

/**
 * Ejecuta una petición HTTP contra la API reintentando cuando el backend
 * responde 429 (límite global por IP). Devuelve la respuesta final.
 */
export async function apiRequest(request, token, method, url, data) {
  let response = null;

  for (let attempt = 1; attempt <= RATE_LIMIT_MAX_ATTEMPTS; attempt++) {
    response = await request.fetch(url, { method, headers: authHeaders(token), data });
    if (response.status() !== 429) {
      return response;
    }
    await new Promise((resolve) => setTimeout(resolve, RATE_LIMIT_RETRY_DELAY_MS));
  }

  return response;
}

/**
 * Igual que apiRequest, pero devuelve el cuerpo JSON parseado y falla con un
 * mensaje claro cuando la respuesta no es exitosa.
 */
export async function apiJson(request, token, method, url, data, { expectStatus = [200, 201] } = {}) {
  const response = await apiRequest(request, token, method, url, data);

  if (!expectStatus.includes(response.status())) {
    const body = await response.text();
    throw new Error(`La petición ${method} ${url} respondió ${response.status()}: ${body.slice(0, 400)}`);
  }

  const json = await response.json().catch(() => ({}));
  return json?.data ?? json;
}

/**
 * Abre el mapa curricular del estudiante y espera la respuesta del backend.
 * Si el backend limita la tasa, reintenta con el botón "Actualizar" del propio
 * mapa en lugar de recargar la aplicación.
 */
export async function openCurricularMap(page, courseId, { timeout = 45000, attempts = 8, isReady } = {}) {
  const mapUrl = `/api/v1/student/courses/${courseId}/map`;
  const map = page.locator('curricular-map');

  const ready =
    isReady ??
    (async () => await map.locator('.module-card').first().isVisible().catch(() => false));

  for (let attempt = 0; attempt < attempts; attempt++) {
    const okResponse = page
      .waitForResponse((res) => res.url().includes(mapUrl) && res.status() === 200, { timeout })
      .catch(() => null);

    if (attempt === 0) {
      await page.goto(`/student/courses/${courseId}`, { waitUntil: 'domcontentloaded' });
    } else {
      await page.waitForTimeout(RATE_LIMIT_RETRY_DELAY_MS);
      const refresh = map.locator('button.btn-refresh');
      if (await refresh.count()) {
        await refresh.click();
      } else {
        await page.reload({ waitUntil: 'domcontentloaded' });
      }
    }

    await okResponse;
    await page.waitForTimeout(500);

    if (await ready()) {
      return map;
    }
  }

  throw new Error(`El mapa curricular de ${courseId} no quedó listo tras ${attempts} intentos`);
}
