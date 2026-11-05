// Gate de terminología de modalidades.
//
// Contrato: los identificadores `JUEZ_EFIMERO`, `IDE_PERSISTENTE` y `AMBOS` no se
// traducen, y sus nombres visibles («Entorno de Evaluación», «Entorno de
// Desarrollo», «Entorno de Desarrollo y Evaluación») salen del catálogo
// `src/app/shared/constants/environment-types.constants.ts` a través del pipe
// `environmentLabel`.
//
// El gate no compara contra una lista de palabras conocidas. Clasifica cada
// texto por su forma:
//   1) `official`     — nombre oficial de una modalidad; solo el catálogo lo declara.
//   2) `engine`       — «Juez Virtual», nombre del motor; válido en prosa y en
//                       encabezados, nunca como etiqueta de modalidad.
//   3) `non-official` — cualquier otra frase con forma de modalidad o de motor
//                       («Entorno de Programación», «IDE Persistente»,
//                       «Evaluador Automático»). No es nombre válido en ningún sitio.
//   4) `other`        — prosa, conceptos ajenos («Entorno de trabajo»,
//                       «Espacio de trabajo») e identificadores.
//
// La clasificación se hace sobre textos normalizados (minúsculas, sin acentos,
// sin comillas ni puntuación de borde), así que «Entorno de evaluación» y
// «ENTORNO DE EVALUACION» caen en la misma clase que «Entorno de Evaluación».
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve, sep } from 'node:path';
import { pathToFileURL } from 'node:url';
import { parseTemplate } from '@angular/compiler';
import ts from 'typescript';

const ROOT = process.cwd();
const REPO_ROOT = join(ROOT, '..');
const SRC_APP = join(ROOT, 'src', 'app');
const CATALOG_FILE = join(SRC_APP, 'shared', 'constants', 'environment-types.constants.ts');
const PIPE_FILE = join(SRC_APP, 'shared', 'pipes', 'environment-label.pipe.ts');
const AGENTS_FILE = join(REPO_ROOT, 'AGENTS.md');
const SKILL_FILE = join(REPO_ROOT, '.agents', 'skills', 'solv-context', 'SKILL.md');
const DOCS_DIR = join(REPO_ROOT, 'docs');

/** Decisión que fija la terminología vigente. */
export const POLICY_ADR = 'ADR-067';
export const POLICY_ADR_NUMBER = 67;

/** Correspondencia oficial. El catálogo debe coincidir exactamente. */
export const CANONICAL_TERMINOLOGY = {
  JUEZ_EFIMERO: 'Entorno de Evaluación',
  IDE_PERSISTENTE: 'Entorno de Desarrollo',
  AMBOS: 'Entorno de Desarrollo y Evaluación',
};

/** Identificadores de contrato presentes en la documentación. */
export const CONTRACT_IDENTIFIERS = Object.keys(CANONICAL_TERMINOLOGY);

const IDENTIFIER_BY_OFFICIAL_KEY = new Map(
  Object.entries(CANONICAL_TERMINOLOGY).map(([identifier, label]) => [normalizeTermLabel(label), identifier])
);

/**
 * Nombres que nombran el motor. «Juez Virtual» es el nombre vigente; el resto
 * son variantes antiguas que solo pueden aparecer en un registro histórico.
 */
const ENGINE_NAME_KEY = 'juez virtual';

const MODALITY_HEADS = new Set(['entorno', 'ambiente', 'modalidad', 'espacio']);
const ENGINE_HEADS = new Set(['juez', 'evaluador', 'corrector', 'verificador', 'validador']);
const IDE_TAILS = new Set(['persistente', 'efimero', 'virtual', 'remoto', 'local', 'web', 'compartido']);

/**
 * Conceptos con forma «Entorno de …» que no son modalidades. Se listan para no
 * confundir prosa («Entorno de trabajo», «Entorno de la práctica») con una
 * etiqueta. Cualquier otra forma «Entorno de X» se trata como etiqueta.
 */
const OTHER_CONCEPT_HEADS = new Set([
  'trabajo',
  'practica',
  'practicas',
  'laboratorio',
  'aprendizaje',
  'ejecucion',
  'prueba',
  'pruebas',
  'clase',
  'curso',
  'asignatura',
  'materia',
  'estudiante',
  'docente',
  'usuario',
  'equipo',
  'produccion',
  'despliegue',
  'revision',
  'entrega',
  'entorno',
  'construccion',
  'compilacion',
  'instalacion',
  'integracion',
  'validacion',
  'empaquetado',
]);

/**
 * Palabras que convierten una frase con forma «Entorno de …» en nombre de
 * modalidad. Sin una de ellas la frase es prosa descriptiva («Entorno
 * interactivo web», «Entorno y entrega»).
 */
