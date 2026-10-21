import { Component, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideBoxes, LucideAlertTriangle } from '@lucide/angular';
import {
  AdminTemplatesService,
  ComponentLayer,
  CreateTemplateComponentDTO,
  UpdateTemplateComponentDTO,
  InstallMode,
  TemplateComponent
} from '../../../../../services/admin-templates.service';
import { CURATED_TECH_ICONS, TechLogoComponent } from '../../../tech-logo/tech-logo.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

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

function slugify(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

@Component({
  selector: 'component-form-modal',
  imports: [
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    TechLogoComponent,
    ComboboxComponent,
    LucideBoxes,
    LucideAlertTriangle
  ],
  templateUrl: './component-form-modal.component.html',
  styleUrls: ['./component-form-modal.component.scss']
})
export class ComponentFormModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() defaultLayer: ComponentLayer = 'addon';
  @Input() defaultSubCategory: 'backend' | 'web' = 'backend';
  @Input() defaultCompatibleRuntime: string = '';
  @Input() availableRuntimes: TemplateComponent[] = [];
  @Input() editingComponent: TemplateComponent | null = null;
  @Input() lockContext: boolean = false;

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<TemplateComponent>();

  readonly curatedIcons = CURATED_TECH_ICONS;
  readonly officialEcosystemProfiles = OFFICIAL_ECOSYSTEM_PROFILES;
  readonly officialWebProfiles = OFFICIAL_WEB_PROFILES;
  readonly officialAddonPresets = OFFICIAL_ADDON_PRESETS;
  readonly isContextLocked = signal<boolean>(false);
  readonly showExpertMode = signal<boolean>(false);
  readonly selectedEcosystemProfileId = signal<string>('');
  readonly selectedWebProfileId = signal<string>('');
  readonly selectedAddonPresetId = signal<string>('');
  readonly selectedVersion = signal<string>('');
  readonly isCustomPresetSelected = signal<boolean>(false);
  readonly existingCatalogMatch = signal<TemplateComponent | null>(null);

  readonly layer = signal<ComponentLayer>('addon');
  readonly subCategory = signal<string>('framework');
  readonly name = signal<string>('');
  readonly description = signal<string>('');
  readonly iconKey = signal<string>('docker');
  readonly isIconManuallySelected = signal<boolean>(false);
  readonly customIconUrl = signal<string>('');
  readonly showIconPickerGrid = signal<boolean>(false);
  readonly showCustomIconInput = signal<boolean>(false);
  readonly iconSearchQuery = signal<string>('');
  readonly baseImage = signal<string>('');
  readonly compatibleRuntime = signal<string>('');
  readonly installMode = signal<InstallMode>('pip');
  readonly installPreset = signal<string>('pip');
  readonly packageNamesInput = signal<string>('');
  readonly customBuildSnippet = signal<string>('');
  readonly toolsProvidedInput = signal<string>('');
  readonly serviceEngine = signal<string>('postgres');
  readonly judgeEntrypoint = signal<string>('');
  readonly memoryProfile = signal<ComponentMemoryProfile>('library');
  readonly ramDeltaMB = signal<number>(0);

  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly ecosystemProfileOptions: ComboboxOption<string>[] = OFFICIAL_ECOSYSTEM_PROFILES.map(p => ({
    id: p.id,
    label: `${p.name} — ${p.baseImage}`,
    value: p.id
  }));

  readonly catalogKindOptions: ComboboxOption<ComponentCatalogKind>[] = [
    { id: 'backend', label: 'Columna 1 · Lenguaje Principal (C, C++, Python, Java, Go, Rust...)', value: 'backend' },
    { id: 'web', label: 'Columna 1 · Interfaz Web / Frontend (Angular, React, Vue...)', value: 'web' },
    { id: 'addon', label: 'Columna 2 · Complemento, Framework o Librería', value: 'addon' }
  ];

  readonly catalogKind = computed<ComponentCatalogKind>(() => {
    if (this.layer() === 'addon') return 'addon';
    return this.subCategory() === 'web' ? 'web' : 'backend';
  });

  readonly uiCopy = computed(() => {
    const kind = this.catalogKind();
    const isEditing = !!this.editingComponent;

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
    } else {
      modalTitle = isEditing ? 'Editar Complemento del Catálogo' : 'Registrar Complemento o Librería';
      contextBadgeLabel = 'Columna 2 · Complemento / Librería';
      namePlaceholder = 'Ej: Depuración GDB + Valgrind / CMake / FastAPI / Pytest';
      descPlaceholder = 'Ej: Librería o herramienta adicional para el entorno';
      baseImagePlaceholder = 'Ej: python:3.12-slim-bookworm';
      toolsPlaceholder = 'Ej: pytest, uvicorn, gdb, valgrind, cmake';
    }

    const preset = this.installPreset();
    let packagesPlaceholder = 'Ej: fastapi uvicorn pytest';
    if (preset === 'npm') {
      packagesPlaceholder = 'Ej: @angular/cli@18 o @angular/material';
    } else if (preset === 'apt') {
      packagesPlaceholder = 'Ej: gdb valgrind cmake';
    } else if (preset === 'cargo') {
      packagesPlaceholder = 'Ej: serde tokio';
    } else if (preset === 'go') {
      packagesPlaceholder = 'Ej: golang.org/x/tools/gopls@latest';
    } else if (preset === 'composer') {
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
  });

  readonly layerOptions: ComboboxOption<ComponentLayer>[] = [
    { id: 'runtime', label: '1. Tecnología Principal o Interfaz Web', value: 'runtime' },
    { id: 'addon', label: '2. Complemento / Framework / Librería', value: 'addon' }
  ];

  readonly subCategoryOptions: ComboboxOption<string>[] = [
    { id: 'backend', label: 'Lenguaje Principal (Backend / General)', value: 'backend' },
    { id: 'web', label: 'Plataforma Web / Frontend', value: 'web' }
  ];

  readonly installPresetOptions: ComboboxOption<string>[] = [
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

  /**
   * Filtra los gestores de instalación permitidos según la familia tecnológica activa,
   * bloqueando gestores de otros ecosistemas (ej. impide elegir npm o pip para C/C++).
   */
  readonly filteredInstallPresetOptions = computed<ComboboxOption<string>[]>(() => {
    const family = this.resolvedEcosystemFamily();
    const isRuntime = this.layer() === 'runtime';
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
        // Universal o familia no restringida: permite los gestores estándar
        for (const opt of this.installPresetOptions) {
          allowedIds.add(opt.id);
        }
        break;
    }

    if (this.showExpertMode() || this.installPreset() === 'custom') {
      allowedIds.add('custom');
    }

    return this.installPresetOptions.filter(o => allowedIds.has(o.id));
  });

  readonly resolvedEcosystemFamily = computed<string>(() => {
    if (this.layer() === 'addon') {
      const key = (this.compatibleRuntime() || '').trim().toLowerCase();
      if (!key || key === 'universal') return 'universal';
      const matched = (this.availableRuntimes || []).find(r =>
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

    if (this.subCategory() === 'web') return 'web';
    const combined = `${this.iconKey()} ${this.name()} ${this.baseImage()}`.toLowerCase();
    if (/\bc\+\+|cpp|gcc|clang|\bc\b/.test(combined)) return 'cpp';
    if (/python/.test(combined)) return 'python';
    if (/node|typescript/.test(combined)) return 'node';
    if (/rust/.test(combined)) return 'rust';
    if (/\bgo\b|golang/.test(combined)) return 'go';
    if (/php/.test(combined)) return 'php';
    if (/java|temurin|openjdk/.test(combined)) return 'java';
    if (/dotnet|\.net/.test(combined)) return 'dotnet';
    return 'universal';
  });

  readonly autoResolvedSummary = computed<string>(() => {
    if (this.layer() === 'runtime') {
      const img = this.baseImage().trim() || 'Pendiente de selección';
      const ram = this.ramDeltaMB() || 512;
      const judge = this.judgeEntrypoint().trim();
      return judge
        ? `Imagen: ${img} · Evaluador: ${judge} · Memoria base: ${ram} MB`
        : `Imagen: ${img} · Memoria base: ${ram} MB`;
    }
    const preset = this.installPreset();
    const installerLabel = this.installPresetOptions.find(o => o.id === preset)?.label || preset;
    return `${installerLabel} · Sin recargo de memoria residente (0 MB extra)`;
  });

  readonly databaseEngineOptions: ComboboxOption<string>[] = [
    { id: 'postgres', label: 'PostgreSQL', value: 'postgres' },
    { id: 'mysql', label: 'MySQL / MariaDB', value: 'mysql' },
    { id: 'sqlite', label: 'SQLite (Archivo local)', value: 'sqlite' },
    { id: 'mongodb', label: 'MongoDB', value: 'mongodb' },
    { id: 'redis', label: 'Redis (Caché)', value: 'redis' }
  ];

  readonly addonMemoryProfileOptions: ComboboxOption<ComponentMemoryProfile>[] = [
    { id: 'library', label: 'Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)', value: 'library' },
    { id: 'devserver', label: 'Servidor de desarrollo activo en segundo plano (ej. Vite, Webpack, CLI en vivo)', value: 'devserver' },
    { id: 'heavy', label: 'Motor pesado en memoria (ej. Ciencia de Datos / Machine Learning / JVM)', value: 'heavy' }
  ];

  readonly runtimeMemoryProfileOptions: ComboboxOption<ComponentMemoryProfile>[] = [
    { id: 'light', label: 'Ligero — Compilador nativo o consola (ej. C, C++, Bash)', value: 'light' },
    { id: 'standard', label: 'Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)', value: 'standard' },
    { id: 'heavy', label: 'Intensivo — Máquina Virtual o SDK pesado (ej. Java JVM, .NET)', value: 'heavy' }
  ];

  readonly compatibleRuntimeOptions = computed<ComboboxOption<string>[]>(() => {
    const runtimes = (this.availableRuntimes || []).filter(r => r.layer === 'runtime');
    const seen = new Set<string>();
    const options: ComboboxOption<string>[] = [];

    for (const r of runtimes) {
      const key = (r.slug || r.icon_key || r.name).toLowerCase();
      if (!seen.has(key)) {
        seen.add(key);
        options.push({
          id: key,
          label: r.name,
          value: key
        });
      }
    }

    options.push({
      id: 'universal',
      label: 'Universal (Compatible con cualquier tecnología)',
      value: 'universal'
    });

    return options;
  });

  readonly effectiveIconKey = computed<string>(() => {
    const custom = this.customIconUrl().trim();
    if (custom) {
      return custom;
    }
    return this.iconKey().trim() || 'docker';
  });

  findCatalogMatch(presetId: string, presetName: string): TemplateComponent | null {
    const list = this.availableRuntimes || [];
    const targetSlug = slugify(presetName);
    const idLow = presetId.toLowerCase();
    return (
      list.find(
        c =>
          c.slug?.toLowerCase() === idLow ||
          c.slug?.toLowerCase() === targetSlug ||
          c.name?.trim().toLowerCase() === presetName.trim().toLowerCase()
      ) || null
    );
  }

  isPresetInCatalog(presetId: string, presetName: string): boolean {
    return !!this.findCatalogMatch(presetId, presetName);
  }

  resolveDynamicWebBaseImage(): string {
    const runtimes = (this.availableRuntimes || []).filter(r => r.layer === 'runtime' && r.is_active !== false);
    const nodeRuntime = runtimes.find(r =>
      r.icon_key?.toLowerCase() === 'node' ||
      r.slug?.toLowerCase().startsWith('node') ||
      r.base_image?.toLowerCase().startsWith('node:')
    );
    if (nodeRuntime?.base_image?.trim()) {
      return nodeRuntime.base_image.trim();
    }
    const nodeProfile = this.officialEcosystemProfiles.find(p => p.id === 'node-20');
    return nodeProfile?.baseImage || 'node:20-bookworm-slim';
  }

  readonly availableVersionsForSelectedProfile = computed<EcosystemVersionOption[]>(() => {
    const kind = this.catalogKind();
    if (kind === 'backend') {
      const prof = this.officialEcosystemProfiles.find(p => p.id === this.selectedEcosystemProfileId());
      return prof?.suggestedVersions || [];
    }
    if (kind === 'web') {
      const webProf = this.officialWebProfiles.find(p => p.id === this.selectedWebProfileId());
      return webProf?.suggestedVersions || [];
    }
    return [];
  });

  readonly versionComboboxOptions = computed<ComboboxOption<string>[]>(() => {
    return this.availableVersionsForSelectedProfile().map(v => ({
      id: v.version,
      label: v.label,
      value: v.version
    }));
  });

  onVersionChange(rawVersion: string): void {
    const cleanVer = (rawVersion || '').trim();
    this.selectedVersion.set(cleanVer);
    if (!cleanVer) return;

    const kind = this.catalogKind();
    if (kind === 'backend') {
      const prof = this.officialEcosystemProfiles.find(p => p.id === this.selectedEcosystemProfileId());
      if (prof) {
        const resolvedImg = prof.imageTemplate.replace('{version}', cleanVer);
        const majorLabel = cleanVer.split('.')[0] && (prof.id === 'c-gcc13' || prof.id === 'cpp-gcc13')
          ? cleanVer.split('.')[0]
          : cleanVer;
        const resolvedName = prof.nameTemplate.replace('{version}', majorLabel);
        this.baseImage.set(resolvedImg);
        this.name.set(resolvedName);
        this.existingCatalogMatch.set(this.findCatalogMatch(slugify(resolvedName), resolvedName));
      }
    } else if (kind === 'web') {
      const webProf = this.officialWebProfiles.find(p => p.id === this.selectedWebProfileId());
      if (webProf) {
        if (webProf.packageTemplate && webProf.packageTemplate.length > 0) {
          const pkgs = webProf.packageTemplate.map(t => t.replace('{version}', cleanVer));
          this.packageNamesInput.set(pkgs.join(', '));
        }
        if (webProf.nameTemplate) {
          const isDefault = cleanVer === webProf.defaultVersion;
          const resolvedName = isDefault ? webProf.name : webProf.nameTemplate.replace('{version}', cleanVer);
          this.name.set(resolvedName);
          this.existingCatalogMatch.set(this.findCatalogMatch(slugify(resolvedName), resolvedName));
        }
      }
    }
  }

  readonly sortedEcosystemProfiles = computed<OfficialEcosystemProfile[]>(() => {
    return [...this.officialEcosystemProfiles].sort((a, b) => {
      const aInstalled = this.isPresetInCatalog(a.id, a.name) ? 1 : 0;
      const bInstalled = this.isPresetInCatalog(b.id, b.name) ? 1 : 0;
      return aInstalled - bInstalled;
    });
  });

  readonly sortedWebProfiles = computed<OfficialWebProfile[]>(() => {
    return [...this.officialWebProfiles].sort((a, b) => {
      const aInstalled = this.isPresetInCatalog(a.id, a.name) ? 1 : 0;
      const bInstalled = this.isPresetInCatalog(b.id, b.name) ? 1 : 0;
      return aInstalled - bInstalled;
    });
  });

  readonly contextualAddonPresets = computed<OfficialAddonPreset[]>(() => {
    const family = this.resolvedEcosystemFamily();
    let filtered = this.officialAddonPresets;
    if (family && family !== 'universal') {
      const familyAliases = new Set<string>([family]);
      if (family === 'c' || family === 'cpp') {
        familyAliases.add('c');
        familyAliases.add('cpp');
      } else if (family === 'node' || family === 'web') {
        familyAliases.add('node');
        familyAliases.add('web');
      }
      filtered = this.officialAddonPresets.filter(preset =>
        preset.compatibleFamilies.some(f => familyAliases.has(f))
      );
    }
    return [...filtered].sort((a, b) => {
      const aInstalled = this.isPresetInCatalog(a.id, a.name) ? 1 : 0;
      const bInstalled = this.isPresetInCatalog(b.id, b.name) ? 1 : 0;
      return aInstalled - bInstalled;
    });
  });

  readonly filteredCuratedIcons = computed(() => {
    const query = this.iconSearchQuery().trim().toLowerCase();
    const kind = this.catalogKind();
    const family = this.resolvedEcosystemFamily();

    const databaseKeys = new Set(['postgres', 'mysql', 'sqlite', 'mongodb', 'redis', 'mariadb', 'db']);
    const webOnlyKeys = new Set(['angular', 'react', 'vue', 'vite', 'nextjs']);
    const backendKeys = new Set(['c', 'cpp', 'python', 'java', 'node', 'typescript', 'go', 'rust', 'php', 'dotnet', 'bun', 'deno', 'docker']);

    let pool = this.curatedIcons;
    if (kind === 'backend') {
      pool = pool.filter(ic => backendKeys.has(ic.key));
    } else if (kind === 'web') {
      pool = pool.filter(ic => webOnlyKeys.has(ic.key) || ic.key === 'typescript' || ic.key === 'node' || ic.key === 'docker');
    } else if (kind === 'addon') {
      // Nunca mostrar motores de base de datos de Columna 3 en Complementos de Columna 2
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

    if (!query) return pool;
    return pool.filter(
      ic => ic.label.toLowerCase().includes(query) || ic.key.toLowerCase().includes(query)
    );
  });

  readonly activeIconLabel = computed<string>(() => {
    if (this.customIconUrl().trim()) {
      return 'Icono personalizado (URL / SVG)';
    }
    const key = this.iconKey().trim().toLowerCase();
    const matched = this.curatedIcons.find(ic => ic.key.toLowerCase() === key);
    if (matched) {
      return matched.label;
    }
    return this.iconKey() || 'Docker';
  });

  readonly iconResolutionSource = computed<string>(() => {
    if (this.customIconUrl().trim()) {
      return 'URL personalizada';
    }
    if (this.isIconManuallySelected()) {
      return 'Seleccionado manualmente';
    }
    if (this.iconKey() && this.iconKey() !== 'docker') {
      return 'Detectado automáticamente';
    }
    return 'Icono estándar por defecto';
  });

  ngOnInit(): void {
    this.isContextLocked.set(this.lockContext && !this.editingComponent);

    if (this.editingComponent) {
      const comp = this.editingComponent;
      this.layer.set(comp.layer);
      this.subCategory.set(comp.sub_category || (comp.layer === 'runtime' ? 'backend' : 'framework'));
      this.name.set(comp.name);
      this.description.set(comp.description || '');
      this.iconKey.set(comp.icon_key || 'docker');
      if (comp.icon_key && (comp.icon_key.startsWith('/') || comp.icon_key.startsWith('http'))) {
        this.customIconUrl.set(comp.icon_key);
        this.showCustomIconInput.set(true);
      } else if (comp.icon_key && comp.icon_key !== 'docker') {
        this.isIconManuallySelected.set(true);
      } else {
        this.isIconManuallySelected.set(false);
        this.iconKey.set(this.resolveAutoIconKey(comp.name, comp.compatible_runtimes?.[0] || ''));
      }
      this.baseImage.set(comp.base_image || '');
      if (comp.compatible_runtimes && comp.compatible_runtimes.length > 0) {
        this.compatibleRuntime.set(comp.compatible_runtimes[0]);
      }
      this.installMode.set(comp.install_mode || 'none');
      if (comp.install_mode === 'custom') {
        this.installPreset.set('custom');
        this.customBuildSnippet.set(comp.build_layer_snippet || '');
        this.showExpertMode.set(true);
      } else if (comp.install_mode && comp.install_mode !== 'none') {
        this.installPreset.set(comp.install_mode);
      } else {
        this.installPreset.set('none');
      }
      if (comp.package_names && comp.package_names.length > 0) {
        this.packageNamesInput.set(comp.package_names.join(', '));
      }
      if (comp.tools_provided && comp.tools_provided.length > 0) {
        this.toolsProvidedInput.set(comp.tools_provided.join(', '));
      }
      this.serviceEngine.set(comp.service_engine || 'postgres');
      this.judgeEntrypoint.set(comp.judge_entrypoint || '');
      this.ramDeltaMB.set(comp.ram_delta_mb || 0);

      if (comp.layer === 'addon') {
        if (comp.ram_delta_mb >= 384) {
          this.memoryProfile.set('heavy');
        } else if (comp.ram_delta_mb >= 256) {
          this.memoryProfile.set('devserver');
        } else {
          this.memoryProfile.set('library');
        }
      } else if (comp.layer === 'runtime') {
        if (comp.ram_delta_mb >= 1024) {
          this.memoryProfile.set('heavy');
        } else if (comp.ram_delta_mb <= 384) {
          this.memoryProfile.set('light');
        } else {
          this.memoryProfile.set('standard');
        }
      }
      return;
    }

    const initialKind: ComponentCatalogKind = this.defaultLayer === 'addon'
      ? 'addon'
      : (this.defaultSubCategory === 'web' ? 'web' : 'backend');
    this.onCatalogKindChange(initialKind);
    if (!this.editingComponent && this.defaultLayer === 'addon') {
      if (this.defaultCompatibleRuntime?.trim()) {
        this.onCompatibleRuntimeChange(this.defaultCompatibleRuntime.trim().toLowerCase());
      } else {
        const firstRuntime = (this.availableRuntimes || []).find(r => r.layer === 'runtime' && r.is_active !== false);
        if (firstRuntime) {
          const fallbackKey = (firstRuntime.slug || firstRuntime.icon_key || firstRuntime.name).toLowerCase();
          this.onCompatibleRuntimeChange(fallbackKey);
        }
      }
    } else if (this.defaultCompatibleRuntime?.trim()) {
      this.compatibleRuntime.set(this.defaultCompatibleRuntime.trim().toLowerCase());
      if (initialKind === 'addon') {
        this.onCompatibleRuntimeChange(this.defaultCompatibleRuntime.trim().toLowerCase());
      }
    }
  }

  /**
   * Aplica un Perfil de Ecosistema Oficial preconfigurado (estilo Judge0/Devcontainers)
   * para que el administrador no tenga que escribir imagen Docker, comando de juez ni memoria.
   */
  onSelectEcosystemProfile(profileId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedEcosystemProfileId.set(profileId);
    const profile = this.officialEcosystemProfiles.find(p => p.id === profileId);
    if (!profile) {
      return;
    }
    this.selectedVersion.set(profile.defaultVersion);
    this.existingCatalogMatch.set(this.findCatalogMatch(profile.id, profile.name));
    this.name.set(profile.name);
    this.description.set(profile.description);
    this.iconKey.set(profile.iconKey);
    this.isIconManuallySelected.set(false);
    this.baseImage.set(profile.baseImage);
    this.judgeEntrypoint.set(profile.judgeEntrypoint);
    this.toolsProvidedInput.set(profile.toolsProvided.join(', '));
    this.setMemoryProfile(profile.memoryProfile);
    this.installPreset.set('none');
    this.installMode.set('none');
  }

  onSelectWebProfile(profileId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedWebProfileId.set(profileId);
    const profile = this.officialWebProfiles.find(p => p.id === profileId);
    if (!profile) {
      return;
    }
    this.selectedVersion.set(profile.defaultVersion || '');
    this.existingCatalogMatch.set(this.findCatalogMatch(profile.id, profile.name));
    this.name.set(profile.name);
    this.description.set(profile.description);
    this.iconKey.set(profile.iconKey);
    this.isIconManuallySelected.set(false);
    this.baseImage.set(this.resolveDynamicWebBaseImage());
    this.installPreset.set(profile.installPreset);
    this.installMode.set(profile.installPreset as InstallMode);
    this.packageNamesInput.set(profile.packageNames.join(', '));
    this.toolsProvidedInput.set(profile.toolsProvided.join(', '));
    this.setMemoryProfile(profile.memoryProfile);
  }

  onSelectAddonPreset(presetId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedAddonPresetId.set(presetId);
    const preset = this.officialAddonPresets.find(p => p.id === presetId);
    if (!preset) {
      return;
    }
    this.existingCatalogMatch.set(this.findCatalogMatch(preset.id, preset.name));
    this.name.set(preset.name);
    this.description.set(preset.description);
    this.iconKey.set(preset.iconKey);
    this.isIconManuallySelected.set(false);
    this.installPreset.set(preset.installPreset);
    this.packageNamesInput.set(preset.packageNames.join(', '));
    this.toolsProvidedInput.set(preset.toolsProvided.join(', '));
    this.setMemoryProfile(preset.memoryProfile);
  }

  onSelectCustomMode(): void {
    this.isCustomPresetSelected.set(true);
    this.selectedEcosystemProfileId.set('');
    this.selectedWebProfileId.set('');
    this.selectedAddonPresetId.set('');
    this.selectedVersion.set('');
    this.existingCatalogMatch.set(null);
    this.name.set('');
    this.description.set('');
    this.iconKey.set('docker');
    this.isIconManuallySelected.set(false);
    this.packageNamesInput.set('');
    this.toolsProvidedInput.set('');
    this.judgeEntrypoint.set('');
    if (this.catalogKind() === 'web') {
      this.baseImage.set(this.resolveDynamicWebBaseImage());
      this.installPreset.set('npm');
      this.installMode.set('npm');
    } else if (this.catalogKind() === 'backend') {
      this.baseImage.set('');
      this.installPreset.set('none');
      this.installMode.set('none');
      this.showExpertMode.set(true);
    } else {
      this.baseImage.set('');
    }
  }

  onCatalogKindChange(kind: ComponentCatalogKind): void {
    const dynamicWebImg = this.resolveDynamicWebBaseImage();
    if (kind === 'backend') {
      this.layer.set('runtime');
      this.subCategory.set('backend');
      this.installMode.set('none');
      this.installPreset.set('none');
      this.setMemoryProfile('standard');
      if (!this.editingComponent && this.baseImage().trim() === dynamicWebImg) {
        this.baseImage.set('');
      }
    } else if (kind === 'web') {
      this.layer.set('runtime');
      this.subCategory.set('web');
      if (!this.baseImage().trim()) {
        this.baseImage.set(dynamicWebImg);
      }
      this.installMode.set('npm');
      this.installPreset.set('npm');
      this.setMemoryProfile('standard');
    } else if (kind === 'addon') {
      this.layer.set('addon');
      this.subCategory.set('framework');
      this.setMemoryProfile('library');
      this.onCompatibleRuntimeChange(this.defaultCompatibleRuntime || this.compatibleRuntime() || '');
    }

    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(this.name(), this.compatibleRuntime()));
    }
  }

  private resolveAutoIconKey(nameVal: string, compatRuntimeKey: string): string {
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
      const sortedCurated = [...this.curatedIcons]
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

    if (this.layer() === 'addon' && compatRuntimeKey) {
      const lowerCompat = compatRuntimeKey.trim().toLowerCase();
      if (lowerCompat && lowerCompat !== 'universal') {
        const matchedRt = (this.availableRuntimes || []).find(r =>
          r.slug?.toLowerCase() === lowerCompat ||
          r.icon_key?.toLowerCase() === lowerCompat ||
          r.name?.toLowerCase() === lowerCompat
        );
        if (matchedRt?.icon_key && matchedRt.icon_key !== 'docker') {
          return matchedRt.icon_key;
        }
        const matchedCurated = this.curatedIcons.find(ic =>
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

  onCompatibleRuntimeChange(runtimeKey: string): void {
    this.compatibleRuntime.set(runtimeKey);
    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(this.name(), runtimeKey));
    }
    if (this.layer() === 'addon') {
      const key = runtimeKey.toLowerCase();
      const matched = (this.availableRuntimes || []).find(r =>
        r.slug?.toLowerCase() === key ||
        r.icon_key?.toLowerCase() === key ||
        r.name?.toLowerCase() === key
      );
      const isWeb = matched?.sub_category === 'web';

      if (isWeb) {
        this.installPreset.set('npm');
      } else if (matched?.install_mode && matched.install_mode !== 'none' && matched.install_mode !== 'custom') {
        this.installPreset.set(matched.install_mode);
      } else {
        const combined = (key + ' ' + (matched?.name || '') + ' ' + (matched?.icon_key || '')).toLowerCase();
        if (/node|angular|react|vue|javascript|typescript/.test(combined)) {
          this.installPreset.set('npm');
        } else if (/python/.test(combined)) {
          this.installPreset.set('pip');
        } else if (/rust/.test(combined)) {
          this.installPreset.set('cargo');
        } else if (/\bgo\b|golang/.test(combined)) {
          this.installPreset.set('go');
        } else if (/php/.test(combined)) {
          this.installPreset.set('composer');
        } else if (/cpp|\bc\b|c\+\+|gcc|linux|ubuntu|debian|java|dotnet/.test(combined)) {
          this.installPreset.set('apt');
        }
      }
    }
  }

  onSubCategoryChange(newSubCat: string): void {
    if (this.layer() === 'runtime') {
      this.onCatalogKindChange(newSubCat === 'web' ? 'web' : 'backend');
    } else {
      this.subCategory.set(newSubCat);
    }
  }

  onNameChange(val: string): void {
    this.name.set(val);
    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(val, this.compatibleRuntime()));
    }
    // Si está creando un runtime backend y escribe C, C++, Python, etc., auto-sugiere los valores técnicos si están vacíos
    if (!this.editingComponent && this.layer() === 'runtime' && this.subCategory() === 'backend') {
      const lower = (val || '').trim().toLowerCase();
      const matchedProfile = this.officialEcosystemProfiles.find(p => {
        if (p.id === 'cpp-gcc13' && (lower.includes('c++') || lower.includes('g++'))) return true;
        if (p.id === 'c-gcc13' && (/^c(\s|\(|$)/.test(lower) && !lower.includes('c++') && !lower.includes('c#'))) return true;
        return lower.startsWith(p.iconKey) && p.iconKey.length >= 2;
      });
      if (matchedProfile) {
        if (!this.baseImage().trim()) {
          this.baseImage.set(matchedProfile.baseImage);
        }
        if (!this.judgeEntrypoint().trim()) {
          this.judgeEntrypoint.set(matchedProfile.judgeEntrypoint);
        }
        if (!this.toolsProvidedInput().trim()) {
          this.toolsProvidedInput.set(matchedProfile.toolsProvided.join(', '));
        }
        this.setMemoryProfile(matchedProfile.memoryProfile);
      }
    }
  }

  selectCuratedIcon(key: string): void {
    this.isIconManuallySelected.set(true);
    this.iconKey.set(key);
    this.customIconUrl.set('');
    this.showCustomIconInput.set(false);
    this.showIconPickerGrid.set(false);

    // En modo creación, si el usuario hace clic en una tarjeta de tecnología oficial, autocompleta su perfil técnico completo
    if (!this.editingComponent) {
      const kind = this.catalogKind();
      if (kind === 'backend') {
        const matchedProfile = this.officialEcosystemProfiles.find(p => p.iconKey === key);
        if (matchedProfile) {
          this.onSelectEcosystemProfile(matchedProfile.id);
          this.isIconManuallySelected.set(true);
        }
      } else if (kind === 'web') {
        const matchedWeb = this.officialWebProfiles.find(p => p.iconKey === key);
        if (matchedWeb) {
          this.onSelectWebProfile(matchedWeb.id);
          this.isIconManuallySelected.set(true);
        }
      } else if (kind === 'addon') {
        const matchedAddon = this.contextualAddonPresets().find(p => p.iconKey === key && p.iconKey !== 'cpp' && p.iconKey !== 'python');
        if (matchedAddon) {
          this.onSelectAddonPreset(matchedAddon.id);
          this.isIconManuallySelected.set(true);
        }
      }
    }
  }

  toggleIconPickerGrid(): void {
    this.showIconPickerGrid.update(v => !v);
    if (this.showIconPickerGrid()) {
      this.showCustomIconInput.set(false);
    }
  }

  toggleCustomIconInput(): void {
    this.showCustomIconInput.update(v => !v);
    if (this.showCustomIconInput()) {
      this.showIconPickerGrid.set(false);
    }
  }

  toggleExpertMode(): void {
    this.showExpertMode.update(v => !v);
  }

  setMemoryProfile(profile: ComponentMemoryProfile): void {
    this.memoryProfile.set(profile);
    if (this.layer() === 'addon') {
      if (profile === 'devserver') {
        this.ramDeltaMB.set(256);
        this.subCategory.set('devserver');
      } else if (profile === 'heavy') {
        this.ramDeltaMB.set(384);
        this.subCategory.set('heavy');
      } else {
        this.ramDeltaMB.set(0);
        this.subCategory.set('framework');
      }
    } else if (this.layer() === 'runtime') {
      if (profile === 'light') {
        this.ramDeltaMB.set(384);
      } else if (profile === 'heavy') {
        this.ramDeltaMB.set(1024);
      } else {
        this.ramDeltaMB.set(512);
      }
    }
  }

  setLayerDefaults(newLayer: ComponentLayer): void {
    if (newLayer === 'runtime') {
      this.onCatalogKindChange('backend');
    } else if (newLayer === 'addon') {
      this.onCatalogKindChange('addon');
    } else {
      this.layer.set(newLayer);
      this.subCategory.set('relational');
      this.installMode.set('none');
      this.installPreset.set('none');
      this.ramDeltaMB.set(50);
    }
  }

  onClose(): void {
    this.closed.emit();
  }

  onSave(): void {
    const trimmedName = this.name().trim();
    if (trimmedName.length < 2) {
      this.errorMessage.set('El nombre de la tecnología debe tener al menos 2 caracteres.');
      return;
    }

    const currentLayer = this.layer();
    if (currentLayer === 'runtime' && !this.baseImage().trim()) {
      this.errorMessage.set('Debe seleccionar un perfil tecnológico o indicar la imagen base para el lenguaje principal.');
      return;
    }

    if (currentLayer === 'addon' && !this.compatibleRuntime().trim()) {
      this.errorMessage.set('Seleccione el lenguaje principal con el que es compatible esta tecnología.');
      return;
    }

    let preset = this.installPreset();
    let pkgs = this.packageNamesInput()
      .split(/[\s,]+/)
      .map(s => s.trim())
      .filter(Boolean);

    // Separación tolerante a espacios y comas (ej. "gdb valgrind cmake" o "gdb, valgrind, cmake")
    let tools = this.toolsProvidedInput()
      .split(/[\s,]+/)
      .map(s => s.trim())
      .filter(Boolean);

    // Si en un runtime el usuario dejó instalación en 'none' pero agregó paquetes en packageNamesInput, o escribió herramientas adicionales como gdb/valgrind/cmake en un runtime C/C++
    if (currentLayer === 'runtime' && preset === 'none' && pkgs.length > 0) {
      preset = this.resolvedEcosystemFamily() === 'python' ? 'pip' : (this.resolvedEcosystemFamily() === 'node' || this.resolvedEcosystemFamily() === 'web' ? 'npm' : 'apt');
    }

    const safePkgRegex = /^[a-zA-Z0-9@_./=<>-]+$/;
    for (const pkg of pkgs) {
      if (!safePkgRegex.test(pkg)) {
        this.errorMessage.set(`El nombre de paquete "${pkg}" contiene caracteres inválidos.`);
        return;
      }
    }

    if (preset !== 'custom' && preset !== 'none' && pkgs.length === 0) {
      this.errorMessage.set('Indique al menos un nombre de paquete para instalar.');
      return;
    }

    if (preset === 'custom' && !this.customBuildSnippet().trim()) {
      this.errorMessage.set('Indique la instrucción RUN personalizada.');
      return;
    }

    let resolvedInstallMode: InstallMode = 'none';
    let resolvedSnippet = '';

    if (preset !== 'none') {
      if (preset === 'pip') {
        resolvedInstallMode = 'pip';
      } else if (preset === 'uv') {
        resolvedInstallMode = 'uv';
        resolvedSnippet = `RUN uv pip install --system --no-cache ${pkgs.join(' ')}`;
      } else if (preset === 'npm') {
        resolvedInstallMode = 'npm';
      } else if (preset === 'pnpm') {
        resolvedInstallMode = 'pnpm';
        resolvedSnippet = `RUN corepack enable && pnpm add -g ${pkgs.join(' ')}`;
      } else if (preset === 'yarn') {
        resolvedInstallMode = 'yarn';
        resolvedSnippet = `RUN corepack enable && yarn global add ${pkgs.join(' ')}`;
      } else if (preset === 'bun') {
        resolvedInstallMode = 'bun';
        resolvedSnippet = `RUN bun add -g ${pkgs.join(' ')}`;
      } else if (preset === 'apt') {
        resolvedInstallMode = 'apt';
      } else if (preset === 'cargo') {
        resolvedInstallMode = 'cargo';
        resolvedSnippet = `RUN cargo install ${pkgs.join(' ')}`;
      } else if (preset === 'go') {
        resolvedInstallMode = 'go';
        resolvedSnippet = `RUN go install ${pkgs.join(' ')}`;
      } else if (preset === 'composer') {
        resolvedInstallMode = 'composer';
        resolvedSnippet = `RUN composer global require ${pkgs.join(' ')}`;
      } else if (preset === 'dotnet') {
        resolvedInstallMode = 'dotnet';
        resolvedSnippet = `RUN dotnet tool install -g ${pkgs.join(' ')}`;
      } else if (preset === 'custom') {
        resolvedInstallMode = 'custom';
        resolvedSnippet = this.customBuildSnippet().trim();
      }
    }

    // Sincronización automática: si se instalan paquetes del sistema (apt) y no se declararon binarios aparte, se verifican automáticamente
    if (resolvedInstallMode === 'apt' && tools.length === 0 && pkgs.length > 0) {
      tools = [...pkgs];
    }

    const generatedSlug = slugify(trimmedName);
    let compatibleRuntimes: string[] = [];

    if (currentLayer === 'database') {
      compatibleRuntimes = ['universal'];
    } else if (currentLayer === 'addon') {
      const selectedCompat = this.compatibleRuntime().trim();
      if (selectedCompat === 'c' || selectedCompat === 'cpp' || selectedCompat === 'c-gcc13' || selectedCompat === 'cpp-gcc13') {
        compatibleRuntimes = Array.from(new Set([selectedCompat, 'c', 'cpp']));
      } else {
        compatibleRuntimes = [selectedCompat];
      }
    } else if (currentLayer === 'runtime') {
      let familyKey: string[] = [];
      if (this.subCategory() === 'web') {
        familyKey = ['node', 'web'];
      } else {
        const rawIcon = this.iconKey().toLowerCase();
        if (rawIcon === 'c' || rawIcon === 'cpp') {
          familyKey = ['c', 'cpp'];
        } else if (rawIcon !== 'docker' && !rawIcon.includes('/') && !rawIcon.includes(':')) {
          familyKey = [rawIcon];
        }
      }
      const baseSlug = this.editingComponent?.slug || generatedSlug;
      compatibleRuntimes = Array.from(new Set([baseSlug, ...familyKey].filter(Boolean)));
    }

    const judgeEntrypointValue =
      currentLayer === 'runtime' && this.subCategory() === 'backend'
        ? this.judgeEntrypoint().trim()
        : '';

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    const targetExisting = this.editingComponent || this.existingCatalogMatch();
    if (targetExisting) {
      const updateDto: UpdateTemplateComponentDTO = {
        name: trimmedName,
        sub_category: this.subCategory(),
        description: this.description().trim(),
        icon_key: this.effectiveIconKey(),
        base_image: currentLayer === 'runtime' ? this.baseImage().trim() : '',
        compatible_runtimes: compatibleRuntimes,
        install_mode: resolvedInstallMode,
        package_names: pkgs,
        build_layer_snippet: resolvedSnippet,
        setup_snippet: targetExisting.setup_snippet,
        judge_entrypoint: judgeEntrypointValue,
        tools_provided: tools,
        service_engine: currentLayer === 'database' ? this.serviceEngine().trim() : '',
        ram_delta_mb: Number(this.ramDeltaMB()) || 0,
        is_active: true
      };

      this.templatesService.updateTechCatalogItem(targetExisting.id, updateDto).subscribe({
        next: (updated) => {
          this.isSubmitting.set(false);
          this.saved.emit(updated);
        },
        error: (err) => {
          this.isSubmitting.set(false);
          const msg = err.error?.error?.message || err.message || 'No se pudo actualizar la tecnología.';
          this.errorMessage.set(msg);
        }
      });
      return;
    }

    const createDto: CreateTemplateComponentDTO = {
      name: trimmedName,
      slug: generatedSlug,
      layer: currentLayer,
      sub_category: this.subCategory(),
      description: this.description().trim(),
      icon_key: this.effectiveIconKey(),
      base_image: currentLayer === 'runtime' ? this.baseImage().trim() : '',
      compatible_runtimes: compatibleRuntimes,
      install_mode: resolvedInstallMode,
      package_names: pkgs,
      build_layer_snippet: resolvedSnippet,
      judge_entrypoint: judgeEntrypointValue,
      tools_provided: tools,
      service_engine: currentLayer === 'database' ? this.serviceEngine().trim() : '',
      ram_delta_mb: Number(this.ramDeltaMB()) || 0
    };

    this.templatesService.createTechCatalogItem(createDto).subscribe({
      next: (created) => {
        this.isSubmitting.set(false);
        this.saved.emit(created);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error?.message || err.message || 'No se pudo registrar la tecnología.';
        this.errorMessage.set(msg);
      }
    });
  }
}
