import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ComponentLayer } from '../../../../../services/admin-templates.service';

export type ComponentMemoryProfile = 'library' | 'devserver' | 'heavy' | 'light' | 'standard';
export type ComponentCatalogKind = 'backend' | 'web' | 'addon';

export interface EcosystemVersionOption {
  version: string;
  label: string;
  isDefault?: boolean;
}

export interface OfficialEcosystemProfile {
  id: string;
  name: string;
  familyTitle: string;
  nameTemplate: string;
  description: string;
  iconKey: string;
  baseImage: string;
  imageTemplate: string;
  defaultVersion: string;
  suggestedVersions: EcosystemVersionOption[];
  judgeEntrypoint: string;
  toolsProvided: string[];
  memoryProfile: ComponentMemoryProfile;
  nativeInstaller: string;
  compatibleFamilies: string[];
}

export interface OfficialWebProfile {
  id: string;
  name: string;
  nameTemplate?: string;
  description: string;
  iconKey: string;
  baseImage: string;
  installPreset: string;
  packageNames: string[];
  packageTemplate?: string[];
  defaultVersion?: string;
  suggestedVersions?: EcosystemVersionOption[];
  toolsProvided: string[];
  memoryProfile: ComponentMemoryProfile;
}

export interface OfficialAddonPreset {
  id: string;
  name: string;
  description: string;
  iconKey: string;
  compatibleFamilies: string[];
  installPreset: string;
  packageNames: string[];
  toolsProvided: string[];
  memoryProfile: ComponentMemoryProfile;
}

