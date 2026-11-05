import { describe, expect, it } from 'vitest';
import {
  CANONICAL_TERMINOLOGY,
  GOVERNANCE_RECORDS,
  POLICY_ADR,
  classifyTerm,
  embeddedLabelSpans,
  validateAgentInstructions,
  validateCatalog,
  validateDocText,
  validateExceptionRegistry,
  validatePipeImplementation,
  validateSourceFile,
} from './terminology-gate.mjs';

const CATALOG = `
export type EnvironmentTypeKey = 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS';

export const EVALUATION_ENGINE_NAME = 'Juez Virtual';

export const ENVIRONMENT_TYPES: Record<EnvironmentTypeKey, EnvironmentTypeMeta> = {
  IDE_PERSISTENTE: {
    key: 'IDE_PERSISTENTE',
    label: 'Entorno de Desarrollo',
    short: 'Desarrollo',
    description: 'Espacio de trabajo interactivo con VS Code.',
    badgeClass: 'env-desarrollo',
  },
  JUEZ_EFIMERO: {
    key: 'JUEZ_EFIMERO',
    label: 'Entorno de Evaluación',
    short: 'Evaluación',
    description: 'Corrección automática contra casos de prueba.',
    badgeClass: 'env-evaluacion',
  },
  AMBOS: {
    key: 'AMBOS',
    label: 'Entorno de Desarrollo y Evaluación',
    short: 'Desarrollo y Evaluación',
    description: 'Sirve para las dos modalidades.',
    badgeClass: 'env-dual',
  },
};
`;

describe('terminology-gate classification', () => {
  it('classifies official labels, the engine and modality-shaped names apart', () => {
    expect(classifyTerm('Entorno de Evaluación')).toMatchObject({ kind: 'official', identifier: 'JUEZ_EFIMERO' });
    expect(classifyTerm('Entorno de Desarrollo y Evaluación')).toMatchObject({ kind: 'official', identifier: 'AMBOS' });
    expect(classifyTerm('Juez Virtual')).toMatchObject({ kind: 'engine' });
    expect(classifyTerm('Entorno de Programación')).toMatchObject({ kind: 'non-official', category: 'modalidad' });
    expect(classifyTerm('Evaluador Automático')).toMatchObject({ kind: 'non-official', category: 'motor' });
    expect(classifyTerm('IDE Persistente')).toMatchObject({ kind: 'non-official', category: 'herramienta' });
  });

  it('does not treat other concepts or prose as modality labels', () => {
    expect(classifyTerm('Entorno de trabajo').kind).toBe('other');
    expect(classifyTerm('Entorno de la práctica').kind).toBe('other');
    expect(classifyTerm('Espacio de trabajo').kind).toBe('other');
    expect(classifyTerm('Entorno de Laboratorio OpenVSCode').kind).toBe('other');
    expect(classifyTerm('Entorno interactivo web').kind).toBe('other');
    expect(classifyTerm('Modalidad de entorno').kind).toBe('other');
    expect(classifyTerm('El estudiante trabaja en el entorno de desarrollo').kind).toBe('other');
  });

  it('fails closed for an unknown noun after «Entorno de»', () => {
    expect(classifyTerm('Entorno de Codificación')).toMatchObject({ kind: 'non-official', category: 'modalidad' });
    expect(classifyTerm('Juez de Corrección')).toMatchObject({ kind: 'non-official', category: 'motor' });
  });

  it('normalizes case and accents before classifying', () => {
    expect(classifyTerm('Entorno de evaluación')).toMatchObject({ kind: 'official' });
    expect(classifyTerm('ENTORNO DE EVALUACION')).toMatchObject({ kind: 'official' });
    expect(classifyTerm('entorno de programacion')).toMatchObject({ kind: 'non-official' });
    expect(classifyTerm('IDE PERSISTENTE')).toMatchObject({ kind: 'non-official' });
  });

  it('strips numbering, quotes and trailing punctuation from a label', () => {
    expect(classifyTerm('2.3.1 «Entorno de Evaluación»:')).toMatchObject({ kind: 'official' });
    expect(classifyTerm('`Juez Virtual`')).toMatchObject({ kind: 'engine' });
  });
});