const MODALITY_QUALIFIERS = new Set([
  'desarrollo',
  'evaluacion',
  'programacion',
  'codigo',
  'codigos',
  'correccion',
  'calificacion',
  'compilacion',
  'dual',
  'doble',
  'mixto',
  'combinado',
  'integrado',
]);

/** Palabras que convierten una frase con forma «Juez …» en nombre de motor. */
const ENGINE_QUALIFIERS = new Set([
  'virtual',
  'automatico',
  'automatizada',
  'efimero',
  'evaluador',
  'juez',
  'evaluacion',
  'codigo',
  'soluciones',
  'pruebas',
  'correccion',
  'calificacion',
]);

/** Palabras que pueden formar parte de una etiqueta de modalidad. */
const TERM_WORDS = new Set([
  'de',
  'del',
  'la',
  'el',
  'los',
  'las',
  'y',
  'e',
  'evaluador',
  'juez',
  'desarrollo',
  'evaluacion',
  'programacion',
  'codigo',
  'codigos',
  'correccion',
  'calificacion',
  'compilacion',
  'entrega',
  'revision',
  'trabajo',
  'dual',
  'combinado',
  'doble',
  'mixto',
  'integrado',
  'virtual',
  'automatico',
  'automatizada',
  'efimero',
  'persistente',
  'remoto',
  'local',
  'web',
  'sandbox',
  'aislado',
  'seguro',
  'interactivo',
  'colaborativo',
  'guiado',
  'autonomo',
]);

/**
 * Registro de documentos que conservan nombres anteriores porque registran el
 * estado previo a la política. La excepción es por regla: dentro de estos
 * documentos siguen vigentes las reglas de forma y la comprobación de
 * correspondencia; solo se exime la mención entrecomillada de un nombre antiguo.
 * Toda entrada debe superar `validateExceptionRegistry`.
 */
export const GOVERNANCE_RECORDS = [
  {
    path: 'docs/GOBERNANZA/AUDITORIA_TERMINOLOGIA.md',
    reason: 'Auditoría que documenta con citas el estado anterior a la política.',
    policyAdr: POLICY_ADR,
    allows: 'Nombres antiguos entrecomillados en tablas y prosa del registro.',
    stillDetects: 'Un sinónimo de modalidad nuevo en celda, encabezado o glosario, y toda desalineación de identificador.',
  },
  {
    path: 'docs/GOBERNANZA/PROPUESTA_VOCABULARIO_DOMINIO.md',
    reason: 'Propuesta de vocabulario que compara candidatos descartados.',
    policyAdr: POLICY_ADR,
    allows: 'Filas de comparación con nombres candidatos del motor.',
    stillDetects: 'Un sinónimo de modalidad nuevo en celda, encabezado o glosario, y toda desalineación de identificador.',
  },
];

const MAX_GOVERNANCE_RECORDS = 4;
const HISTORICAL_MARKER = /terminolog[ií]a vigente:?[\s*_]*ADR-\d+|sustituid[oa] por\s+ADR-\d+/i;

