import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import {
  ComponentLayer,
  InstallMode,
  TemplateComponent
} from '../../../../../services/admin-templates.service';
import { CuratedIconMeta } from '../../../tech-logo/tech-logo.component';
import {
  ComponentCatalogKind,
  OfficialAddonPreset,
  OfficialEcosystemProfile
} from './component-catalog.presets';

export interface ComponentUiCopy {
  modalTitle: string;
  contextBadgeLabel: string;
  namePlaceholder: string;
  descPlaceholder: string;
  baseImagePlaceholder: string;
  packagesPlaceholder: string;
  toolsPlaceholder: string;
}

export function slugifyComponentName(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function computeComponentUiCopy(
  kind: ComponentCatalogKind,
  isEditing: boolean,
  installPreset: string
): ComponentUiCopy {
  let modalTitle: string;
  let contextBadgeLabel: string;
  let namePlaceholder: string;
  let descPlaceholder: string;
  let baseImagePlaceholder: string;
  let toolsPlaceholder: string;

  if (kind === 'backend') {
    modalTitle = isEditing ? 'Editar Lenguaje Principal' : 'Registrar Lenguaje Principal';
    contextBadgeLabel = 'Columna 1 · Lenguaje Principal';
    namePlaceholder = 'Ej: C (GCC 13) / C++ (G++ 13) / Python 3.12 / Rust 1.81';
    descPlaceholder = 'Ej: Intérprete o compilador oficial para prácticas y evaluaciones';
    baseImagePlaceholder = 'Ej: python:3.12-slim-bookworm o gcc:13.2-bookworm';
    toolsPlaceholder = 'Ej: python3, gcc, g++, rustc';
  } else if (kind === 'web') {
    modalTitle = isEditing ? 'Editar Plataforma Web / Frontend' : 'Registrar Plataforma Web / Frontend';
    contextBadgeLabel = 'Columna 1 · Interfaz Web / Frontend';
    namePlaceholder = 'Ej: Angular 18 / React + Vite / Vue 3';
    descPlaceholder = 'Ej: Entorno Node.js con servidor de desarrollo web';
    baseImagePlaceholder = 'Ej: node:20-bookworm-slim';
    toolsPlaceholder = 'Ej: node, npm, ng';
  } else if (kind === 'database') {
    modalTitle = isEditing ? 'Editar Motor de Base de Datos' : 'Registrar Motor de Base de Datos';
    contextBadgeLabel = 'Columna 3 · Motor de Base de Datos';
    namePlaceholder = 'Ej: MySQL 8.4 / MariaDB 11.4 / MongoDB 7.0 / Redis 7.2';
    descPlaceholder = 'Ej: Motor de base de datos auto-orquestado y aislado por estudiante';
    baseImagePlaceholder = 'Ej: mysql:8.4, mariadb:11.4, mongo:7.0 o redis:7.2-alpine';
    toolsPlaceholder = 'Ej: mysql, mongosh, redis-cli, psql';
  } else {
    modalTitle = isEditing ? 'Editar Complemento del Catálogo' : 'Registrar Complemento o Librería';
    contextBadgeLabel = 'Columna 2 · Complemento / Librería';
    namePlaceholder = 'Ej: Depuración GDB + Valgrind / CMake / FastAPI / Pytest';
    descPlaceholder = 'Ej: Librería o herramienta adicional para el entorno';
    baseImagePlaceholder = 'Ej: python:3.12-slim-bookworm';
    toolsPlaceholder = 'Ej: pytest, uvicorn, gdb, valgrind, cmake';
  }

  let packagesPlaceholder = 'Ej: fastapi uvicorn pytest';
  if (installPreset === 'npm' || installPreset === 'pnpm' || installPreset === 'yarn' || installPreset === 'bun') {
    packagesPlaceholder = 'Ej: @angular/cli@18 o @angular/material';
  } else if (installPreset === 'apt') {
    packagesPlaceholder = 'Ej: gdb valgrind cmake';
  } else if (installPreset === 'cargo') {
    packagesPlaceholder = 'Ej: serde tokio';
  } else if (installPreset === 'go') {
    packagesPlaceholder = 'Ej: golang.org/x/tools/gopls@latest';
  } else if (installPreset === 'composer') {
    packagesPlaceholder = 'Ej: laravel/installer';
  }

  return {
    modalTitle,
    contextBadgeLabel,
    namePlaceholder,
    descPlaceholder,
    baseImagePlaceholder,
    packagesPlaceholder,
    toolsPlaceholder
  };
}

export function filterInstallPresetOptionsByFamily(
  family: string,
  isRuntime: boolean,
  allowCustom: boolean,
  installPresetOptions: ComboboxOption<string>[]
): ComboboxOption<string>[] {
  const allowedIds = new Set<string>();

  if (isRuntime) {
    allowedIds.add('none');
  }

  switch (family) {
    case 'python':
      allowedIds.add('pip');
      allowedIds.add('uv');
      allowedIds.add('apt');
      break;
    case 'node':
    case 'web':
    case 'typescript':
    case 'angular':
    case 'react':
    case 'vue':
      allowedIds.add('npm');
      allowedIds.add('pnpm');
      allowedIds.add('yarn');
      allowedIds.add('bun');
      allowedIds.add('apt');
      break;
    case 'rust':
      allowedIds.add('cargo');
      allowedIds.add('apt');
      break;
    case 'go':
      allowedIds.add('go');
      allowedIds.add('apt');
      break;
    case 'php':
      allowedIds.add('composer');
      allowedIds.add('apt');
      break;
    case 'dotnet':
      allowedIds.add('dotnet');
      allowedIds.add('apt');
      break;
    case 'c':
    case 'cpp':
    case 'java':
      allowedIds.add('apt');
      break;
    default:
      for (const opt of installPresetOptions) {
        allowedIds.add(opt.id);
      }
      break;
  }

  if (allowCustom) {
    allowedIds.add('custom');
  }

  return installPresetOptions.filter(o => allowedIds.has(o.id));
}

export function resolveEcosystemFamilyFromContext(params: {
  layer: ComponentLayer;
  subCategory: string;
  compatibleRuntime: string;
  availableRuntimes: TemplateComponent[];
  iconKey: string;
  name: string;
  baseImage: string;
}): string {
  const { layer, subCategory, compatibleRuntime, availableRuntimes, iconKey, name, baseImage } = params;
  if (layer === 'addon') {
    const key = (compatibleRuntime || '').trim().toLowerCase();
    if (!key || key === 'universal') return 'universal';
    const matched = (availableRuntimes || []).find(r =>
      r.slug?.toLowerCase() === key ||
      r.icon_key?.toLowerCase() === key ||
      r.name?.toLowerCase() === key
    );
    if (matched?.sub_category === 'web') return 'web';
    const combined = `${key} ${matched?.icon_key || ''} ${matched?.name || ''} ${(matched?.compatible_runtimes || []).join(' ')}`.toLowerCase();
    if (/\bc\+\+|cpp|gcc|clang|\bc\b/.test(combined)) return 'cpp';
    if (/python/.test(combined)) return 'python';
    if (/node|angular|react|vue|typescript|javascript/.test(combined)) return 'node';
    if (/rust/.test(combined)) return 'rust';
    if (/\bgo\b|golang/.test(combined)) return 'go';
    if (/php|laravel/.test(combined)) return 'php';
    if (/java|spring|temurin|openjdk/.test(combined)) return 'java';
    if (/dotnet|\.net|c#/.test(combined)) return 'dotnet';
    return key;
  }

  if (subCategory === 'web') return 'web';
  const combined = `${iconKey} ${name} ${baseImage}`.toLowerCase();
  if (/\bc\+\+|cpp|gcc|clang|\bc\b/.test(combined)) return 'cpp';
  if (/python/.test(combined)) return 'python';
  if (/node|typescript/.test(combined)) return 'node';
  if (/rust/.test(combined)) return 'rust';
  if (/\bgo\b|golang/.test(combined)) return 'go';
  if (/php/.test(combined)) return 'php';
  if (/java|temurin|openjdk/.test(combined)) return 'java';
  if (/dotnet|\.net/.test(combined)) return 'dotnet';
  return 'universal';
}

export function findCatalogMatchInList(
  list: TemplateComponent[],
  presetId: string,
  presetName: string
): TemplateComponent | null {
  const safeList = list || [];
  const targetSlug = slugifyComponentName(presetName);
  const idLow = presetId.toLowerCase();
  return (
    safeList.find(
      c =>
        c.slug?.toLowerCase() === idLow ||
        c.slug?.toLowerCase() === targetSlug ||
        c.name?.trim().toLowerCase() === presetName.trim().toLowerCase()
    ) || null
  );
}

export function resolveDynamicWebBaseImageFromCatalog(
  availableRuntimes: TemplateComponent[],
  officialProfiles: OfficialEcosystemProfile[]
): string {
  const runtimes = (availableRuntimes || []).filter(r => r.layer === 'runtime' && r.is_active !== false);
  const nodeRuntime = runtimes.find(r =>
    r.icon_key?.toLowerCase() === 'node' ||
    r.slug?.toLowerCase().startsWith('node') ||
    r.base_image?.toLowerCase().startsWith('node:')
  );
  if (nodeRuntime?.base_image?.trim()) {
    return nodeRuntime.base_image.trim();
  }
  const nodeProfile = officialProfiles.find(p => p.id === 'node-20');
  return nodeProfile?.baseImage || 'node:20-bookworm-slim';
}

export function filterAddonPresetsByFamily(
  presets: OfficialAddonPreset[],
  family: string
): OfficialAddonPreset[] {
  if (!family || family === 'universal') {
    return presets;
  }
  const familyAliases = new Set<string>([family]);
  if (family === 'c' || family === 'cpp') {
    familyAliases.add('c');
    familyAliases.add('cpp');
  } else if (family === 'node' || family === 'web') {
    familyAliases.add('node');
    familyAliases.add('web');
  }
  return presets.filter(preset =>
    preset.compatibleFamilies.some(f => familyAliases.has(f))
  );
}

export function filterCuratedIconsByContext(
  curatedIcons: CuratedIconMeta[],
  query: string,
  kind: ComponentCatalogKind,
  family: string
): CuratedIconMeta[] {
  const cleanQuery = (query || '').trim().toLowerCase();
  const databaseKeys = new Set(['postgres', 'mysql', 'sqlite', 'mongodb', 'redis', 'mariadb', 'db']);
  const webOnlyKeys = new Set(['angular', 'react', 'vue', 'vite', 'nextjs']);
  const backendKeys = new Set(['c', 'cpp', 'python', 'java', 'node', 'typescript', 'go', 'rust', 'php', 'dotnet', 'bun', 'deno', 'docker']);

  let pool = curatedIcons;
  if (kind === 'backend') {
    pool = pool.filter(ic => backendKeys.has(ic.key));
  } else if (kind === 'web') {
    pool = pool.filter(ic => webOnlyKeys.has(ic.key) || ic.key === 'typescript' || ic.key === 'node' || ic.key === 'docker');
  } else if (kind === 'database') {
    pool = pool.filter(ic => databaseKeys.has(ic.key) || ic.key === 'docker');
  } else if (kind === 'addon') {
    pool = pool.filter(ic => !databaseKeys.has(ic.key));
    if (family === 'python') {
      const pyAllowed = new Set(['python', 'fastapi', 'django', 'flask', 'streamlit', 'docker']);
      pool = pool.filter(ic => pyAllowed.has(ic.key));
    } else if (family === 'c' || family === 'cpp') {
      const cppAllowed = new Set(['c', 'cpp', 'docker']);
      pool = pool.filter(ic => cppAllowed.has(ic.key));
    } else if (family === 'java') {
      const javaAllowed = new Set(['java', 'spring', 'docker']);
      pool = pool.filter(ic => javaAllowed.has(ic.key));
    } else if (family === 'php') {
      const phpAllowed = new Set(['php', 'laravel', 'docker']);
      pool = pool.filter(ic => phpAllowed.has(ic.key));
    } else if (family === 'node' || family === 'web') {
      const nodeAllowed = new Set(['node', 'typescript', 'angular', 'react', 'vue', 'vite', 'bun', 'deno', 'docker']);
      pool = pool.filter(ic => nodeAllowed.has(ic.key));
    } else if (family === 'rust') {
      const rustAllowed = new Set(['rust', 'docker']);
      pool = pool.filter(ic => rustAllowed.has(ic.key));
    } else if (family === 'go') {
      const goAllowed = new Set(['go', 'docker']);
      pool = pool.filter(ic => goAllowed.has(ic.key));
    }
  }

  if (!cleanQuery) return pool;
  return pool.filter(
    ic => ic.label.toLowerCase().includes(cleanQuery) || ic.key.toLowerCase().includes(cleanQuery)
  );
}

export function resolveAutoIconKeyFromContext(
  nameVal: string,
  compatRuntimeKey: string,
  layer: ComponentLayer,
  availableRuntimes: TemplateComponent[],
  curatedIcons: CuratedIconMeta[]
): string {
  const normalized = (nameVal || '').trim().toLowerCase();

  if (/c\+\+|g\+\+|cpp|valgrind|gdb|cmake/i.test(normalized)) {
    return 'cpp';
  }
  if (/^c\s*\(|lenguaje c\b|\bgcc\b/i.test(normalized) && !/c\+\+/i.test(normalized)) {
    return 'c';
  }
  if (/pytest|numpy|pandas|sqlalchemy|psycopg|pydantic|uvicorn/i.test(normalized)) {
    return 'python';
  }
  if (/material|primeng|rxjs/i.test(normalized)) {
    return 'angular';
  }
  if (/junit|maven|gradle/i.test(normalized)) {
    return 'java';
  }
  if (/tokio|serde|cargo/i.test(normalized)) {
    return 'rust';
  }

  if (normalized) {
    const sortedCurated = [...curatedIcons]
      .filter(ic => ic.key !== 'docker')
      .sort((a, b) => b.key.length - a.key.length);

    const matchToken = (token: string): boolean => {
      if (!token || token.length < 2) return false;
      if (token.length <= 3) {
        const escaped = token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const re = new RegExp(`(^|[^a-z0-9])${escaped}([^a-z0-9]|$)`, 'i');
        return re.test(normalized);
      }
      return normalized.includes(token);
    };

    for (const ic of sortedCurated) {
      const key = ic.key.toLowerCase();
      const firstWord = ic.label.split(/[\s/]+/)[0].toLowerCase();
      if (matchToken(key) || matchToken(firstWord)) {
        return ic.key;
      }
    }
  }

  if (layer === 'addon' && compatRuntimeKey) {
    const lowerCompat = compatRuntimeKey.trim().toLowerCase();
    if (lowerCompat && lowerCompat !== 'universal') {
      const matchedRt = (availableRuntimes || []).find(r =>
        r.slug?.toLowerCase() === lowerCompat ||
        r.icon_key?.toLowerCase() === lowerCompat ||
        r.name?.toLowerCase() === lowerCompat
      );
      if (matchedRt?.icon_key && matchedRt.icon_key !== 'docker') {
        return matchedRt.icon_key;
      }
      const matchedCurated = curatedIcons.find(ic =>
        ic.key.toLowerCase() === lowerCompat ||
        ic.label.toLowerCase() === lowerCompat
      );
      if (matchedCurated && matchedCurated.key !== 'docker') {
        return matchedCurated.key;
      }
    }
  }

  return 'docker';
}

export function inferAddonInstallPresetFromRuntime(
  runtimeKey: string,
  availableRuntimes: TemplateComponent[]
): string | null {
  const key = runtimeKey.toLowerCase();
  const matched = (availableRuntimes || []).find(r =>
    r.slug?.toLowerCase() === key ||
    r.icon_key?.toLowerCase() === key ||
    r.name?.toLowerCase() === key
  );
  if (matched?.sub_category === 'web') {
    return 'npm';
  }
  if (matched?.install_mode && matched.install_mode !== 'none' && matched.install_mode !== 'custom') {
    return matched.install_mode;
  }
  const combined = `${key} ${matched?.name || ''} ${matched?.icon_key || ''}`.toLowerCase();
  if (/node|angular|react|vue|javascript|typescript/.test(combined)) return 'npm';
  if (/python/.test(combined)) return 'pip';
  if (/rust/.test(combined)) return 'cargo';
  if (/\bgo\b|golang/.test(combined)) return 'go';
  if (/php/.test(combined)) return 'composer';
  if (/cpp|\bc\b|c\+\+|gcc|linux|ubuntu|debian|java|dotnet/.test(combined)) return 'apt';
  return null;
}

export function synthesizeInstallModeAndSnippet(
  preset: string,
  pkgs: string[],
  customBuildSnippet: string
): { installMode: InstallMode; snippet: string } {
  if (preset === 'none') {
    return { installMode: 'none', snippet: '' };
  }
  switch (preset) {
    case 'pip':
      return { installMode: 'pip', snippet: '' };
    case 'uv':
      return { installMode: 'uv', snippet: `RUN uv pip install --system --no-cache ${pkgs.join(' ')}` };
    case 'npm':
      return { installMode: 'npm', snippet: '' };
    case 'pnpm':
      return { installMode: 'pnpm', snippet: `RUN corepack enable && pnpm add -g ${pkgs.join(' ')}` };
    case 'yarn':
      return { installMode: 'yarn', snippet: `RUN corepack enable && yarn global add ${pkgs.join(' ')}` };
    case 'bun':
      return { installMode: 'bun', snippet: `RUN bun add -g ${pkgs.join(' ')}` };
    case 'apt':
      return { installMode: 'apt', snippet: '' };
    case 'cargo':
      return { installMode: 'cargo', snippet: `RUN cargo install ${pkgs.join(' ')}` };
    case 'go':
      return { installMode: 'go', snippet: `RUN go install ${pkgs.join(' ')}` };
    case 'composer':
      return { installMode: 'composer', snippet: `RUN composer global require ${pkgs.join(' ')}` };
    case 'dotnet':
      return { installMode: 'dotnet', snippet: `RUN dotnet tool install -g ${pkgs.join(' ')}` };
    case 'custom':
      return { installMode: 'custom', snippet: customBuildSnippet.trim() };
    default:
      return { installMode: 'none', snippet: '' };
  }
}

export function computeCompatibleRuntimesForComponent(params: {
  layer: ComponentLayer;
  subCategory: string;
  compatibleRuntime: string;
  iconKey: string;
  editingSlug?: string;
  generatedSlug: string;
}): string[] {
  const { layer, subCategory, compatibleRuntime, iconKey, editingSlug, generatedSlug } = params;
  if (layer === 'database') {
    return ['universal'];
  }
  if (layer === 'addon') {
    const selectedCompat = compatibleRuntime.trim();
    if (
      selectedCompat === 'c' ||
      selectedCompat === 'cpp' ||
      selectedCompat === 'c-gcc13' ||
      selectedCompat === 'cpp-gcc13'
    ) {
      return Array.from(new Set([selectedCompat, 'c', 'cpp']));
    }
    return [selectedCompat];
  }
  let familyKey: string[] = [];
  if (subCategory === 'web') {
    familyKey = ['node', 'web'];
  } else {
    const rawIcon = iconKey.toLowerCase();
    if (rawIcon === 'c' || rawIcon === 'cpp') {
      familyKey = ['c', 'cpp'];
    } else if (rawIcon !== 'docker' && !rawIcon.includes('/') && !rawIcon.includes(':')) {
      familyKey = [rawIcon];
    }
  }
  const baseSlug = editingSlug || generatedSlug;
  return Array.from(new Set([baseSlug, ...familyKey].filter(Boolean)));
}