describe('terminology-gate catalog', () => {
  it('accepts the official catalog', () => {
    expect(validateCatalog(CATALOG)).toEqual([]);
  });

  it('rejects a label that does not match the official correspondence', () => {
    const broken = CATALOG.replace("label: 'Entorno de Evaluación'", "label: 'Juez Virtual'");
    expect(validateCatalog(broken).join('\n')).toContain('JUEZ_EFIMERO debe mostrarse como «Entorno de Evaluación»');
  });

  it('rejects a catalog without the dual modality', () => {
    const broken = CATALOG.replace(/  AMBOS: \{[\s\S]*?\n  \},\n/, '');
    expect(validateCatalog(broken).join('\n')).toContain('falta el tipo AMBOS');
  });

  it('rejects an identifier whose declared key does not match', () => {
    const broken = CATALOG.replace("    key: 'JUEZ_EFIMERO',", "    key: 'JUEZ_VIRTUAL',");
    expect(validateCatalog(broken).join('\n')).toContain('debe coincidir con su clave');
  });

  it('rejects a short name that introduces another modality name', () => {
    const broken = CATALOG.replace("    short: 'Evaluación',", "    short: 'Juez Virtual',");
    expect(validateCatalog(broken).join('\n')).toContain('en short');
  });

  it('rejects a short name that is not an abbreviation of the label', () => {
    const broken = CATALOG.replace("    short: 'Desarrollo',", "    short: 'Código',");
    expect(validateCatalog(broken).join('\n')).toContain('no abrevia');
  });

  it('rejects a description that names a non-official modality', () => {
    const broken = CATALOG.replace(
      "description: 'Corrección automática contra casos de prueba.'",
      "description: 'Se ejecuta en el Entorno de Programación.'"
    );
    expect(validateCatalog(broken).join('\n')).toContain('Entorno de Programación');
  });

  it('rejects a description that names a deprecated engine', () => {
    const broken = CATALOG.replace(
      "description: 'Corrección automática contra casos de prueba.'",
      "description: 'Lo ejecuta el Evaluador Automático.'"
    );
    expect(validateCatalog(broken).join('\n')).toContain('motor no oficial');
  });

  it('rejects a catalog without the engine name', () => {
    const broken = CATALOG.replace("export const EVALUATION_ENGINE_NAME = 'Juez Virtual';", '');
    expect(validateCatalog(broken).join('\n')).toContain('EVALUATION_ENGINE_NAME');
  });

  it('rejects two types with the same visible label', () => {
    const broken = CATALOG.replace("label: 'Entorno de Desarrollo y Evaluación'", "label: 'Entorno de Desarrollo'");
    expect(validateCatalog(broken).join('\n')).toContain('misma etiqueta');
  });
});

describe('terminology-gate pipe', () => {
  it('accepts a pipe that resolves through the catalog', () => {
    const source = `import { getEnvironmentMeta } from '../constants/environment-types.constants';
const meta = getEnvironmentMeta(value);
if (!meta) return '';`;
    expect(validatePipeImplementation(source)).toEqual([]);
  });

  it('rejects a pipe that declares its own labels', () => {
    const source = `const label = 'Entorno de Evaluación';\nimport { getEnvironmentMeta } from './x';\nif (!meta) return '';`;
    expect(validatePipeImplementation(source).join('\n')).toContain('no debe declarar');
  });

  it('rejects a pipe that does not use the catalog', () => {
    expect(validatePipeImplementation('export class P {}').join('\n')).toContain('getEnvironmentMeta');
  });

  it('rejects a pipe that falls back to a modality for unknown values', () => {
    const source = `import { getEnvironmentMeta } from './x';\nconst meta = getEnvironmentMeta(value);\nreturn meta.label;`;
    expect(validatePipeImplementation(source).join('\n')).toContain('tipo desconocido');
  });
});

