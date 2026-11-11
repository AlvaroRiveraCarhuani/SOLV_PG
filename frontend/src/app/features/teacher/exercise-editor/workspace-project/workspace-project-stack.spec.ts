import {
  detectEnvironmentStack,
  detectProjectStack,
  findStackMismatch,
} from './workspace-project-stack';

const reactEnvironment = {
  id: 'tpl-react',
  name: 'React + Vite',
  docker_image: 'solv-local/react:1.0.0',
  stack_config: { runtime_id: 'node', web_platform_id: 'react' },
};

const pythonEnvironment = {
  id: 'tpl-python',
  name: 'Python 3.12',
  docker_image: 'solv-local/python:3.12',
  stack_config: { runtime_id: 'python-312' },
};

const pythonAndAngularProject = [
  { path: 'main.py', content: 'print(1)\n' },
  {
    path: 'package.json',
    content: JSON.stringify({ dependencies: { '@angular/core': '^22.0.0' } }),
  },
];

const reactProject = [
  {
    path: 'package.json',
    content: JSON.stringify({ dependencies: { react: '^19.0.0' } }),
  },
];

describe('workspace project stack', () => {
  it('detecta Python y Angular en el mismo proyecto', () => {
    const detected = detectProjectStack(pythonAndAngularProject);
    expect(detected.techs.map(tech => tech.label)).toEqual(
      expect.arrayContaining(['Python', 'Angular', 'Node.js']),
    );
  });

  it('advierte cuando el proyecto incluye tecnologías que el entorno no declara', () => {
    const mismatch = findStackMismatch(
      detectProjectStack(pythonAndAngularProject),
      detectEnvironmentStack(reactEnvironment),
      reactEnvironment.name,
    );

    expect(mismatch).not.toBeNull();
    expect(mismatch?.unsupportedLabels).toEqual(expect.arrayContaining(['Python', 'Angular']));
    expect(mismatch?.environmentName).toBe('React + Vite');
  });

  it('no advierte cuando el proyecto coincide con el entorno', () => {
    const mismatch = findStackMismatch(
      detectProjectStack(reactProject),
      detectEnvironmentStack(reactEnvironment),
      reactEnvironment.name,
    );

    expect(mismatch).toBeNull();
  });

  it('no advierte cuando el entorno es Python y el proyecto solo tiene Python', () => {
    const mismatch = findStackMismatch(
      detectProjectStack([{ path: 'app.py', content: 'print(1)\n' }]),
      detectEnvironmentStack(pythonEnvironment),
      pythonEnvironment.name,
    );

    expect(mismatch).toBeNull();
  });

  it('reconoce Svelte en el proyecto y en el entorno', () => {
    const svelteEnvironment = {
      id: 'tpl-svelte',
      name: 'Svelte + Vite',
      stack_config: { runtime_id: 'node', web_platform_id: 'svelte' },
    };

    const mismatch = findStackMismatch(
      detectProjectStack([
        { path: 'src/App.svelte', content: '<h1>Hola</h1>\n' },
        { path: 'package.json', content: JSON.stringify({ devDependencies: { svelte: '^5.0.0' } }) },
      ]),
      detectEnvironmentStack(svelteEnvironment),
      svelteEnvironment.name,
    );

    expect(mismatch).toBeNull();
  });

  it('no inventa una incompatibilidad cuando el proyecto no declara stack', () => {
    const mismatch = findStackMismatch(
      detectProjectStack([{ path: 'README.md', content: '# Proyecto\n' }]),
      detectEnvironmentStack(reactEnvironment),
      reactEnvironment.name,
    );

    expect(mismatch).toBeNull();
  });
});
