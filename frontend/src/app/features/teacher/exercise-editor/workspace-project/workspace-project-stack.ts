import {
  ResolvableStackConfig,
  familyForTechId,
  inferTemplateTechnologies,
} from '@features/admin/templates/template-tech-detector.utils';
import { WorkspaceProjectFile } from './workspace-project.utils';

export interface DetectedStackTech {
  id: string;
  family: string;
  label: string;
}

export interface DetectedStack {
  techs: DetectedStackTech[];
}

export interface StackMismatch {
  unsupportedLabels: string[];
  environmentName: string;
}

/**
 * Detecta familias tecnológicas a partir de los archivos del proyecto. No
 * intenta compilar ni ejecutar nada: solo lee extensiones y manifiestos para
 * avisar cuando el entorno elegido no trae las herramientas del proyecto.
 */
export function detectProjectStack(files: WorkspaceProjectFile[]): DetectedStack {
  const techs = new Map<string, DetectedStackTech>();
  const add = (id: string, family: string, label: string): void => {
    if (!techs.has(id)) techs.set(id, { id, family, label });
  };

  const paths = files.map(file => file.path.toLowerCase());
  const packageFile = files.find(file => file.path.toLowerCase().endsWith('package.json'));
  if (packageFile) {
    let dependencies: Record<string, unknown> = {};
    try {
      const parsed = JSON.parse(packageFile.content) as {
        dependencies?: Record<string, unknown>;
        devDependencies?: Record<string, unknown>;
      };
      dependencies = { ...parsed.dependencies, ...parsed.devDependencies };
    } catch {
      // Un package.json ilegible no alcanza para afirmar nada sobre el stack.
    }
    if ('@angular/core' in dependencies) add('angular', 'angular', 'Angular');
    if ('next' in dependencies) add('nextjs', 'react', 'Next.js');
    else if ('react' in dependencies) add('react', 'react', 'React');
    if ('vue' in dependencies) add('vue', 'vue', 'Vue');
    if ('svelte' in dependencies) add('svelte', 'svelte', 'Svelte');
    add('node', 'node', 'Node.js');
  }

  if (
    paths.some(path => path.endsWith('.py')) ||
    paths.some(path => /(^|\/)(requirements\.txt|pyproject\.toml|pipfile|manage\.py)$/.test(path))
  ) {
    add('python', 'python', 'Python');
  }
  if (
    paths.some(path => path.endsWith('.java')) ||
    paths.some(path => /(^|\/)(pom\.xml|build\.gradle|build\.gradle\.kts)$/.test(path))
  ) {
    add('java', 'java', 'Java');
  }
  if (paths.some(path => path.endsWith('.go') || path.endsWith('go.mod'))) {
    add('go', 'go', 'Go');
  }
  if (paths.some(path => /\.(cs|csproj)$/.test(path))) {
    add('dotnet', 'dotnet', '.NET / C#');
  }
  if (paths.some(path => /\.(c|cc|cpp|h|hpp)$/.test(path))) {
    add('cpp', 'cpp', 'C / C++');
  }
  if (paths.some(path => path.endsWith('.php') || path.endsWith('composer.json'))) {
    add('php', 'php', 'PHP');
  }
  if (paths.some(path => path.endsWith('.svelte'))) {
    add('svelte', 'svelte', 'Svelte');
  }
  if (paths.some(path => path.endsWith('.vue'))) {
    add('vue', 'vue', 'Vue');
  }
  if (paths.some(path => path.endsWith('.rs') || path.endsWith('cargo.toml'))) {
    add('rust', 'rust', 'Rust');
  }

  return { techs: Array.from(techs.values()) };
}

/** Resuelve el stack declarado por el entorno homologado elegido. */
export function detectEnvironmentStack(template: {
  name?: string;
  docker_image?: string;
  description?: string;
  stack_config?: ResolvableStackConfig | Record<string, unknown> | null;
}): DetectedStack {
  const inferred = inferTemplateTechnologies({
    name: template.name,
    docker_image: template.docker_image,
    description: template.description,
    stack_config: (template.stack_config ?? null) as ResolvableStackConfig | null,
  });

  const techs = new Map<string, DetectedStackTech>();
  for (const tech of inferred) {
    const family = familyForTechId(tech.id);
    if (!family || techs.has(tech.id)) continue;
    techs.set(tech.id, { id: tech.id, family, label: tech.name });
  }
  return { techs: Array.from(techs.values()) };
}

/**
 * Devuelve las tecnologías del proyecto que el entorno no declara. Un stack
 * desconocido en cualquiera de los dos lados no produce advertencia: preferimos
 * callar antes que inventar una incompatibilidad.
 */
export function findStackMismatch(
  project: DetectedStack,
  environment: DetectedStack,
  environmentName: string,
): StackMismatch | null {
  if (project.techs.length === 0 || environment.techs.length === 0) return null;
  const environmentFamilies = new Set(environment.techs.map(tech => tech.family));
  const unsupported = project.techs.filter(tech => !environmentFamilies.has(tech.family));
  if (unsupported.length === 0) return null;
  const labels = Array.from(new Set(unsupported.map(tech => tech.label)));
  return { unsupportedLabels: labels, environmentName };
}