describe('terminology-gate source scanning', () => {
  it('rejects a modality label written as a literal', () => {
    expect(validateSourceFile('src/example.component.ts', `export const label = 'Entorno de Evaluación';`)).toEqual([
      expect.stringContaining('nombre oficial de modalidad «Entorno de Evaluación»'),
    ]);
  });

  it('rejects a new synonym used as a label', () => {
    expect(validateSourceFile('src/example.component.ts', `export const label = 'Entorno de Programación';`)).toEqual([
      expect.stringContaining('nombre de modalidad no oficial «Entorno de Programación»'),
    ]);
  });

  it('rejects a new synonym written in a template text node', () => {
    expect(validateSourceFile('src/example.component.html', '<span class="badge">Entorno de Programación</span>')).toEqual([
      expect.stringContaining('Entorno de Programación'),
    ]);
  });

  it('rejects case and accent variants of an official label', () => {
    for (const variant of ['Entorno de evaluación', 'ENTORNO DE EVALUACION', 'entorno de evaluacion']) {
      expect(validateSourceFile('src/example.component.ts', `const x = '${variant}';`)).toEqual([
        expect.stringContaining('environmentLabel'),
      ]);
    }
  });

  it('rejects a modality label written in a template text node', () => {
    expect(validateSourceFile('src/example.component.html', '<span>Entorno de Desarrollo</span>')).toEqual([
      expect.stringContaining('Entorno de Desarrollo'),
    ]);
  });

  it('rejects a modality label written inside an interpolation', () => {
    const template = `<span>{{ isIde() ? 'Entorno de Desarrollo' : 'Entorno de Evaluación' }}</span>`;
    expect(validateSourceFile('src/example.component.html', template)).toHaveLength(2);
  });

  it('rejects the evaluation engine used as a modality label', () => {
    expect(validateSourceFile('src/example.component.html', '<span class="badge">Evaluador Automático</span>')).toEqual([
      expect.stringContaining('Evaluador Automático'),
    ]);
  });

  it('rejects the engine name used as a label', () => {
    expect(validateSourceFile('src/example.component.html', '<span class="badge">Juez Virtual</span>')).toEqual([
      expect.stringContaining('nombre del motor'),
    ]);
  });

  it('accepts the catalog through the pipe', () => {
    const template = `<span>{{ exercise.environment_type | environmentLabel }}</span>`;
    expect(validateSourceFile('src/example.component.html', template)).toEqual([]);
  });

  it('accepts the identifiers as contract values', () => {
    const template = `<span>{{ (isIde() ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO') | environmentLabel }}</span>`;
    expect(validateSourceFile('src/example.component.html', template)).toEqual([]);
  });

  it('accepts prose that describes the evaluation engine', () => {
    const template = '<p>El Juez Virtual ejecutará tu programa contra los casos de prueba.</p>';
    expect(validateSourceFile('src/example.component.html', template)).toEqual([]);
  });

  it('accepts prose inside a component string', () => {
    const source = `export const message = 'No se pudo evaluar la solución en el Entorno de Evaluación.';`;
    expect(validateSourceFile('src/example.component.ts', source)).toEqual([]);
  });

  it('accepts other concepts that share the word entorno', () => {
    for (const phrase of ['Entorno de trabajo', 'Espacio de trabajo', 'Entorno de la práctica', 'Entorno interactivo web']) {
      expect(validateSourceFile('src/example.component.ts', `const x = '${phrase}';`)).toEqual([]);
    }
  });

  it('rejects a modality label inside an inline template', () => {
    const source = `
@Component({
  selector: 'example',
  template: \`<span>Entorno de Evaluación</span>\`,
})
export class ExampleComponent {}
`;
    expect(validateSourceFile('src/example.component.ts', source)).toEqual([expect.stringContaining('template inline')]);
  });
});

describe('terminology-gate agent instructions', () => {
  const agents = Object.entries(CANONICAL_TERMINOLOGY)
    .map(([identifier, label]) => `| \`${identifier}\` | ${label} |`)
    .join('\n');

  it('accepts instructions that declare the correspondence, the catalog and the decision', () => {
    const text = `${agents}\nUsar environmentLabel. Política ${POLICY_ADR}.`;
    expect(validateAgentInstructions(text, text)).toEqual([]);
  });

  it('rejects instructions that do not declare a type', () => {
    const text = `${agents.replace(/^\| `AMBOS`.*$/m, '')}\nUsar environmentLabel. ${POLICY_ADR}.`;
    expect(validateAgentInstructions(text, text).join('\n')).toContain('falta la correspondencia AMBOS');
  });

  it('rejects instructions that assign a non-official name', () => {
    const text = `${agents.replace('| `JUEZ_EFIMERO` | Entorno de Evaluación |', '| `JUEZ_EFIMERO` | Juez Virtual |')}\nUsar environmentLabel. ${POLICY_ADR}.`;
    expect(validateAgentInstructions(text, text).join('\n')).toContain('JUEZ_EFIMERO → «Entorno de Evaluación»');
  });

  it('rejects instructions that do not point to the catalog', () => {
    expect(validateAgentInstructions(`${agents} ${POLICY_ADR}`, `${agents} ${POLICY_ADR}`).join('\n')).toContain('environmentLabel');
  });

  it('rejects instructions that do not reference the current decision', () => {
    expect(validateAgentInstructions(`${agents} environmentLabel`, `${agents} environmentLabel`).join('\n')).toContain(POLICY_ADR);
  });
});