export const OFFICIAL_ECOSYSTEM_PROFILES: OfficialEcosystemProfile[] = [
  {
    id: 'c-gcc13',
    name: 'C (GCC 13)',
    familyTitle: 'C (GCC)',
    nameTemplate: 'C (GCC {version})',
    description: 'Compilador oficial gcc (C17) con libc y herramientas make para programación estructurada y sistemas',
    iconKey: 'c',
    baseImage: 'gcc:13.2-bookworm',
    imageTemplate: 'gcc:{version}-bookworm',
    defaultVersion: '13.2',
    suggestedVersions: [
      { version: '14.2', label: 'GCC 14.2 (Reciente)' },
      { version: '13.2', label: 'GCC 13.2 (Recomendada)', isDefault: true },
      { version: '12.4', label: 'GCC 12.4 ( Clásica)' }
    ],
    judgeEntrypoint: 'gcc -O2 solution.c -o solution && ./solution',
    toolsProvided: ['gcc', 'make'],
    memoryProfile: 'light',
    nativeInstaller: 'apt',
    compatibleFamilies: ['c', 'cpp']
  },
  {
    id: 'cpp-gcc13',
    name: 'C++ (G++ 13)',
    familyTitle: 'C++ (G++)',
    nameTemplate: 'C++ (G++ {version})',
    description: 'Compilador oficial g++ (C++20) con biblioteca estándar STL y make para estructuras de datos y algoritmia',
    iconKey: 'cpp',
    baseImage: 'gcc:13.2-bookworm',
    imageTemplate: 'gcc:{version}-bookworm',
    defaultVersion: '13.2',
    suggestedVersions: [
      { version: '14.2', label: 'G++ 14.2 (C++23 / Reciente)' },
      { version: '13.2', label: 'G++ 13.2 (C++20 Recomendada)', isDefault: true },
      { version: '12.4', label: 'G++ 12.4 (C++17 Clásica)' }
    ],
    judgeEntrypoint: 'g++ -O2 solution.cpp -o solution && ./solution',
    toolsProvided: ['g++', 'gcc', 'make'],
    memoryProfile: 'light',
    nativeInstaller: 'apt',
    compatibleFamilies: ['cpp', 'c']
  },
  {
    id: 'python-312',
    name: 'Python 3.12',
    familyTitle: 'Python',
    nameTemplate: 'Python {version}',
    description: 'Entorno oficial con pip para desarrollo general, APIs y ciencia de datos',
    iconKey: 'python',
    baseImage: 'python:3.12-slim-bookworm',
    imageTemplate: 'python:{version}-slim-bookworm',
    defaultVersion: '3.12',
    suggestedVersions: [
      { version: '3.13', label: 'Python 3.13 (Reciente)' },
      { version: '3.12', label: 'Python 3.12 (Recomendada)', isDefault: true },
      { version: '3.11', label: 'Python 3.11 (Estable)' },
      { version: '3.10', label: 'Python 3.10 (Compatibilidad)' }
    ],
    judgeEntrypoint: 'python3 solution.py',
    toolsProvided: ['python3', 'pip'],
    memoryProfile: 'standard',
    nativeInstaller: 'pip',
    compatibleFamilies: ['python']
  },
  {
    id: 'java-21',
    name: 'Java 21 LTS',
    familyTitle: 'Java (OpenJDK Temurin)',
    nameTemplate: 'Java {version} LTS',
    description: 'Entorno OpenJDK Temurin para programación orientada a objetos y sistemas empresariales',
    iconKey: 'java',
    baseImage: 'eclipse-temurin:21-jdk-jammy',
    imageTemplate: 'eclipse-temurin:{version}-jdk-jammy',
    defaultVersion: '21',
    suggestedVersions: [
      { version: '21', label: 'Java 21 LTS (Recomendada)', isDefault: true },
      { version: '17', label: 'Java 17 LTS (Clásica)' },
      { version: '11', label: 'Java 11 LTS (Legado)' }
    ],
    judgeEntrypoint: 'javac Solution.java && java -Xmx256m Solution',
    toolsProvided: ['java', 'javac', 'jar'],
    memoryProfile: 'heavy',
    nativeInstaller: 'apt',
    compatibleFamilies: ['java']
  },
  {
    id: 'node-20',
    name: 'Node.js 20 LTS',
    familyTitle: 'Node.js',
    nameTemplate: 'Node.js {version} LTS',
    description: 'Entorno JavaScript y TypeScript con gestores npm, npx, pnpm y yarn (corepack)',
    iconKey: 'node',
    baseImage: 'node:20-bookworm-slim',
    imageTemplate: 'node:{version}-bookworm-slim',
    defaultVersion: '20',
    suggestedVersions: [
      { version: '22', label: 'Node.js 22 LTS (Reciente)' },
      { version: '20', label: 'Node.js 20 LTS (Recomendada)', isDefault: true },
      { version: '18', label: 'Node.js 18 LTS (Compatibilidad)' }
    ],
    judgeEntrypoint: 'node solution.js',
    toolsProvided: ['node', 'npm', 'npx'],
    memoryProfile: 'standard',
    nativeInstaller: 'npm',
    compatibleFamilies: ['node', 'web']
  },
  {
    id: 'go-122',
    name: 'Go 1.22',
    familyTitle: 'Go (Golang)',
    nameTemplate: 'Go {version}',
    description: 'Compilador oficial de Go con herramientas de formateo y concurrencia nativa',
    iconKey: 'go',
    baseImage: 'golang:1.22-bookworm',
    imageTemplate: 'golang:{version}-bookworm',
    defaultVersion: '1.22',
    suggestedVersions: [
      { version: '1.23', label: 'Go 1.23 (Reciente)' },
      { version: '1.22', label: 'Go 1.22 (Recomendada)', isDefault: true },
      { version: '1.21', label: 'Go 1.21 (Estable)' }
    ],
    judgeEntrypoint: 'go run main.go',
    toolsProvided: ['go', 'gofmt'],
    memoryProfile: 'standard',
    nativeInstaller: 'go',
    compatibleFamilies: ['go']
  },
  {
    id: 'rust-stable',
    name: 'Rust',
    familyTitle: 'Rust (Cargo)',
    nameTemplate: 'Rust {version}',
    description: 'Compilador rustc y gestor Cargo para programación de sistemas segura',
    iconKey: 'rust',
    baseImage: 'rust:1.77-slim-bookworm',
    imageTemplate: 'rust:{version}-slim-bookworm',
    defaultVersion: '1.77',
    suggestedVersions: [
      { version: '1.81', label: 'Rust 1.81 (Reciente)' },
      { version: '1.77', label: 'Rust 1.77 (Recomendada)', isDefault: true },
      { version: '1.75', label: 'Rust 1.75 (Estable)' }
    ],
    judgeEntrypoint: 'rustc -O solution.rs -o solution && ./solution',
    toolsProvided: ['rustc', 'cargo'],
    memoryProfile: 'standard',
    nativeInstaller: 'cargo',
    compatibleFamilies: ['rust']
  },
  {
    id: 'php-83',
    name: 'PHP 8.3 CLI',
    familyTitle: 'PHP CLI',
    nameTemplate: 'PHP {version} CLI',
    description: 'Intérprete oficial PHP para desarrollo backend y aplicaciones web',
    iconKey: 'php',
    baseImage: 'php:8.3-cli-bookworm',
    imageTemplate: 'php:{version}-cli-bookworm',
    defaultVersion: '8.3',
    suggestedVersions: [
      { version: '8.3', label: 'PHP 8.3 (Recomendada)', isDefault: true },
      { version: '8.2', label: 'PHP 8.2 (Estable)' },
      { version: '8.1', label: 'PHP 8.1 (Compatibilidad)' }
    ],
    judgeEntrypoint: 'php solution.php',
    toolsProvided: ['php'],
    memoryProfile: 'standard',
    nativeInstaller: 'composer',
    compatibleFamilies: ['php']
  },
  {
    id: 'dotnet-8',
    name: '.NET 8 SDK (C#)',
    familyTitle: '.NET SDK (C#)',
    nameTemplate: '.NET {version} SDK (C#)',
    description: 'SDK oficial de Microsoft .NET para desarrollo en C# y aplicaciones empresariales',
    iconKey: 'dotnet',
    baseImage: 'mcr.microsoft.com/dotnet/sdk:8.0-bookworm-slim',
    imageTemplate: 'mcr.microsoft.com/dotnet/sdk:{version}-bookworm-slim',
    defaultVersion: '8.0',
    suggestedVersions: [
      { version: '9.0', label: '.NET 9.0 (Reciente)' },
      { version: '8.0', label: '.NET 8.0 LTS (Recomendada)', isDefault: true },
      { version: '7.0', label: '.NET 7.0 (Compatibilidad)' }
    ],
    judgeEntrypoint: 'dotnet run',
    toolsProvided: ['dotnet'],
    memoryProfile: 'heavy',
    nativeInstaller: 'dotnet',
    compatibleFamilies: ['dotnet']
  }
];

