import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { parseTemplate } from '@angular/compiler';
import ts from 'typescript';

const SRC_APP = join(process.cwd(), 'src', 'app');
const TECHNICAL_MEMBER_NAMES = new Set([
  'id',
  'code',
  'uuid',
  'hash',
  'version',
  'verdict',
  'level',
  'tag',
  'image_tag',
  'workspace_id',
  'docker_version',
  'uptime_seconds',
  'path',
  'base_ram_mb',
  'currentRAM',
  'currentCPU',
  'ttl_remaining_seconds',
  'resource_type',
  'docker_image',
  'memory_used_mb',
  'memory_limit_mb',
  'cpu_cores',
  'cpu_percent',
  'disk_percent',
  'containers_active',
  'containers_max',
  'containers_hibernated',
  'ram_percent',
  'ram_used_gb',
  'timestamp',
  'duration_ms',
  'ramPercentComputed',
  'ramUsedGB',
  'ramTotalGB',
  'diskUsedGB',
  'diskTotalGB',
  'concurrencyPercent',
  'getMemoryPercent',
  'formatUptime',
  'formatTTL'
]);
const MACHINE_MARKED_ELEMENTS = new Set(['code', 'pre', 'kbd', 'samp']);
const FORM_CONTROL_ELEMENTS = new Set(['input', 'select', 'option', 'textarea']);

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const info = statSync(path);
    if (info.isDirectory()) walk(path, out);
    else if ((path.endsWith('.html') || path.endsWith('.ts')) && !path.endsWith('.spec.ts')) out.push(path);
  }
  return out;
}

function isTechnicalMemberName(name) {
  return TECHNICAL_MEMBER_NAMES.has(name) || name === 'id' || /_id$/i.test(name);
}

function isTechnicalOutputExpression(expression) {
  if (!expression || typeof expression !== 'object') return false;

  const expressionType = expression.constructor?.name;
  if (expressionType === 'PropertyRead' || expressionType === 'SafePropertyRead') {
    return isTechnicalMemberName(expression.name);
  }
  if (expressionType === 'Call') {
    return Boolean(expression.receiver?.name && TECHNICAL_MEMBER_NAMES.has(expression.receiver.name));
  }
  if (expressionType === 'Conditional') {
    return isTechnicalOutputExpression(expression.trueExp) || isTechnicalOutputExpression(expression.falseExp);
  }
  if (expressionType === 'Binary') {
    return expression.operation === '+' &&
      (isTechnicalOutputExpression(expression.left) || isTechnicalOutputExpression(expression.right));
  }
  if (expressionType === 'BindingPipe') {
    return isTechnicalOutputExpression(expression.exp);
  }
  if (expressionType === 'KeyedRead') {
    return isTechnicalOutputExpression(expression.receiver);
  }
  return false;
}

function isTechnicalInterpolation(boundText) {
  const expressions = boundText.value?.ast?.expressions;
  if (!Array.isArray(expressions)) return false;
  return expressions.some(isTechnicalOutputExpression);
}

function walkTemplateNodes(nodes, file, errors, machineDataAncestor = false, formControlAncestor = false) {
  for (const node of nodes ?? []) {
    const staticClasses = node.attributes?.find((attribute) => attribute.name === 'class')?.value?.split(/\s+/) ?? [];
    const marked = machineDataAncestor || MACHINE_MARKED_ELEMENTS.has(node.name) || staticClasses.includes('font-mono') || Boolean(
      node.attributes?.some((attribute) => attribute.name === 'machineData')
    );
    const inFormControl = formControlAncestor || FORM_CONTROL_ELEMENTS.has(node.name);

    if (!inFormControl && node.value?.ast?.expressions && isTechnicalInterpolation(node) && !marked) {
      const line = node.sourceSpan?.start?.line;
      errors.push(`${file}:${line == null ? '' : line + 1}: dato técnico interpolado sin machineData.`);
    }

    if (node.children) walkTemplateNodes(node.children, file, errors, marked, inFormControl);
    for (const branch of node.branches ?? []) {
      walkTemplateNodes(branch.children, file, errors, marked, inFormControl);
    }
    for (const group of node.groups ?? []) {
      for (const branch of group.cases ?? []) walkTemplateNodes(branch.children, file, errors, marked, inFormControl);
    }
    for (const block of node.cases ?? []) walkTemplateNodes(block.children, file, errors, marked, inFormControl);
  }
}

export function validateMachineDataTemplate(file, template) {
  const errors = [];
  const parsed = parseTemplate(template, file);
  if (parsed.errors?.length) {
    return parsed.errors.map((error) => `${file}: plantilla Angular inválida: ${error.msg}`);
  }
  walkTemplateNodes(parsed.nodes, file, errors);
  return errors;
}

function collectInlineTemplates(source, file) {
  const sourceFile = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const templates = [];

  function visit(node) {
    if (ts.isPropertyAssignment(node) && node.name.getText(sourceFile) === 'template') {
      const initializer = node.initializer;
      if (ts.isStringLiteral(initializer) || ts.isNoSubstitutionTemplateLiteral(initializer)) {
        templates.push(initializer.text);
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return templates;
}

export function runMachineDataGate() {
  const errors = [];
  for (const path of walk(SRC_APP)) {
    const source = readFileSync(path, 'utf8');
    if (path.endsWith('.html')) {
      errors.push(...validateMachineDataTemplate(relative(process.cwd(), path), source));
      continue;
    }

    for (const template of collectInlineTemplates(source, path)) {
      errors.push(...validateMachineDataTemplate(relative(process.cwd(), path), template));
    }
  }

  if (errors.length) {
    console.error(
      `\n✖ machine-data-gate: ${errors.length} dato(s) técnico(s) sin la directiva [machineData].\n\n` +
        errors.map((error) => `  - ${error}`).join('\n') +
        '\n'
    );
    process.exitCode = 1;
    return;
  }

  console.log('machine-data-gate: OK — datos técnicos marcados en templates Angular');
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  runMachineDataGate();
}