describe('terminology-gate documentation', () => {
  const adr = 'docs/ARQUITECTURA/ADR/ADR-052.md';
  const marker = `> **Terminología vigente:** ${POLICY_ADR}. Los nombres de este ADR no se reescriben.`;

  it('rejects a non-official name used as a table label', () => {
    const doc = '| Tipo | Nombre |\n| :--- | :--- |\n| `JUEZ_EFIMERO` | Juez Virtual |\n';
    expect(validateDocText('docs/UI/example.md', doc).join('\n')).toContain('Juez Virtual');
  });

  it('rejects a new synonym in a table cell', () => {
    const doc = '| Tipo | Nombre |\n| :--- | :--- |\n| IDE | Entorno de Programación |\n';
    expect(validateDocText('docs/UI/example.md', doc).join('\n')).toContain('Entorno de Programación');
  });

  it('rejects a new synonym in a heading', () => {
    expect(validateDocText('docs/UI/example.md', '## Entorno de Programación\n').join('\n')).toContain('encabezado');
  });

  it('rejects a deprecated engine name in a heading', () => {
    expect(validateDocText('docs/UI/example.md', '## Evaluador Automático\n').join('\n')).toContain('encabezado');
  });

  it('accepts the engine name in a heading', () => {
    expect(validateDocText('docs/UI/example.md', '## Juez Virtual\n')).toEqual([]);
  });

  it('rejects a non-official name used as a glossary entry', () => {
    expect(validateDocText('docs/UI/example.md', '- **Juez Evaluador (Sandbox):** ejecución aislada.\n')).toEqual([
      expect.stringContaining('Juez Evaluador'),
    ]);
  });

  it('rejects a contract identifier paired with a non-official modality name', () => {
    const doc = 'El filtro usa `JUEZ_EFIMERO` con etiqueta "Evaluador Automático".\n';
    expect(validateDocText('docs/UI/example.md', doc).join('\n')).toContain('nombre visible');
  });

  it('rejects an identifier labelled with another type', () => {
    const doc = '| Tipo | Nombre |\n| :--- | :--- |\n| `JUEZ_EFIMERO` | Entorno de Desarrollo |\n';
    expect(validateDocText('docs/UI/example.md', doc).join('\n')).toContain('Entorno de Desarrollo');
  });

  it('accepts the official names in tables and bullets', () => {
    const doc = [
      '| Tipo | Nombre |',
      '| :--- | :--- |',
      '| `JUEZ_EFIMERO` | Entorno de Evaluación |',
      '- **Entorno de Evaluación (`JUEZ_EFIMERO`):** corrección automática.',
    ].join('\n');
    expect(validateDocText('docs/UI/example.md', doc)).toEqual([]);
  });

  it('accepts the engine in an actor table', () => {
    const doc = '| Componente | Responsabilidad |\n| :--- | :--- |\n| Juez Virtual | ejecuta los casos privados |\n';
    expect(validateDocText('docs/UI/example.md', doc)).toEqual([]);
  });

  it('accepts prose about the evaluation engine', () => {
    const doc = 'El Juez Virtual ejecutará el programa contra los casos de prueba privados.\n';
    expect(validateDocText('docs/USER_GUIDE_STUDENT.md', doc)).toEqual([]);
  });

  it('ignores fenced code blocks', () => {
    const doc = '```mermaid\nparticipant UI as Juez Virtual\n```\n';
    expect(validateDocText('docs/UI/example.md', doc)).toEqual([]);
  });

  it('does not treat ascii art with tildes as a code fence', () => {
    const doc = ['```text', '   ~~~~~~~~~  [aviso]', '```', '', '## Entorno de Programación'].join('\n');
    expect(validateDocText('docs/UI/example.md', doc).join('\n')).toContain('Entorno de Programación');
  });

  it('keeps detecting errors in documents that used to be fully exempt', () => {
    const doc = '| Tipo | Nombre |\n| :--- | :--- |\n| `JUEZ_EFIMERO` | Juez Virtual |\n';
    expect(validateDocText('docs/UI/ESTUDIANTE/JUEZ_VIRTUAL.md', doc).join('\n')).toContain('Juez Virtual');
    expect(validateDocText('docs/UI/DOCENTE/JUEZ_VIRTUAL.md', '## Entorno de Programación\n').join('\n')).toContain(
      'Entorno de Programación'
    );
  });

  it('keeps the engine prose of the Juez Virtual documentation', () => {
    const doc = 'El Juez Virtual de SOLV es el entorno de resolución algorítmica interactiva.\n';
    expect(validateDocText('docs/UI/ESTUDIANTE/JUEZ_VIRTUAL.md', doc)).toEqual([]);
  });

  it('keeps a historical ADR untouched when it references the current decision', () => {
    const doc = ['# ADR-052 histórico', '', marker, '', '| `JUEZ_EFIMERO` | Juez Virtual |'].join('\n');
    expect(validateDocText(adr, doc)).toEqual([]);
  });

  it('rejects a historical claim without the reference to the current decision', () => {
    const doc = ['# ADR-052', '', '> **Terminología vigente:** los nombres actuales.', '', '| `JUEZ_EFIMERO` | Juez Virtual |'].join('\n');
    expect(validateDocText(adr, doc).join('\n')).toContain('Juez Virtual');
  });

  it('rejects a new ADR that tries to exempt itself', () => {
    const doc = ['# ADR-070', '', marker, '', '| `JUEZ_EFIMERO` | Juez Virtual |'].join('\n');
    expect(validateDocText('docs/ARQUITECTURA/ADR/ADR-070.md', doc).join('\n')).toContain('Juez Virtual');
  });

  it('rejects a historical ADR that references a different decision', () => {
    const doc = ['# ADR-052', '', '> **Terminología vigente:** ADR-999.', '', '| `JUEZ_EFIMERO` | Juez Virtual |'].join('\n');
    expect(validateDocText(adr, doc).join('\n')).toContain('Juez Virtual');
  });

  it('accepts quoted history in a governance record and still flags a new synonym', () => {
    const file = 'docs/GOBERNANZA/AUDITORIA_TERMINOLOGIA.md';
    const header = `> **Terminología vigente:** ${POLICY_ADR}. Registro de nombres anteriores.`;
    expect(validateDocText(file, `${header}\n| UI | «Evaluador Automático» |\n`)).toEqual([]);
    expect(validateDocText(file, `${header}\n| UI | Entorno de Programación |\n`).join('\n')).toContain(
      'Entorno de Programación'
    );
  });

  it('does not exempt a governance record without the decision reference', () => {
    const file = 'docs/GOBERNANZA/AUDITORIA_TERMINOLOGIA.md';
    expect(validateDocText(file, '| UI | «Evaluador Automático» |\n').join('\n')).toContain('Evaluador Automático');
  });
});

