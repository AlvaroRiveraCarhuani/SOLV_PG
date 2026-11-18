import assert from 'node:assert/strict';
import { test } from 'node:test';
import { AdvanceClient } from './advance-client';
import { SolvClient } from '../client/solv-client';
import { SolvSettings } from '../config/solv-settings';

function buildClient(): {
  advances: AdvanceClient;
  bodies: unknown[];
  requests: Array<{ url: string; method?: string }>;
} {
  const settings = {
    apiBaseUrl: 'https://solv.dedyn.io',
    getTokenEnv: async () => 'token-de-prueba'
  } as unknown as SolvSettings;
  const bodies: unknown[] = [];
  const requests: Array<{ url: string; method?: string }> = [];
  globalThis.fetch = (async (url: string, init: { body?: string; method?: string }) => {
    bodies.push(init.body ? JSON.parse(init.body) : undefined);
    requests.push({ url, method: init.method });
    return { ok: true, status: 200, text: async () => '{"data":{"files":[]}}' };
  }) as unknown as typeof fetch;
  return { advances: new AdvanceClient(new SolvClient(settings)), bodies, requests };
}

test('emitir con archivos elegidos envía la selección', async () => {
  const original = globalThis.fetch;
  const { advances, bodies } = buildClient();
  try {
    await advances.emit({
      sessionId: 's-1',
      exerciseId: 'e-1',
      scope: 'class',
      selectedFiles: ['src/a.ts', 'README.md']
    });
  } finally {
    globalThis.fetch = original;
  }
  assert.deepEqual(bodies[0], {
    exercise_id: 'e-1',
    title: '',
    scope: 'class',
    student_ids: [],
    selected_files: ['src/a.ts', 'README.md']
  });
});

test('emitir todo el proyecto no manda una selección', async () => {
  const original = globalThis.fetch;
  const { advances, bodies } = buildClient();
  try {
    await advances.emit({ sessionId: 's-1', exerciseId: 'e-1', scope: 'class' });
  } finally {
    globalThis.fetch = original;
  }
  assert.deepEqual(bodies[0], {
    exercise_id: 'e-1',
    title: '',
    scope: 'class',
    student_ids: [],
    selected_files: []
  });
});

test('emitir con título personalizado envía el título', async () => {
  const original = globalThis.fetch;
  const { advances, bodies } = buildClient();
  try {
    await advances.emit({
      sessionId: 's-1',
      exerciseId: 'e-1',
      title: 'Paso 2: rutas del servicio',
      scope: 'class'
    });
  } finally {
    globalThis.fetch = original;
  }
  assert.deepEqual(bodies[0], {
    exercise_id: 'e-1',
    title: 'Paso 2: rutas del servicio',
    scope: 'class',
    student_ids: [],
    selected_files: []
  });
});

test('deshacer manda solo los archivos elegidos', async () => {
  const original = globalThis.fetch;
  const { advances, bodies, requests } = buildClient();
  try {
    await advances.rollback('a-1', ['src/a.ts', 'README.md']);
  } finally {
    globalThis.fetch = original;
  }
  assert.equal(requests[0].method, 'POST');
  assert.match(requests[0].url, /\/api\/v1\/class-advances\/a-1\/rollback$/);
  assert.deepEqual(bodies[0], { paths: ['src/a.ts', 'README.md'] });
});

test('el inventario para deshacer se pide con GET', async () => {
  const original = globalThis.fetch;
  const { advances, requests } = buildClient();
  try {
    await advances.rollbackFiles('a-1');
  } finally {
    globalThis.fetch = original;
  }
  assert.equal(requests[0].method, 'GET');
  assert.match(requests[0].url, /\/api\/v1\/class-advances\/a-1\/rollback$/);
});

test('abrir la clase manda la práctica cuando se conoce', async () => {
  const original = globalThis.fetch;
  const { advances, bodies } = buildClient();
  try {
    await advances.openSession('m-1', 90, 'e-1');
  } finally {
    globalThis.fetch = original;
  }
  assert.deepEqual(bodies[0], { subject_id: 'm-1', duration_minutes: 90, exercise_id: 'e-1' });
});

test('abrir la clase sin práctica no inventa exercise_id', async () => {
  const original = globalThis.fetch;
  const { advances, bodies } = buildClient();
  try {
    await advances.openSession('m-1', 60);
  } finally {
    globalThis.fetch = original;
  }
  assert.deepEqual(bodies[0], { subject_id: 'm-1', duration_minutes: 60 });
});