export function normalizeKey(value) {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[`"'«»“”]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase();
}

/** Normaliza una etiqueta completa: sin numeración, sin paréntesis finales ni dos puntos. */
export function normalizeTermLabel(value) {
  return normalizeKey(value)
    .replace(/^[\s>*#|\-\d.)]+/, '')
    .replace(/\([^)]*\)\s*$/, '')
    .replace(/[:;,.]+$/, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function stripLeadingArticle(words) {
  const articles = new Set(['de', 'del', 'la', 'el', 'los', 'las']);
  let index = 0;
  while (index < words.length && articles.has(words[index])) index += 1;
  return words.slice(index);
}

/**
 * Clasifica un texto completo por su forma.
 * @returns {{kind: 'official'|'engine'|'non-official'|'other', category?: string, identifier?: string, label?: string}}
 */
export function classifyTerm(value) {
  const key = normalizeTermLabel(value);
  if (!key) return { kind: 'other' };

  if (IDENTIFIER_BY_OFFICIAL_KEY.has(key)) {
    return {
      kind: 'official',
      identifier: IDENTIFIER_BY_OFFICIAL_KEY.get(key),
      label: CANONICAL_TERMINOLOGY[IDENTIFIER_BY_OFFICIAL_KEY.get(key)],
    };
  }

  if (key === ENGINE_NAME_KEY) return { kind: 'engine', label: 'Juez Virtual', raw: normalize(value) };

  const words = key.split(' ');
  const head = words[0];
  const connector = words[1] === 'de' || words[1] === 'del' ? words[1] : null;
  const afterConnector = connector ? words.slice(2) : words.slice(1);
  const rest = stripLeadingArticle(afterConnector);
  if (!rest.length) return { kind: 'other' };

  // «del» contrae el artículo: «Entorno del Despachador» es una referencia
  // descriptiva, no un nombre de modalidad.
  if (connector === 'del') return { kind: 'other' };

  const significant = rest.filter((word) => !['de', 'del', 'la', 'el', 'los', 'las', 'y', 'e'].includes(word));
  if (!significant.length) return { kind: 'other' };

  // «Entorno de trabajo», «Espacio de trabajo», «Entorno de la práctica».
  if (OTHER_CONCEPT_HEADS.has(significant[0])) return { kind: 'other' };

  // «Entorno de <sustantivo>» es un nombre de modalidad aunque el sustantivo no
  // esté en el vocabulario conocido: falla cerrado y obliga a usar el catálogo o
  // a declarar el concepto ajeno. Un artículo intermedio («Entorno de la X»)
  // indica una referencia descriptiva y queda fuera.
  if (connector) {
    const hasArticle = ['la', 'el', 'los', 'las'].includes(afterConnector[0]);
    const singleNoun = significant.length === 1 && /^[a-z]+$/.test(significant[0]);
    if (!hasArticle && singleNoun) {
      if (MODALITY_HEADS.has(head)) {
        return { kind: 'non-official', category: 'modalidad', label: normalizeTermLabel(value), raw: normalize(value) };
      }
      if (ENGINE_HEADS.has(head)) {
        return { kind: 'non-official', category: 'motor', label: normalizeTermLabel(value), raw: normalize(value) };
      }
    }
  }

  if (head === 'ide') {
    if (significant.length === 1 && IDE_TAILS.has(significant[0])) {
      return { kind: 'non-official', category: 'herramienta', label: normalizeTermLabel(value), raw: normalize(value) };
    }
    return { kind: 'other' };
  }

  if (!significant.every((word) => TERM_WORDS.has(word))) return { kind: 'other' };

  if (MODALITY_HEADS.has(head)) {
    if (!significant.some((word) => MODALITY_QUALIFIERS.has(word))) return { kind: 'other' };
    return { kind: 'non-official', category: 'modalidad', label: normalizeTermLabel(value), raw: normalize(value) };
  }
  if (ENGINE_HEADS.has(head)) {
    if (!significant.some((word) => ENGINE_QUALIFIERS.has(word))) return { kind: 'other' };
    return { kind: 'non-official', category: 'motor', label: normalizeTermLabel(value), raw: normalize(value) };
  }
  return { kind: 'other' };
}

function describeTerm(term) {
  const written = term.raw ?? term.label;
  if (term.kind === 'official') return `nombre oficial de modalidad «${term.label}»`;
  if (term.kind === 'engine') return `nombre del motor «${written}»`;
  if (term.category === 'motor') return `nombre de motor no oficial «${written}»`;
  if (term.category === 'herramienta') return `nombre de herramienta usado como modalidad «${written}»`;
  return `nombre de modalidad no oficial «${written}»`;
}

function walk(dir, extensions, out = []) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const info = statSync(path);
    if (info.isDirectory()) walk(path, extensions, out);
    else if (extensions.some((extension) => path.endsWith(extension))) out.push(path);
  }
  return out;
}

function normalize(value) {
  return String(value ?? '').replace(/\s+/g, ' ').trim();
}

/** Quita el sufijo explicativo y los dos puntos de una etiqueta de glosario. */
function normalizeModalityName(value) {
  return normalize(value)
    .replace(/^[`"«']+|[`"»']+$/g, '')
    .replace(/:$/, '')
    .replace(/\s*\([^)]*\)\s*$/, '');
}

const LABEL_SPAN_MAX_WORDS = 5;

/**
 * Extrae tramos con forma de etiqueta dentro de una frase. Solo cuenta como
 * inicio una palabra con mayúscula inicial: en prosa, un nombre se escribe como
 * etiqueta («…el Entorno de Programación…»), mientras que el uso común en
 * minúscula («el entorno de desarrollo local») es prosa descriptiva y no se
 * reporta. Devuelve el tramo con sus posiciones para poder revisar su contexto.
 */
export function findLabelSpans(text) {
  const source = String(text ?? '');
  const tokens = [...source.matchAll(/[\p{L}\p{N}_]+/gu)].map((match) => ({
    word: match[0],
    start: match.index,
    end: match.index + match[0].length,
  }));
  const found = new Map();
  const reportedStarts = new Set();

  tokens.forEach((token, index) => {
    if (!/^\p{Lu}/u.test(token.word)) return;
    // Un nombre permitido puede aparecer seguido de más palabras («Entorno de
    // Evaluación (JUEZ_EFIMERO): ¿…»). Desde que el tramo es un nombre válido,
    // las ampliaciones no se reportan.
    let allowed = false;
    for (let length = 2; length <= LABEL_SPAN_MAX_WORDS; length += 1) {
      const slice = tokens.slice(index, index + length);
      if (slice.length < length) break;
      // Un nombre no atraviesa marcas de formato ni puntuación: si entre dos
      // palabras hay algo más que espacios («Entorno \*:** `Evaluador»), el
      // tramo no es una etiqueta.
      const previous = slice[slice.length - 2];
      const gap = source.slice(previous.end, slice[slice.length - 1].start);
      if (!/^\s+$/.test(gap)) break;
      const term = classifyTerm(slice.map((item) => item.word).join(' '));
      if (term.kind === 'official' || term.kind === 'engine') {
        allowed = true;
        continue;
      }
      if (allowed || reportedStarts.has(index) || term.kind !== 'non-official') continue;
      const key = `${term.kind}:${term.label}`;
      const start = token.start;
      const end = slice[slice.length - 1].end;
      reportedStarts.add(index);
      if (!found.has(key)) found.set(key, { ...term, value: source.slice(start, end), start, end });
    }
  });

  return [...found.values()];
}

