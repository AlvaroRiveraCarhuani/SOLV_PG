import assert from 'node:assert/strict';
import { test } from 'node:test';
import { buildKeystrokeEvent, classifyChange } from './keystroke-classifier';

test('una tecla suelta es inserción', () => {
  assert.equal(classifyChange('a', 0, 50), 'insert');
});

test('un borrado no se confunde con pegado', () => {
  assert.equal(classifyChange('', 10, 50), 'delete');
});

test('un bloque grande es pegado aunque reemplace texto', () => {
  assert.equal(classifyChange('x'.repeat(80), 12, 50), 'paste');
});

test('un reemplazo multilínea mediano es pegado', () => {
  assert.equal(classifyChange('uno\ndos\ntres\ncuatro', 4, 50), 'paste');
});

test('el evento de pegado marca su origen y conserva la posición', () => {
  const event = buildKeystrokeEvent('y'.repeat(60), 0, 120, 9000, 50);
  assert.equal(event.event_type, 'paste');
  assert.equal(event.paste_source_detected, true);
  assert.equal(event.position, 120);
  assert.equal(event.timestamp_ms, 9000);
});
