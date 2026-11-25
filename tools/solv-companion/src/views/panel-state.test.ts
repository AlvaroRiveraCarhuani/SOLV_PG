import assert from 'node:assert/strict';
import { test } from 'node:test';
import { AdvancePreview } from '../domain/advance';
import {
  SOLV_READONLY_PATTERNS,
  computeReadonlyIncludeDecision
} from '../workspace/readonly-lock';
import {
  advanceFileActionText,
  applySelection,
  buildChecklist,
  deriveDeliveryState,
  diffTitleText,
  excerptText,
  rubricBadgeText,
  rubricTitleText,
  selectedPaths,
  submissionActionText
} from './panel-state';

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

test('el extracto de la consigna saca las marcas de markdown', () => {
  assert.equal(
    excerptText('## Contexto\n\n- **Objetivo**: construir la API.\n- Usar `Express`.'),
    'Contexto Objetivo: construir la API. Usar Express.'
  );
});

test('el extracto corta en un espacio y avisa con puntos suspensivos', () => {
  const excerpt = excerptText('palabra '.repeat(120), 100);
  assert.ok(excerpt.length <= 101);
  assert.ok(excerpt.endsWith('…'));
  assert.ok(!excerpt.includes('palabr…'));
});

test('un archivo nuevo ofrece ver contenido y no diferencias', () => {
  assert.equal(advanceFileActionText('new'), 'Ver contenido');
  assert.equal(advanceFileActionText('modified'), 'Ver diferencias');
  assert.equal(advanceFileActionText('unchanged'), '');
});

test('el diff de un archivo nuevo se anuncia como archivo nuevo', () => {
  assert.equal(diffTitleText('hola.js', true), 'hola.js — Archivo nuevo del avance');
  assert.equal(
    diffTitleText('src/app.ts', false),
    'src/app.ts — Tu versión y el avance del docente'
  );
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

test('los criterios de rúbrica calculan puntos y niveles correctamente', () => {
  const criteria = [
    {
      id: 'c1',
      name: 'Diseño y Tipado',
      weight: 40,
      description: 'Tipado estricto',
      levels: [
        { name: 'Excelente', score: 100, description: 'Todo tipado' },
        { name: 'Regular', score: 50, description: 'Usa any' }
      ]
    },
    {
      id: 'c2',
      name: 'Calidad de Código',
      weight: 60,
      description: 'Buenas prácticas'
    }
  ];
  const total = criteria.reduce((sum, item) => sum + item.weight, 0);
  assert.equal(total, 100);
  const levels = criteria[0].levels!;
  const excellentPts = Math.round((levels[0].score / 100) * criteria[0].weight * 10) / 10;
  const regularPts = Math.round((levels[1].score / 100) * criteria[0].weight * 10) / 10;
  assert.equal(excellentPts, 40);
  assert.equal(regularPts, 20);
});

test('los textos y badges se adaptan al propósito pedagógico (examen vs práctica)', () => {
  assert.equal(rubricBadgeText('exam'), 'Rúbrica de evaluación');
  assert.equal(rubricBadgeText('class'), 'Rúbrica de práctica');
  assert.equal(rubricBadgeText(undefined), 'Rúbrica de práctica');

  assert.equal(rubricTitleText('exam'), 'Rúbrica de Evaluación');
  assert.equal(rubricTitleText('class'), 'Rúbrica de Práctica');
  assert.equal(rubricTitleText(undefined), 'Rúbrica de Práctica');

  assert.equal(submissionActionText('exam'), '$(cloud-upload) Entregar Examen');
  assert.equal(submissionActionText('class'), '$(cloud-upload) Entregar Práctica');
  assert.equal(submissionActionText(undefined), '$(cloud-upload) Entregar Práctica');
});

test('readonly-lock activa y limpia files.readonlyInclude de forma idempotente sin pisar reglas ajenas', () => {
  const lockFresh = computeReadonlyIncludeDecision(true, undefined, false);
  assert.equal(lockFresh.changed, true);
  assert.equal(lockFresh.nextOwned, true);
  assert.deepEqual(lockFresh.nextValue, SOLV_READONLY_PATTERNS);

  const lockAgain = computeReadonlyIncludeDecision(true, lockFresh.nextValue, lockFresh.nextOwned);
  assert.equal(lockAgain.changed, false);
  assert.equal(lockAgain.nextOwned, true);

  const unlockFresh = computeReadonlyIncludeDecision(false, lockFresh.nextValue, true);
  assert.equal(unlockFresh.changed, true);
  assert.equal(unlockFresh.nextOwned, false);
  assert.equal(unlockFresh.nextValue, undefined);

  const unlockAgain = computeReadonlyIncludeDecision(false, undefined, false);
  assert.equal(unlockAgain.changed, false);

  const withUserPattern = computeReadonlyIncludeDecision(true, { 'vendor/**': true }, false);
  assert.equal(withUserPattern.changed, true);
  assert.deepEqual(withUserPattern.nextValue, {
    'vendor/**': true,
    ...SOLV_READONLY_PATTERNS
  });

  const unlockPreservesUserPattern = computeReadonlyIncludeDecision(
    false,
    withUserPattern.nextValue,
    withUserPattern.nextOwned
  );
  assert.equal(unlockPreservesUserPattern.changed, true);
  assert.equal(unlockPreservesUserPattern.nextOwned, false);
  assert.deepEqual(unlockPreservesUserPattern.nextValue, { 'vendor/**': true });

  const foreignOnly = computeReadonlyIncludeDecision(false, { 'vendor/**': true }, false);
  assert.equal(foreignOnly.changed, false);
  assert.deepEqual(foreignOnly.nextValue, { 'vendor/**': true });
});
