import assert from 'node:assert/strict';
import { test } from 'node:test';
import { SolvClient } from './solv-client';
import { SolvSettings } from '../config/solv-settings';

function clientWith(submissionId: string): SolvClient {
  const settings = {
    apiBaseUrl: 'https://solv.dedyn.io',
    submissionId,
    getTokenEnv: async () => 'token-de-prueba'
  } as unknown as SolvSettings;
  return new SolvClient(settings);
}

function stubFetch(body: string, status: number): void {
  globalThis.fetch = (async () => ({
    ok: false,
    status,
    text: async () => body
  })) as unknown as typeof fetch;
}

test('un 404 sin cuerpo JSON informa la ruta y no la entrega', async () => {
  const original = globalThis.fetch;
  stubFetch('404 page not found', 404);
  try {
    await assert.rejects(clientWith('s-1').fetchComments(), (error: Error) => {
      assert.match(error.message, /HTTP 404/);
      assert.match(error.message, /\/api\/v1\/student\/submissions\/s-1\/comments/);
      assert.doesNotMatch(error.message, /entrega no existe/i);
      return true;
    });
  } finally {
    globalThis.fetch = original;
  }
});

test('el texto del backend manda cuando el 404 trae mensaje', async () => {
  const original = globalThis.fetch;
  stubFetch(JSON.stringify({ message: 'La entrega no existe en esta institución' }), 404);
  try {
    await assert.rejects(clientWith('s-1').fetchComments(), (error: Error) => {
      assert.equal(error.message, 'La entrega no existe en esta institución');
      return true;
    });
  } finally {
    globalThis.fetch = original;
  }
});