/**
 * Categorías que se buscan dentro de una frase. En la interfaz, el nombre de la
 * herramienta también delata una modalidad («…laboratorio de IDE Persistente»);
 * en documentación de arquitectura el IDE es un componente con nombre propio y
 * solo se revisan los nombres de modalidad.
 */
const PROSE_LABEL_CATEGORIES = new Set(['modalidad', 'herramienta']);

/** Tramos con forma de etiqueta dentro de una frase, filtrados por categoría. */
export function embeddedLabelSpans(text, categories = PROSE_LABEL_CATEGORIES) {
  return findLabelSpans(text).filter((span) => categories.has(span.category));
}

/** Detecta frases con forma de etiqueta dentro de un texto libre (descripciones). */
export function findForbiddenTermsInText(text) {
  const words = String(text ?? '').split(/[^\p{L}\p{N}]+/u).filter(Boolean);
  const found = new Map();
  for (let index = 0; index < words.length; index += 1) {
    const head = normalizeKey(words[index]);
    if (!MODALITY_HEADS.has(head) && !ENGINE_HEADS.has(head) && head !== 'ide') continue;
    for (let length = 2; length <= 5; length += 1) {
      const candidate = words.slice(index, index + length);
      if (candidate.length < length) break;
      const term = classifyTerm(candidate.join(' '));
      if (term.kind === 'non-official') found.set(term.label, term);
    }
  }
  return [...found.values()];
}

function readCatalogEntries(source) {
  const entries = [];
  const entryPattern = /^\s{2}([A-Z_]+):\s*\{([\s\S]*?)\n\s{2}\},?/gm;
  let match;
  while ((match = entryPattern.exec(source)) !== null) {
    const [, key, body] = match;
    entries.push({
      key,
      label: /label:\s*'([^']*)'/.exec(body)?.[1],
      short: /short:\s*'([^']*)'/.exec(body)?.[1],
      description: /description:\s*'([^']*)'/.exec(body)?.[1],
      declaredKey: /key:\s*'([^']*)'/.exec(body)?.[1],
    });
  }
  return entries;
}

/**
 * Comprueba que el catálogo declare los tres tipos con su etiqueta oficial y que
 * `label`, `short` y `description` no introduzcan nombres alternativos.
 */
export function validateCatalog(source) {
  const errors = [];
  const entries = readCatalogEntries(source);
  const keys = entries.map((entry) => entry.key);
  const byKey = new Map(entries.map((entry) => [entry.key, entry]));

  for (const entry of entries) {
    if (!entry.label) errors.push(`catálogo: ${entry.key} no declara label.`);
    if (!entry.short) errors.push(`catálogo: ${entry.key} no declara short.`);
    if (!entry.description) errors.push(`catálogo: ${entry.key} no declara description.`);
    if (entry.declaredKey && entry.declaredKey !== entry.key) {
      errors.push(`catálogo: ${entry.key} declara key '${entry.declaredKey}'; debe coincidir con su clave.`);
    }
  }

  for (const [identifier, expected] of Object.entries(CANONICAL_TERMINOLOGY)) {
    const entry = byKey.get(identifier);
    if (!entry) {
      errors.push(`catálogo: falta el tipo ${identifier}.`);
      continue;
    }
    if (entry.label !== expected) {
      errors.push(`catálogo: ${identifier} debe mostrarse como «${expected}» y declara «${entry.label ?? ''}».`);
    }
  }

  for (const key of keys) {
    if (!(key in CANONICAL_TERMINOLOGY)) {
      errors.push(`catálogo: el tipo ${key} no está en la correspondencia oficial.`);
    }
  }

  if (!/export const EVALUATION_ENGINE_NAME = 'Juez Virtual'/.test(source)) {
    errors.push("catálogo: debe declarar EVALUATION_ENGINE_NAME = 'Juez Virtual' como nombre visible del motor.");
  }

  for (const entry of entries) {
    if (!entry.short || !entry.label) continue;
    const shortTerm = classifyTerm(entry.short);
    if (shortTerm.kind !== 'other') {
      errors.push(
        `catálogo: ${entry.key} usa ${describeTerm(shortTerm)} en short; el texto corto debe abreviar el nombre oficial.`
      );
    }
    const labelWords = normalizeTermLabel(entry.label).split(' ');
    const shortWords = normalizeTermLabel(entry.short).split(' ');
    if (!shortWords.every((word) => labelWords.includes(word))) {
      errors.push(`catálogo: ${entry.key} declara short «${entry.short}», que no abrevia «${entry.label}».`);
    }
    if (MODALITY_HEADS.has(normalizeTermLabel(entry.short).split(' ')[0])) {
      errors.push(`catálogo: ${entry.key} declara short «${entry.short}» con forma de modalidad.`);
    }
    if (normalizeTermLabel(entry.short) === ENGINE_NAME_KEY) {
      errors.push(`catálogo: ${entry.key} no puede usar «${entry.short}» como texto corto.`);
    }
  }

  for (const entry of entries) {
    if (!entry.description) continue;
    for (const term of findForbiddenTermsInText(entry.description)) {
      errors.push(`catálogo: la descripción de ${entry.key} menciona ${describeTerm(term)}.`);
    }
  }

  const labels = entries.map((entry) => entry.label).filter(Boolean);
  if (labels.length !== new Set(labels).size) {
    errors.push('catálogo: hay dos tipos con la misma etiqueta visible.');
  }

  return errors;
}