describe('terminology-gate prose spans', () => {
  it('finds an obsolete modality name inside a sentence', () => {
    const spans = embeddedLabelSpans('La solución se corrige en el Entorno de Programación y se entrega.');
    expect(spans.map((span) => span.value)).toContain('Entorno de Programación');
  });

  it('does not report generic prose in lowercase', () => {
    expect(embeddedLabelSpans('Arranca el entorno de desarrollo local con docker compose.')).toEqual([]);
  });

  it('does not report an official name used in prose', () => {
    expect(embeddedLabelSpans('La plantilla entra en el Entorno de Desarrollo sin cambios.')).toEqual([]);
  });

  it('does not report the engine named in prose', () => {
    expect(embeddedLabelSpans('El Juez Virtual evalúa contra los casos privados.')).toEqual([]);
  });

  it('does not join words across formatting', () => {
    expect(
      embeddedLabelSpans('**Nombre del Entorno \\*:** `Evaluador: Programación Competitiva C++`')
    ).toEqual([]);
  });

  it('treats «del» as a descriptive reference', () => {
    expect(embeddedLabelSpans('Variables de Entorno del Despachador de Correo')).toEqual([]);
  });

  it('reports one span per start instead of its extensions', () => {
    const spans = embeddedLabelSpans('Modalidad Dual y Gobernanza de Recursos');
    expect(spans.map((span) => span.value)).toEqual(['Modalidad Dual']);
  });

  it('flags a modality name embedded in a UI literal', () => {
    const source = "const hint = 'Se abre en el Entorno de Programación.';\n";
    expect(validateSourceFile('hint.ts', source).join('\n')).toContain('Entorno de Programación');
  });

  it('flags a tool name embedded in a UI literal', () => {
    const source = "const hint = 'Plataformas disponibles en el IDE persistente';\n";
    expect(validateSourceFile('hint.ts', source).join('\n')).toContain('IDE persistente');
  });

  it('flags an obsolete engine name embedded in a UI literal', () => {
    const source = "const hint = 'Casos de prueba sellados por el Evaluador Automático.';\n";
    const errors = validateSourceFile('hint.ts', source).join('\n');
    expect(errors).toContain('Evaluador Automático');
    expect(errors).toContain('EVALUATION_ENGINE_NAME');
  });

  it('does not report the engine named correctly inside a UI literal', () => {
    const source = "const hint = 'Los casos los sella el Juez Virtual antes de publicar.';\n";
    expect(validateSourceFile('hint.ts', source)).toEqual([]);
  });

  it('flags an obsolete engine name embedded in template prose', () => {
    const template = '<p>Los casos los sella el Evaluador Automático.</p>';
    expect(validateSourceFile('x.component.html', template).join('\n')).toContain('Evaluador Automático');
  });

  it('does not report lowercase engine prose', () => {
    expect(embeddedLabelSpans('El juez virtual sella los casos.')).toEqual([]);
  });

  it('flags a modality name embedded in template prose', () => {
    const template = '<p>La práctica se corrige en el Entorno de Programación.</p>';
    expect(validateSourceFile('x.component.html', template).join('\n')).toContain('Entorno de Programación');
  });

  it('flags a modality name embedded in documentation prose', () => {
    const doc = 'La entrega se revisa en el Entorno de Programación del curso.\n';
    expect(validateDocText('docs/UI/DOCENTE/example.md', doc).join('\n')).toContain('Entorno de Programación');
  });

  it('accepts a quoted obsolete name in documentation prose', () => {
    const doc = 'El nombre anterior era «Entorno de Programación» y ya no se usa.\n';
    expect(validateDocText('docs/UI/DOCENTE/example.md', doc)).toEqual([]);
  });

  it('accepts an obsolete name inside a link to an earlier ADR', () => {
    const doc = '- [Vocabulario de Entorno de Práctica](ADR-065-vocabulario-entorno-de-practica.md)\n';
    expect(validateDocText('docs/ARQUITECTURA/ADR/INDEX.md', doc)).toEqual([]);
  });

  it('does not classify prose in a historical ADR', () => {
    const doc = [
      '# ADR-052 histórico',
      '',
      '> **Terminología vigente:** ADR-067. Los nombres son los de su fecha.',
      '',
      'El flujo del Entorno de Programación se conserva.',
    ].join('\n');
    expect(validateDocText('docs/ARQUITECTURA/ADR/ADR-052-x.md', doc)).toEqual([]);
  });

  it('does not report the workspace component named in architecture prose', () => {
    const doc = 'El componente IDE Persistente documenta base_files[] y los bloques de comentarios.\n';
    expect(validateDocText('docs/ARQUITECTURA/EXAMPLE.md', doc)).toEqual([]);
  });
});