export const OFFICIAL_WEB_PROFILES: OfficialWebProfile[] = [
  {
    id: 'angular-web',
    name: 'Angular',
    nameTemplate: 'Angular {version}',
    description: 'Plataforma de desarrollo Frontend SPA con Angular CLI y soporte TypeScript',
    iconKey: 'angular',
    baseImage: 'node:20-bookworm-slim',
    installPreset: 'npm',
    packageNames: ['@angular/cli'],
    packageTemplate: ['@angular/cli@{version}'],
    defaultVersion: '18',
    suggestedVersions: [
      { version: '19', label: 'Angular 19 (Reciente)' },
      { version: '18', label: 'Angular 18 LTS (Recomendada)', isDefault: true },
      { version: '17', label: 'Angular 17 LTS (Compatibilidad)' }
    ],
    toolsProvided: ['node', 'npm', 'ng'],
    memoryProfile: 'standard'
  },
  {
    id: 'react-vite',
    name: 'React + Vite',
    nameTemplate: 'React + Vite {version}',
    description: 'Entorno ligero de interfaz web con React, Vite y recarga instantánea',
    iconKey: 'react',
    baseImage: 'node:20-bookworm-slim',
    installPreset: 'npm',
    packageNames: ['vite', 'typescript'],
    packageTemplate: ['vite@{version}', 'typescript'],
    defaultVersion: '5',
    suggestedVersions: [
      { version: '6', label: 'Vite 6 (Reciente)' },
      { version: '5', label: 'Vite 5 (Recomendada)', isDefault: true }
    ],
    toolsProvided: ['node', 'npm', 'npx'],
    memoryProfile: 'standard'
  },
  {
    id: 'vue-vite',
    name: 'Vue 3 + Vite',
    nameTemplate: 'Vue 3 + Vite {version}',
    description: 'Framework progresivo para interfaces web reactivas con Vite',
    iconKey: 'vue',
    baseImage: 'node:20-bookworm-slim',
    installPreset: 'npm',
    packageNames: ['vite', '@vue/cli'],
    packageTemplate: ['vite@{version}', '@vue/cli'],
    defaultVersion: '5',
    suggestedVersions: [
      { version: '6', label: 'Vite 6 (Reciente)' },
      { version: '5', label: 'Vite 5 (Recomendada)', isDefault: true }
    ],
    toolsProvided: ['node', 'npm'],
    memoryProfile: 'standard'
  },
  {
    id: 'vite-web',
    name: 'Vite (HTML / TS / JS)',
    nameTemplate: 'Vite {version} (HTML / TS / JS)',
    description: 'Empaquetador web moderno ultrarrápido para proyectos TypeScript, JavaScript y CSS',
    iconKey: 'vite',
    baseImage: 'node:20-bookworm-slim',
    installPreset: 'npm',
    packageNames: ['vite', 'typescript'],
    packageTemplate: ['vite@{version}', 'typescript'],
    defaultVersion: '5',
    suggestedVersions: [
      { version: '6', label: 'Vite 6 (Reciente)' },
      { version: '5', label: 'Vite 5 (Recomendada)', isDefault: true }
    ],
    toolsProvided: ['node', 'npm', 'npx'],
    memoryProfile: 'light'
  }
];