function walkExpression(node, visit, seen = new Set()) {
  if (!node || typeof node !== 'object' || seen.has(node)) return;
  seen.add(node);
  visit(node);
  for (const value of Object.values(node)) {
    if (Array.isArray(value)) value.forEach((item) => walkExpression(item, visit, seen));
    else if (value && typeof value === 'object') walkExpression(value, visit, seen);
  }
}

function collectTemplateLabelUsages(template) {
  const usages = [];
  const parsed = parseTemplate(template, 'terminology-gate');
  if (parsed.errors?.length) return usages;

  const push = (line, value, kind) => {
    const text = String(value);
    const term = classifyTerm(text);
    if (term.kind !== 'other') {
      usages.push({ line, label: normalize(text), term, kind });
      return;
    }
    for (const span of embeddedLabelSpans(text, PROSE_LABEL_CATEGORIES)) {
      usages.push({ line, label: normalize(span.value), term: span, kind: `${kind} en prosa` });
    }
  };

  const visit = (node) => {
    const text = typeof node.value === 'string' ? node.value : node.value?.source;
    if (typeof text === 'string') push(node.sourceSpan?.start?.line, text, 'texto');
    for (const attribute of node.attributes ?? []) {
      if (typeof attribute.value === 'string') {
        push(attribute.sourceSpan?.start?.line, attribute.value, `atributo ${attribute.name}`);
      }
    }
    for (const expression of [node.value?.ast, ...(node.inputs ?? []).map((input) => input.value?.ast)]) {
      if (!expression) continue;
      walkExpression(expression, (candidate) => {
        if (candidate.constructor?.name === 'LiteralPrimitive' && typeof candidate.value === 'string') {
          push(node.sourceSpan?.start?.line, candidate.value, 'expresión');
        }
      });
    }
    for (const child of [
      ...(node.children ?? []),
      ...(node.branches ?? []).flatMap((branch) => branch.children ?? []),
      ...(node.groups ?? []).flatMap((group) => (group.cases ?? []).flatMap((branch) => branch.children ?? [])),
      ...(node.cases ?? []).flatMap((branch) => branch.children ?? []),
    ]) {
      visit(child);
    }
  };

  visit({ children: parsed.nodes });
  return usages;
}

function collectSourceLabelUsages(source, file) {
  const usages = [];
  const sourceFile = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);

  const visit = (node) => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      const line = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line;
      const term = classifyTerm(node.text);
      if (term.kind !== 'other') {
        usages.push({ line, label: normalize(node.text), term, kind: 'literal' });
      } else {
        for (const span of embeddedLabelSpans(node.text, PROSE_LABEL_CATEGORIES)) {
          usages.push({ line, label: normalize(span.value), term: span, kind: 'literal en prosa' });
        }
      }
    }
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return usages;
}

