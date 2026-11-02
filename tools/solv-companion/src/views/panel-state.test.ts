import assert from 'node:assert/strict';
import { test } from 'node:test';
import { AdvancePreview } from '../domain/advance';
import { applySelection, buildChecklist, deriveDeliveryState, selectedPaths } from './panel-state';

function preview(files: Record<string, string>, added: string[], overwritten: string[]): AdvancePreview {
  return {
    advance_id: 'a-1',
    exercise_id: 'e-1',
    file_count: Object.keys(files).length,
    added,
    overwritten,
    files
  };
}

test('un archivo nuevo queda marcado por defecto', () => {
  const checklist = buildChecklist(preview({ 'src/nuevo.ts': 'contenido' }, ['src/nuevo.ts'], []), {});
  assert.equal(checklist[0].status, 'new');
  assert.equal(checklist[0].selected, true);
});

test('un archivo distinto al local se marca como modificado', () => {
  const checklist = buildChecklist(
    preview({ 'src/app.ts': 'docente' }, [], ['src/app.ts']),
    { 'src/app.ts': 'alumno' }
  );
  assert.equal(checklist[0].status, 'modified');
  assert.equal(checklist[0].selected, true);
});

test('un archivo idéntico queda sin cambios y desmarcado', () => {
  const checklist = buildChecklist(
    preview({ 'src/app.ts': 'igual' }, [], ['src/app.ts']),
    { 'src/app.ts': 'igual' }
  );
  assert.equal(checklist[0].status, 'unchanged');
  assert.equal(checklist[0].selected, false);
});

test('un archivo sobrescrito que no existe localmente cuenta como modificado', () => {
  const checklist = buildChecklist(
    preview({ 'src/app.ts': 'docente' }, [], ['src/app.ts']),
    {}
  );
  assert.equal(checklist[0].status, 'modified');
});

test('la selección del alumno se refleja en las rutas a aplicar', () => {
  const checklist = buildChecklist(
    preview({ 'a.ts': 'uno', 'b.ts': 'dos' }, ['a.ts', 'b.ts'], []),
    {}
  );
  const updated = applySelection(checklist, ['b.ts']);
  assert.deepEqual(selectedPaths(updated), ['b.ts']);
});

test('el checklist sale ordenado por ruta', () => {
  const checklist = buildChecklist(
    preview({ 'z.ts': 'z', 'a.ts': 'a', 'm.ts': 'm' }, ['z.ts', 'a.ts', 'm.ts'], []),
    {}
  );
  assert.deepEqual(checklist.map(file => file.path), ['a.ts', 'm.ts', 'z.ts']);
});

test('una entrega con nota queda calificada', () => {
  assert.equal(deriveDeliveryState(85), 'graded');
});

test('una entrega sin nota queda entregada', () => {
  assert.equal(deriveDeliveryState(null), 'submitted');
  assert.equal(deriveDeliveryState(undefined), 'submitted');
});