export const OFFICIAL_ADDON_PRESETS: OfficialAddonPreset[] = [
  {
    id: 'cpp-debug-valgrind',
    name: 'Depuración y Memoria (GDB + Valgrind)',
    description: 'Depurador paso a paso GDB y detector de fugas de memoria Valgrind para prácticas de punteros y estructuras de datos',
    iconKey: 'cpp',
    compatibleFamilies: ['c', 'cpp'],
    installPreset: 'apt',
    packageNames: ['gdb', 'valgrind'],
    toolsProvided: ['gdb', 'valgrind'],
    memoryProfile: 'library'
  },
  {
    id: 'cpp-cmake',
    name: 'Construcción CMake + Ninja',
    description: 'Sistema de generación de proyectos multiplataforma CMake y constructor rápido Ninja para proyectos C/C++',
    iconKey: 'cpp',
    compatibleFamilies: ['c', 'cpp'],
    installPreset: 'apt',
    packageNames: ['cmake', 'ninja-build'],
    toolsProvided: ['cmake', 'ninja'],
    memoryProfile: 'library'
  },
  {
    id: 'cpp-googletest',
    name: 'Pruebas Unitarias C/C++ (GoogleTest)',
    description: 'Cabeceras y bibliotecas de GoogleTest junto con pkg-config para evaluación automatizada en C y C++',
    iconKey: 'cpp',
    compatibleFamilies: ['c', 'cpp'],
    installPreset: 'apt',
    packageNames: ['libgtest-dev', 'pkg-config'],
    toolsProvided: ['pkg-config', 'libgtest-dev'],
    memoryProfile: 'library'
  },
  {
    id: 'fastapi',
    name: 'FastAPI + Uvicorn',
    description: 'Framework moderno de alto rendimiento para crear APIs REST en Python',
    iconKey: 'fastapi',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['fastapi', 'uvicorn'],
    toolsProvided: ['python3', 'pip'],
    memoryProfile: 'library'
  },
  {
    id: 'django',
    name: 'Django 5',
    description: 'Framework web completo en Python con ORM integrado y panel administrativo',
    iconKey: 'django',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['django', 'djangorestframework'],
    toolsProvided: ['python3', 'pip'],
    memoryProfile: 'library'
  },
  {
    id: 'flask',
    name: 'Flask',
    description: 'Micro-framework ligero para aplicaciones web y servicios en Python',
    iconKey: 'flask',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['flask'],
    toolsProvided: ['python3', 'pip'],
    memoryProfile: 'library'
  },
  {
    id: 'streamlit',
    name: 'Streamlit (UI en Python)',
    description: 'Permite crear interfaces web interactivas directamente en Python sin HTML/JS',
    iconKey: 'streamlit',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['streamlit'],
    toolsProvided: ['python3', 'pip'],
    memoryProfile: 'devserver'
  },
  {
    id: 'python-datascience',
    name: 'Ciencia de Datos (NumPy + Pandas + Pytest)',
    description: 'Librerías esenciales para procesamiento numérico, tablas y pruebas automatizadas en Python',
    iconKey: 'python',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['numpy', 'pandas', 'pytest'],
    toolsProvided: ['python3', 'pytest'],
    memoryProfile: 'heavy'
  },
  {
    id: 'typescript-tools',
    name: 'TypeScript + Linters',
    description: 'Compilador tsc y herramientas de tipado estático para proyectos web y Node.js',
    iconKey: 'typescript',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['typescript', 'ts-node'],
    toolsProvided: ['node', 'tsc'],
    memoryProfile: 'library'
  },
  {
    id: 'spring-maven',
    name: 'Maven + Gestión Java',
    description: 'Gestor de dependencias y construcción de proyectos para ecosistemas Java y Spring',
    iconKey: 'spring',
    compatibleFamilies: ['java'],
    installPreset: 'apt',
    packageNames: ['maven'],
    toolsProvided: ['java', 'mvn'],
    memoryProfile: 'library'
  },
  {
    id: 'laravel-installer',
    name: 'Laravel Installer',
    description: 'Herramienta oficial para creación de proyectos web con Laravel en PHP',
    iconKey: 'laravel',
    compatibleFamilies: ['php'],
    installPreset: 'composer',
    packageNames: ['laravel/installer'],
    toolsProvided: ['php'],
    memoryProfile: 'library'
  }
];