function collectInlineTemplates(source, file) {
  const sourceFile = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const templates = [];

  const visit = (node) => {
    if (ts.isPropertyAssignment(node) && node.name.getText(sourceFile) === 'template') {
      const initializer = node.initializer;
      if (ts.isStringLiteral(initializer) || ts.isNoSubstitutionTemplateLiteral(initializer)) {
        templates.push(initializer.text);
      }
    }
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
  return templates;
}

function formatUsages(file, usages) {
  return usages.map((usage) => {
    const hint = usage.term.kind === 'engine' ? 'Nombra el motor en prosa, no como etiqueta.' : 'Usar environmentLabel.';
    return `${file}:${usage.line == null ? '' : usage.line + 1}: ${describeTerm(usage.term)} en ${usage.kind}. ${hint}`;
  });
}

/** Revisa un archivo del frontend (componente, utilidad o plantilla). */
export function validateSourceFile(file, source) {
  if (file.endsWith('.html')) {
    return formatUsages(file, collectTemplateLabelUsages(source));
  }

  const errors = formatUsages(file, collectSourceLabelUsages(source, file));
  for (const template of collectInlineTemplates(source, file)) {
    errors.push(...formatUsages(`${file} (template inline)`, collectTemplateLabelUsages(template)));
  }
  return errors;
}

/** Comprueba que las instrucciones de agentes declaren la correspondencia. */
export function validateAgentInstructions(agentsText, skillText) {
  const errors = [];
  const sources = [
    { file: 'AGENTS.md', text: agentsText },
    { file: '.agents/skills/solv-context/SKILL.md', text: skillText },
  ];

  for (const { file, text } of sources) {
    for (const [identifier, label] of Object.entries(CANONICAL_TERMINOLOGY)) {
      const line = text
        .split('\n')
        .find((candidate) => candidate.includes(`\`${identifier}\``) && candidate.includes(label));
      if (!line) {
        errors.push(`${file}: falta la correspondencia ${identifier} → «${label}».`);
      }
    }
  }

  for (const { file, text } of sources) {
    if (!/environmentLabel|environment-types\.constants/.test(text)) {
      errors.push(`${file}: debe indicar que la etiqueta sale de environmentLabel.`);
    }
    if (!text.includes(POLICY_ADR)) {
      errors.push(`${file}: debe referenciar la decisión vigente ${POLICY_ADR}.`);
    }
  }
  return errors;
}

/**
 * Comprueba que las excepciones históricas sigan siendo precisas. Una excepción
 * sin motivo, con comodines de ruta, fuera de los registros de gobernanza o que
 * no referencie la decisión vigente se rechaza.
 */
export function validateExceptionRegistry(entries, options = {}) {
  const policyAdr = options.policyAdr ?? POLICY_ADR;
  const maxEntries = options.maxEntries ?? MAX_GOVERNANCE_RECORDS;
  const errors = [];

  if (entries.length > maxEntries) {
    errors.push(`excepciones: hay ${entries.length} registros históricos; el máximo es ${maxEntries}.`);
  }

  for (const entry of entries) {
    const path = entry?.path ?? '';
    if (!path) {
      errors.push('excepciones: hay un registro sin ruta.');
      continue;
    }
    if (/[*?[\]]/.test(path)) {
      errors.push(`excepciones: la ruta «${path}» usa comodines; debe ser exacta.`);
    }
    if (!path.startsWith('docs/GOBERNANZA/')) {
      errors.push(`excepciones: «${path}» queda fuera de docs/GOBERNANZA/; solo se admite ese registro histórico.`);
    }
    if (!entry.reason || normalize(entry.reason).length < 20) {
      errors.push(`excepciones: «${path}» no documenta un motivo suficiente.`);
    }
    if (entry.policyAdr !== policyAdr) {
      errors.push(`excepciones: «${path}» debe referenciar la decisión vigente ${policyAdr}.`);
    }
    if (!entry.stillDetects) {
      errors.push(`excepciones: «${path}» no declara qué sigue detectando el gate.`);
    }
  }

  return errors;
}

function adrNumber(file) {
  const match = /ADR-(\d+)/.exec(file);
  return match ? Number(match[1]) : null;
}

/**
 * Un ADR es histórico si es anterior a la decisión vigente y la referencia de
 * forma explícita. Un ADR nuevo no puede autoeximirse.
 */
export function isHistoricalAdr(file, text) {
  const number = adrNumber(file);
  if (number == null || number >= POLICY_ADR_NUMBER) return false;
  if (!text.includes(POLICY_ADR)) return false;
  return HISTORICAL_MARKER.test(text);
}

function governanceRecord(file) {
  return GOVERNANCE_RECORDS.find((entry) => entry.path === file) ?? null;
}

function documentStatus(file, text) {
  if (isHistoricalAdr(file, text)) {
    return { historical: true, reason: `ADR anterior a ${POLICY_ADR}` };
  }
  const record = governanceRecord(file);
  if (record && HISTORICAL_MARKER.test(text) && text.includes(POLICY_ADR)) {
    return { historical: true, reason: record.reason };
  }
  return { historical: false, reason: '' };
}

/**
 * Valla de bloque de código: tres o más tildes o acentos graves al inicio con un
 * lenguaje opcional y nada más. Una línea de arte ASCII con tildes dentro de un
 * bloque `text` no abre ni cierra nada.
 */
function isFencedBlockStart(line) {
  return /^ {0,3}(?:`{3,}|~{3,})[A-Za-z0-9_+#.-]*\s*$/.test(line);
}

function stripFormatting(value) {
  return value.replace(/`/g, '').replace(/\*\*/g, '').replace(/^\s*[-*]\s+/, '').trim();
}

function isQuotedMention(value) {
  const trimmed = value.trim();
  return /^[«"“`'].*[»"”`']$/.test(trimmed);
}

function headingText(line) {
  const match = /^\s{0,3}(#{1,6})\s+(.*)$/.exec(line);
  if (!match) return null;
  return match[2]
    .replace(/^\d+(\.\d+)*\.?\s*/, '')
    .replace(/[*_`]/g, '')
    .trim();
}