describe('terminology-gate exception registry', () => {
  it('accepts the declared governance records', () => {
    expect(validateExceptionRegistry(GOVERNANCE_RECORDS)).toEqual([]);
  });

  it('rejects a wildcard path', () => {
    const entries = [{ ...GOVERNANCE_RECORDS[0], path: 'docs/GOBERNANZA/*.md' }];
    expect(validateExceptionRegistry(entries).join('\n')).toContain('comodines');
  });

  it('rejects an exception outside the governance directory', () => {
    const entries = [{ ...GOVERNANCE_RECORDS[0], path: 'docs/UI/JUEZ_VIRTUAL.md' }];
    expect(validateExceptionRegistry(entries).join('\n')).toContain('docs/GOBERNANZA');
  });

  it('rejects an exception without a documented reason', () => {
    const entries = [{ ...GOVERNANCE_RECORDS[0], reason: 'histórico' }];
    expect(validateExceptionRegistry(entries).join('\n')).toContain('motivo');
  });

  it('rejects an exception that does not reference the current decision', () => {
    const entries = [{ ...GOVERNANCE_RECORDS[0], policyAdr: 'ADR-999' }];
    expect(validateExceptionRegistry(entries).join('\n')).toContain(POLICY_ADR);
  });

  it('rejects an exception that does not declare what is still detected', () => {
    const entries = [{ ...GOVERNANCE_RECORDS[0], stillDetects: '' }];
    expect(validateExceptionRegistry(entries).join('\n')).toContain('sigue detectando');
  });

  it('rejects a registry that grows too wide', () => {
    const entries = Array.from({ length: 5 }, (_, index) => ({
      ...GOVERNANCE_RECORDS[0],
      path: `docs/GOBERNANZA/HISTORICO_${index}.md`,
    }));
    expect(validateExceptionRegistry(entries).join('\n')).toContain('máximo');
  });
});