export const CATALOG_KIND_OPTIONS: ComboboxOption<ComponentCatalogKind>[] = [
  { id: 'backend', label: 'Columna 1 · Lenguaje Principal (C, C++, Python, Java, Go, Rust...)', value: 'backend' },
  { id: 'web', label: 'Columna 1 · Interfaz Web / Frontend (Angular, React, Vue...)', value: 'web' },
  { id: 'addon', label: 'Columna 2 · Complemento, Framework o Librería', value: 'addon' }
];

export const LAYER_OPTIONS: ComboboxOption<ComponentLayer>[] = [
  { id: 'runtime', label: '1. Tecnología Principal o Interfaz Web', value: 'runtime' },
  { id: 'addon', label: '2. Complemento / Framework / Librería', value: 'addon' }
];

export const SUB_CATEGORY_OPTIONS: ComboboxOption<string>[] = [
  { id: 'backend', label: 'Lenguaje Principal (Backend / General)', value: 'backend' },
  { id: 'web', label: 'Plataforma Web / Frontend', value: 'web' }
];

export const INSTALL_PRESET_OPTIONS: ComboboxOption<string>[] = [
  { id: 'none', label: 'Ninguna (La imagen base ya incluye todo)', value: 'none' },
  { id: 'pip', label: 'Automático: Paquete de Python (pip)', value: 'pip' },
  { id: 'uv', label: 'Automático: Paquete de Python ultrarrápido (uv)', value: 'uv' },
  { id: 'npm', label: 'Automático: Paquete de Node.js (npm / npx)', value: 'npm' },
  { id: 'pnpm', label: 'Automático: Gestor rápido de Node.js (pnpm)', value: 'pnpm' },
  { id: 'yarn', label: 'Automático: Gestor clásico de Node.js (yarn)', value: 'yarn' },
  { id: 'bun', label: 'Automático: Gestor ultrarrápido JS/TS (bun)', value: 'bun' },
  { id: 'apt', label: 'Automático: Programa del Sistema (apt / apk)', value: 'apt' },
  { id: 'cargo', label: 'Automático: Paquete de Rust (cargo)', value: 'cargo' },
  { id: 'go', label: 'Automático: Paquete de Go (go install)', value: 'go' },
  { id: 'composer', label: 'Automático: Paquete de PHP (composer)', value: 'composer' },
  { id: 'dotnet', label: 'Automático: Herramienta global .NET (dotnet tool)', value: 'dotnet' },
  { id: 'custom', label: 'Avanzado: Instrucción personalizada (RUN)', value: 'custom' }
];

export const DATABASE_ENGINE_OPTIONS: ComboboxOption<string>[] = [
  { id: 'postgres', label: 'PostgreSQL', value: 'postgres' },
  { id: 'mysql', label: 'MySQL / MariaDB', value: 'mysql' },
  { id: 'sqlite', label: 'SQLite (Archivo local)', value: 'sqlite' },
  { id: 'mongodb', label: 'MongoDB', value: 'mongodb' },
  { id: 'redis', label: 'Redis (Caché)', value: 'redis' }
];

export const ADDON_MEMORY_PROFILE_OPTIONS: ComboboxOption<ComponentMemoryProfile>[] = [
  { id: 'library', label: 'Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)', value: 'library' },
  { id: 'devserver', label: 'Servidor de desarrollo activo en segundo plano (ej. Vite, Webpack, CLI en vivo)', value: 'devserver' },
  { id: 'heavy', label: 'Motor pesado en memoria (ej. Ciencia de Datos / Machine Learning / JVM)', value: 'heavy' }
];

export const RUNTIME_MEMORY_PROFILE_OPTIONS: ComboboxOption<ComponentMemoryProfile>[] = [
  { id: 'light', label: 'Ligero — Compilador nativo o consola (ej. C, C++, Bash)', value: 'light' },
  { id: 'standard', label: 'Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)', value: 'standard' },
  { id: 'heavy', label: 'Intensivo — Máquina Virtual o SDK pesado (ej. Java JVM, .NET)', value: 'heavy' }
];
