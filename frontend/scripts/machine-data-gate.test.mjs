import { describe, expect, it } from 'vitest';
import { validateMachineDataTemplate } from './machine-data-gate.mjs';

describe('machine-data-gate template validation', () => {
  it('requires the marker for known technical bindings', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<span>{{ row.log.resource_type }}</span>')).toEqual([
      expect.stringContaining('machineData'),
    ]);
  });

  it('requires the marker for technical values returned by known methods', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<span>{{ ramPercentComputed() }}%</span>')).toEqual([
      expect.stringContaining('machineData'),
    ]);
  });

  it('accepts known technical bindings marked with machineData', () => {
    expect(
      validateMachineDataTemplate('src/example.component.html', '<span machineData>{{ row.log.resource_type }}</span>')
    ).toEqual([]);
  });

  it('accepts technical bindings inside an existing font-mono ancestor', () => {
    expect(
      validateMachineDataTemplate('src/example.component.html', '<div class="font-mono"><span>{{ row.log.verdict }}</span></div>')
    ).toEqual([]);
  });

  it.each(['code', 'pre', 'kbd', 'samp'])('accepts technical bindings inside <%s>', (element) => {
    expect(
      validateMachineDataTemplate('src/example.component.html', `<${element}>{{ row.log.level }}</${element}>`)
    ).toEqual([]);
  });

  it('detects technical identifiers, tags, verdicts, and log levels when unmarked', () => {
    expect(
      validateMachineDataTemplate(
        'src/example.component.html',
        '<span>{{ row.workspace_id }}</span><span>{{ row.image_tag }}</span><span>{{ row.verdict }}</span><span>{{ row.level }}</span>'
      )
    ).toHaveLength(4);
  });

  it('does not treat an identifier used only to look up a displayed count as machine output', () => {
    expect(
      validateMachineDataTemplate('src/example.component.html', '<span>{{ courseCountByPeriod()[row.id] ?? 0 }} cursos</span>')
    ).toEqual([]);
  });

  it('does not require markers on native form controls or options', () => {
    expect(
      validateMachineDataTemplate('src/example.component.html', '<select><option>{{ row.id }}</option></select><input [value]="row.id" />')
    ).toEqual([]);
  });

  it('does not classify explanatory UI copy by words alone', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<p>La memoria RAM del host está disponible.</p>')).toEqual([]);
  });

  it('does not classify translated status labels as machine data', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<span>{{ row.status.label }}</span>')).toEqual([]);
  });

  it('does not classify technical values used only in a UI text condition', () => {
    expect(
      validateMachineDataTemplate(
        'src/example.component.html',
        '<span>{{ actionInProgressId() === course.workspace_id ? \'Procesando...\' : \'Reiniciar\' }}</span>'
      )
    ).toEqual([]);
  });

  it('does not classify technical arguments that produce a humanized label', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<span>{{ getCategoryName(model.category_id) }}</span>')).toEqual([]);
  });

  it('requires the marker for technical values transformed by a pipe', () => {
    expect(validateMachineDataTemplate('src/example.component.html', '<span>{{ model.source_template_id | slice:0:8 }}</span>')).toEqual([
      expect.stringContaining('machineData'),
    ]);
  });
});