function rowCells(line) {
  const trimmed = line.trim();
  if (!trimmed.startsWith('|')) return null;
  const cells = trimmed.split('|').slice(1, -1).map((cell) => cell.trim());
  if (cells.every((cell) => /^:?-{2,}:?$/.test(cell) || cell === '')) return null;
  return cells;
}

/** Categorías que se revisan dentro de la prosa de un documento vigente. */
const DOC_PROSE_CATEGORIES = new Set(['modalidad']);

/**
 * Tramos de una línea que son citas y no uso: código en línea, texto
 * entrecomillado y enlaces a ADR anteriores a la decisión vigente.
 */
function citationRanges(line) {
  const ranges = [];
  for (const match of line.matchAll(/`[^`]*`|«[^»]*»|"[^"]*"|“[^”]*”/g)) {
    ranges.push([match.index, match.index + match[0].length]);
  }
  for (const match of line.matchAll(/\[([^\]]+)\]\(([^)]+)\)/g)) {
    const adr = /ADR-0*(\d+)/.exec(match[2]) ?? /ADR-0*(\d+)/.exec(match[1]);
    if (adr && Number(adr[1]) < POLICY_ADR_NUMBER) {
      ranges.push([match.index, match.index + match[0].length]);
    }
  }
  return ranges;
}

/** Extrae los identificadores de contrato presentes en un texto. */
export function contractIdentifiersIn(text) {
  return CONTRACT_IDENTIFIERS.filter((identifier) => new RegExp(`\\b${identifier}\\b`).test(text));
}

/**
 * Revisa documentación vigente con reglas por contexto: celda de tabla,
 * encabezado, término de glosario y correspondencia identificador ↔ etiqueta.
 * La prosa descriptiva del motor se conserva.
 */
export function validateDocText(file, text, options = {}) {
  const errors = [];
  const normalizedFile = file.split(sep).join('/');
  const status = options.status ?? documentStatus(normalizedFile, text);
  const lines = text.split('\n');
  let inFence = false;

  lines.forEach((line, index) => {
    const lineNumber = index + 1;

    if (isFencedBlockStart(line)) {
      inFence = !inFence;
      return;
    }
    if (inFence) return;

    const cells = rowCells(line);
    const reportedTerms = new Set();

    if (cells) {
      for (const cell of cells) {
        if (!cell) continue;
        const term = classifyTerm(stripFormatting(cell).replace(/\s*\([^)]*\)\s*$/, ''));
        if (status.historical && (term.category !== 'modalidad' || isQuotedMention(cell))) continue;
        if (term.kind === 'non-official') {
          reportedTerms.add(term.label);
          errors.push(`${normalizedFile}:${lineNumber}: ${describeTerm(term)} en celda de tabla. Usar el nombre oficial.`);
        }
      }

      const identifiers = contractIdentifiersIn(line);
      if (identifiers.length === 1) {
        const identifier = identifiers[0];
        for (const cell of cells) {
          const term = classifyTerm(stripFormatting(cell));
          if (term.kind === 'official' && term.identifier !== identifier) {
            errors.push(
              `${normalizedFile}:${lineNumber}: \`${identifier}\` aparece junto a «${term.label}»; su nombre visible es «${CANONICAL_TERMINOLOGY[identifier]}».`
            );
          }
        }
      }
    }

    const heading = headingText(line);
    if (heading) {
      const term = classifyTerm(heading);
      if (status.historical && (term.category !== 'modalidad' || isQuotedMention(heading))) return;
      if (term.kind === 'non-official') {
        reportedTerms.add(term.label);
        errors.push(`${normalizedFile}:${lineNumber}: ${describeTerm(term)} en encabezado. Usar el nombre oficial.`);
      }
    }

    const boldLabel = /^\s*[-*]\s+\*\*([^*]+)\*\*/.exec(line);
    if (boldLabel) {
      const term = classifyTerm(normalizeModalityName(boldLabel[1]));
      const exempt = status.historical && (term.category !== 'modalidad' || isQuotedMention(boldLabel[1]));
      if (!exempt && term.kind === 'non-official') {
        reportedTerms.add(term.label);
        errors.push(`${normalizedFile}:${lineNumber}: ${describeTerm(term)} en glosario. Usar el nombre oficial.`);
      }
    }

    if (!status.historical) {
      const identifiers = contractIdentifiersIn(line);
      if (identifiers.length) {
        const reported = new Set();
        for (const rawValue of (cells ?? [line]).map((cell) => stripFormatting(cell))) {
          const quoted = [...rawValue.matchAll(/«([^»]+)»|"([^"]+)"/g)].map((match) => (match[1] ?? match[2]).trim());
          for (const candidate of quoted.length ? quoted : [rawValue.replace(/\s*\([^)]*\)\s*$/, '')]) {
            const term = classifyTerm(candidate);
            if (term.kind !== 'non-official' && term.kind !== 'engine') continue;
            const key = `${term.kind}:${term.label}`;
            if (reported.has(key)) continue;
            reported.add(key);
            reportedTerms.add(term.label);
            errors.push(
              `${normalizedFile}:${lineNumber}: un identificador de contrato aparece junto a ${describeTerm(term)}; el nombre visible de \`${identifiers[0]}\` es «${CANONICAL_TERMINOLOGY[identifiers[0]]}».`
            );
          }
        }
      }
    }
    if (status.historical) return;

    const citations = citationRanges(line);
    for (const span of embeddedLabelSpans(line, DOC_PROSE_CATEGORIES)) {
      if (reportedTerms.has(span.label)) continue;
      if (citations.some(([start, end]) => span.start >= start && span.end <= end)) continue;
      errors.push(
        `${normalizedFile}:${lineNumber}: ${describeTerm(span)} en prosa. Usar el nombre oficial o entrecomillarlo como cita.`
      );
    }
  });

  return errors;
}

/**
 * El pipe debe resolver desde el catálogo y no contener etiquetas propias: es el
 * único punto donde un nombre oficial pasa a la interfaz.
 */
export function validatePipeImplementation(source) {
  const errors = [];
  if (!/getEnvironmentMeta\(/.test(source)) {
    errors.push('environment-label.pipe.ts: el pipe debe resolver la etiqueta con getEnvironmentMeta del catálogo.');
  }
  if (!/if \(!meta\) return '';/.test(source)) {
    errors.push("environment-label.pipe.ts: un tipo desconocido debe devolver cadena vacía y no una modalidad.");
  }
  for (const label of Object.values(CANONICAL_TERMINOLOGY)) {
    if (source.includes(`'${label}'`) || source.includes(`"${label}"`)) {
      errors.push(`environment-label.pipe.ts: no debe declarar «${label}»; la etiqueta vive en el catálogo.`);
    }
  }
  return errors;
}

export function runTerminologyGate() {
  const errors = [];
  const required = [CATALOG_FILE, PIPE_FILE, AGENTS_FILE, SKILL_FILE, DOCS_DIR];
  const missing = required.filter((path) => !existsSync(path));
  if (missing.length) {
    console.error(
      '\n✖ terminology-gate: faltan rutas del repositorio.\n\n' +
        missing.map((path) => `  - ${relative(ROOT, path)}`).join('\n') +
        '\n\n  El gate se ejecuta desde frontend/ y espera la raíz del repositorio.\n'
    );
    process.exitCode = 1;
    return;
  }

  errors.push(...validateExceptionRegistry(GOVERNANCE_RECORDS));
  errors.push(...validateCatalog(readFileSync(CATALOG_FILE, 'utf8')));
  errors.push(...validatePipeImplementation(readFileSync(PIPE_FILE, 'utf8')));

  for (const path of walk(SRC_APP, ['.ts', '.html'])) {
    if (path.endsWith('.spec.ts') || resolve(path) === resolve(CATALOG_FILE)) continue;
    errors.push(...validateSourceFile(relative(ROOT, path), readFileSync(path, 'utf8')));
  }

  errors.push(...validateAgentInstructions(readFileSync(AGENTS_FILE, 'utf8'), readFileSync(SKILL_FILE, 'utf8')));

  for (const path of walk(DOCS_DIR, ['.md'])) {
    const relativePath = relative(REPO_ROOT, path).split(sep).join('/');
    errors.push(...validateDocText(relativePath, readFileSync(path, 'utf8')));
  }

  if (errors.length) {
    console.error(
      `\n✖ terminology-gate: ${errors.length} inconsistencia(s) de terminología.\n\n` +
        errors.map((error) => `  - ${error}`).join('\n') +
        '\n\n  Correspondencia oficial: JUEZ_EFIMERO → Entorno de Evaluación · IDE_PERSISTENTE → Entorno de Desarrollo · AMBOS → Entorno de Desarrollo y Evaluación\n' +
        '  Las etiquetas salen de environmentLabel; «Juez Virtual» solo nombra el motor en prosa.\n' +
        `  Política vigente: ${POLICY_ADR}.\n`
    );
    process.exitCode = 1;
    return;
  }

  console.log('terminology-gate: OK — catálogo, etiquetas, agentes y documentación alineados');
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  runTerminologyGate();
}
