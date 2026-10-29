import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ComponentLayer } from '../../../../../services/admin-templates.service';

export type ComponentMemoryProfile = 'library' | 'devserver' | 'heavy' | 'light' | 'standard';
export type ComponentCatalogKind = 'backend' | 'web' | 'addon' | 'database';

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

export interface OfficialDatabaseProfile {
  id: string;
  name: string;
  nameTemplate: string;
  description: string;
  iconKey: string;
  serviceEngine: 'sqlite' | 'postgres' | 'mysql' | 'mariadb' | 'mongodb' | 'redis';
  subCategory: 'relational' | 'nosql';
  baseImage: string;
  imageTemplate: string;
  defaultVersion: string;
  suggestedVersions: EcosystemVersionOption[];
  ramDeltaMB: number;
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
  },
  {
    id: 'nextjs-web',
    name: 'Next.js (React Full-Stack)',
    nameTemplate: 'Next.js {version} (React Full-Stack)',
    description: 'Framework React full-stack con renderizado híbrido, server actions y soporte TypeScript',
    iconKey: 'react',
    baseImage: 'node:20-bookworm-slim',
    installPreset: 'npm',
    packageNames: ['create-next-app', 'typescript'],
    packageTemplate: ['create-next-app@{version}', 'typescript'],
    defaultVersion: '14',
    suggestedVersions: [
      { version: '15', label: 'Next.js 15 (Reciente)' },
      { version: '14', label: 'Next.js 14 LTS (Recomendada)', isDefault: true }
    ],
    toolsProvided: ['node', 'npm', 'npx'],
    memoryProfile: 'standard'
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
    name: 'Apache Maven (Proyectos Java / Spring)',
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
  },
  {
    id: 'python-sqlalchemy',
    name: 'SQLAlchemy (ORM)',
    description: 'Toolkit SQL y mapeador objeto-relacional (ORM) para Python con soporte asíncrono y migraciones Alembic',
    iconKey: 'sqlalchemy',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['sqlalchemy', 'alembic', 'psycopg2-binary', 'pymysql'],
    toolsProvided: ['python3', 'alembic'],
    memoryProfile: 'library'
  },
  {
    id: 'python-pymongo',
    name: 'PyMongo (Cliente MongoDB)',
    description: 'Controlador oficial de MongoDB para Python con soporte de documentos y BSON',
    iconKey: 'mongodb',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['pymongo'],
    toolsProvided: ['python3'],
    memoryProfile: 'library'
  },
  {
    id: 'python-redis',
    name: 'Redis-py (Cliente Redis)',
    description: 'Cliente oficial de Redis para Python con soporte de caché, pub/sub y estructuras en memoria',
    iconKey: 'redis',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['redis'],
    toolsProvided: ['python3'],
    memoryProfile: 'library'
  },
  {
    id: 'python-numpy',
    name: 'NumPy',
    description: 'Cálculo numérico multidimensional, álgebra lineal y operaciones matriciales de alto rendimiento',
    iconKey: 'numpy',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['numpy'],
    toolsProvided: ['python3'],
    memoryProfile: 'library'
  },
  {
    id: 'python-pandas',
    name: 'Pandas',
    description: 'Estructuras de datos DataFrame y herramientas de análisis y manipulación de datos tabulares',
    iconKey: 'pandas',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['pandas'],
    toolsProvided: ['python3'],
    memoryProfile: 'heavy'
  },
  {
    id: 'python-pytest',
    name: 'Pytest',
    description: 'Framework estándar de pruebas unitarias y automatizadas en Python',
    iconKey: 'pytest',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['pytest'],
    toolsProvided: ['pytest'],
    memoryProfile: 'library'
  },
  {
    id: 'python-ml',
    name: 'Machine Learning (Scikit-Learn + PyTorch)',
    description: 'Bibliotecas para aprendizaje automático, algoritmos predictivos y computación tensorial',
    iconKey: 'python',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['scikit-learn', 'torch', 'torchvision'],
    toolsProvided: ['python3'],
    memoryProfile: 'heavy'
  },
  {
    id: 'python-datascience',
    name: 'Ciencia de Datos (NumPy + Pandas)',
    description: 'Librerías esenciales para procesamiento numérico, tablas y análisis estadístico en Python',
    iconKey: 'python',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['numpy', 'pandas'],
    toolsProvided: ['python3'],
    memoryProfile: 'heavy'
  },
  {
    id: 'python-nosql-clients',
    name: 'Clientes NoSQL (MongoDB + Redis)',
    description: 'Controladores oficiales de conexión para bases de datos NoSQL Redis y MongoDB en Python',
    iconKey: 'python',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['redis', 'pymongo'],
    toolsProvided: ['python3'],
    memoryProfile: 'library'
  },
  {
    id: 'python-testing-ruff',
    name: 'Calidad y Pruebas (Pytest + Ruff)',
    description: 'Framework de pruebas unitarias pytest con linter y formateador ultrarrápido ruff',
    iconKey: 'python',
    compatibleFamilies: ['python'],
    installPreset: 'pip',
    packageNames: ['pytest', 'ruff'],
    toolsProvided: ['pytest', 'ruff'],
    memoryProfile: 'library'
  },
  {
    id: 'java-springboot',
    name: 'Spring Boot Framework',
    description: 'Framework empresarial en Java para microservicios y APIs REST con servidor web integrado',
    iconKey: 'spring',
    compatibleFamilies: ['java'],
    installPreset: 'apt',
    packageNames: ['default-jdk'],
    toolsProvided: ['java', 'javac'],
    memoryProfile: 'devserver'
  },
  {
    id: 'java-hibernate-jdbc',
    name: 'Hibernate ORM & Drivers JDBC',
    description: 'Mapeador objeto-relacional JPA/Hibernate con conectores PostgreSQL, MySQL y SQLite',
    iconKey: 'java',
    compatibleFamilies: ['java'],
    installPreset: 'apt',
    packageNames: ['libpostgresql-jdbc-java'],
    toolsProvided: ['java'],
    memoryProfile: 'library'
  },
  {
    id: 'java-gradle',
    name: 'Gradle Build Tool',
    description: 'Sistema de automatización de compilación y gestión de dependencias de alto rendimiento para Java y Kotlin',
    iconKey: 'java',
    compatibleFamilies: ['java'],
    installPreset: 'apt',
    packageNames: ['gradle'],
    toolsProvided: ['gradle'],
    memoryProfile: 'library'
  },
  {
    id: 'java-junit5',
    name: 'Pruebas Unitarias (JUnit 5 + AssertJ)',
    description: 'Ecosistema canónico de pruebas unitarias y aserciones para proyectos Java orientados a objetos',
    iconKey: 'java',
    compatibleFamilies: ['java'],
    installPreset: 'apt',
    packageNames: ['junit5'],
    toolsProvided: ['java'],
    memoryProfile: 'library'
  },
  {
    id: 'node-express',
    name: 'Express.js',
    description: 'Micro-framework rápido y minimalista para construcción de APIs REST y servicios web en Node.js',
    iconKey: 'node',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['express', 'cors'],
    toolsProvided: ['node', 'npm'],
    memoryProfile: 'library'
  },
  {
    id: 'node-nestjs',
    name: 'NestJS Framework',
    description: 'Framework progresivo TypeScript para desarrollo de arquitecturas empresariales y microservicios escalables',
    iconKey: 'node',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['@nestjs/cli'],
    toolsProvided: ['nest', 'node'],
    memoryProfile: 'library'
  },
  {
    id: 'node-prisma',
    name: 'Prisma ORM',
    description: 'ORM de tipado estático con generador de esquemas, migraciones declarativas y cliente de base de datos',
    iconKey: 'node',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['prisma', '@prisma/client'],
    toolsProvided: ['prisma', 'node'],
    memoryProfile: 'library'
  },
  {
    id: 'node-typeorm',
    name: 'TypeORM',
    description: 'ORM tradicional con soporte para patrones Data Mapper y Active Record en TypeScript y JavaScript',
    iconKey: 'node',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['typeorm', 'reflect-metadata'],
    toolsProvided: ['typeorm', 'node'],
    memoryProfile: 'library'
  },
  {
    id: 'node-vitest-jest',
    name: 'Testing (Vitest + Jest)',
    description: 'Entorno de pruebas unitarias y de integración de alta velocidad compatible con TypeScript',
    iconKey: 'node',
    compatibleFamilies: ['node', 'web'],
    installPreset: 'npm',
    packageNames: ['vitest', 'jest'],
    toolsProvided: ['vitest', 'node'],
    memoryProfile: 'library'
  },
  {
    id: 'cpp-sql-drivers',
    name: 'Drivers SQL (libpq + SQLite3 Dev)',
    description: 'Bibliotecas cliente nativas y cabeceras para conexión a PostgreSQL y SQLite en C y C++',
    iconKey: 'cpp',
    compatibleFamilies: ['c', 'cpp'],
    installPreset: 'apt',
    packageNames: ['libpq-dev', 'libsqlite3-dev'],
    toolsProvided: ['pkg-config'],
    memoryProfile: 'library'
  },
  {
    id: 'cpp-openmp',
    name: 'Concurrencia OpenMP & TBB',
    description: 'Bibliotecas y directivas de compilador para programación paralela, memoria compartida y multiprocesamiento',
    iconKey: 'cpp',
    compatibleFamilies: ['c', 'cpp'],
    installPreset: 'apt',
    packageNames: ['libomp-dev', 'libtbb-dev'],
    toolsProvided: ['gcc', 'g++'],
    memoryProfile: 'library'
  },
  {
    id: 'go-gin-chi',
    name: 'Gin & Chi Web Frameworks',
    description: 'Micro-frameworks web ultrarrápidos y enrutamiento HTTP modular para desarrollo de microservicios en Go',
    iconKey: 'go',
    compatibleFamilies: ['go'],
    installPreset: 'go',
    packageNames: ['github.com/gin-gonic/gin@latest', 'github.com/go-chi/chi/v5@latest'],
    toolsProvided: ['go'],
    memoryProfile: 'library'
  },
  {
    id: 'go-gorm',
    name: 'GORM & Drivers SQL',
    description: 'ORM completo para Go con soporte de transacciones, asociaciones y migraciones automáticas',
    iconKey: 'go',
    compatibleFamilies: ['go'],
    installPreset: 'go',
    packageNames: ['gorm.io/gorm@latest', 'gorm.io/driver/postgres@latest', 'gorm.io/driver/sqlite@latest'],
    toolsProvided: ['go'],
    memoryProfile: 'library'
  },
  {
    id: 'go-golangci-lint',
    name: 'Calidad y Linters (GolangCI-Lint)',
    description: 'Agregador oficial de linters de código estático y análisis de concurrencia para proyectos Go',
    iconKey: 'go',
    compatibleFamilies: ['go'],
    installPreset: 'apt',
    packageNames: ['golangci-lint'],
    toolsProvided: ['golangci-lint'],
    memoryProfile: 'library'
  },
  {
    id: 'rust-axum',
    name: 'Axum Web Framework',
    description: 'Framework web modular y ergonómico diseñado para el ecosistema asíncrono Tokio y Tower',
    iconKey: 'rust',
    compatibleFamilies: ['rust'],
    installPreset: 'cargo',
    packageNames: ['axum', 'tokio'],
    toolsProvided: ['cargo'],
    memoryProfile: 'library'
  },
  {
    id: 'rust-sqlx',
    name: 'SQLx Async ORM',
    description: 'Framework asíncrono de SQL puro para Rust con validación de consultas en tiempo de compilación',
    iconKey: 'rust',
    compatibleFamilies: ['rust'],
    installPreset: 'cargo',
    packageNames: ['sqlx-cli'],
    toolsProvided: ['cargo', 'sqlx'],
    memoryProfile: 'library'
  },
  {
    id: 'rust-clippy',
    name: 'Herramientas de Calidad (Clippy + Rustfmt)',
    description: 'Herramientas oficiales de buenas prácticas, linter de rendimiento y formateo canónico de Rust',
    iconKey: 'rust',
    compatibleFamilies: ['rust'],
    installPreset: 'cargo',
    packageNames: ['clippy'],
    toolsProvided: ['cargo-clippy', 'rustfmt'],
    memoryProfile: 'library'
  },
  {
    id: 'dotnet-aspnet',
    name: 'ASP.NET Core Web APIs',
    description: 'Framework de alto rendimiento para creación de APIs REST y servicios web modernos con C# y .NET',
    iconKey: 'dotnet',
    compatibleFamilies: ['dotnet'],
    installPreset: 'dotnet',
    packageNames: ['Microsoft.AspNetCore.App'],
    toolsProvided: ['dotnet'],
    memoryProfile: 'library'
  },
  {
    id: 'dotnet-ef',
    name: 'Entity Framework Core (EF Core)',
    description: 'Herramienta oficial de migración y mapeo objeto-relacional para aplicaciones de datos en .NET',
    iconKey: 'dotnet',
    compatibleFamilies: ['dotnet'],
    installPreset: 'dotnet',
    packageNames: ['dotnet-ef'],
    toolsProvided: ['dotnet-ef', 'dotnet'],
    memoryProfile: 'library'
  },
  {
    id: 'dotnet-xunit',
    name: 'Pruebas Unitarias (xUnit + Moq)',
    description: 'Framework de pruebas unitarias y biblioteca de simulación de dependencias (mocking) para C#',
    iconKey: 'dotnet',
    compatibleFamilies: ['dotnet'],
    installPreset: 'dotnet',
    packageNames: ['xunit', 'moq'],
    toolsProvided: ['dotnet'],
    memoryProfile: 'library'
  }
];

// Motores de base de datos gestionados por el catálogo del nodo. Los contenedores
// satélite (mysql, mariadb, mongodb, redis) se auto-orquestan bajo demanda y se
// apagan solos cuando ningún laboratorio activo los usa; PostgreSQL se comparte
// en el host del aula y SQLite vive dentro del contenedor del alumno.
export const OFFICIAL_DATABASE_PROFILES: OfficialDatabaseProfile[] = [
  {
    id: 'mysql-84',
    name: 'MySQL 8.4',
    nameTemplate: 'MySQL {version}',
    description: 'Motor relacional auto-orquestado y aislado por estudiante',
    iconKey: 'mysql',
    serviceEngine: 'mysql',
    subCategory: 'relational',
    baseImage: 'mysql:8.4',
    imageTemplate: 'mysql:{version}',
    defaultVersion: '8.4',
    suggestedVersions: [
      { version: '8.4', label: 'MySQL 8.4 LTS (Recomendada)', isDefault: true },
      { version: '8.0', label: 'MySQL 8.0 (Clásica)' }
    ],
    ramDeltaMB: 128
  },
  {
    id: 'mariadb-11',
    name: 'MariaDB 11.4',
    nameTemplate: 'MariaDB {version}',
    description: 'Motor relacional auto-orquestado compatible con MySQL',
    iconKey: 'mysql',
    serviceEngine: 'mariadb',
    subCategory: 'relational',
    baseImage: 'mariadb:11.4',
    imageTemplate: 'mariadb:{version}',
    defaultVersion: '11.4',
    suggestedVersions: [
      { version: '11.4', label: 'MariaDB 11.4 LTS (Recomendada)', isDefault: true },
      { version: '10.11', label: 'MariaDB 10.11 LTS' }
    ],
    ramDeltaMB: 128
  },
  {
    id: 'mongodb-70',
    name: 'MongoDB 7.0',
    nameTemplate: 'MongoDB {version}',
    description: 'Base de datos documental auto-orquestada para ejercicios NoSQL',
    iconKey: 'mongodb',
    serviceEngine: 'mongodb',
    subCategory: 'nosql',
    baseImage: 'mongo:7.0',
    imageTemplate: 'mongo:{version}',
    defaultVersion: '7.0',
    suggestedVersions: [
      { version: '7.0', label: 'MongoDB 7.0 (Recomendada)', isDefault: true },
      { version: '6.0', label: 'MongoDB 6.0' }
    ],
    ramDeltaMB: 128
  },
  {
    id: 'redis-72',
    name: 'Redis 7.2 (Caché)',
    nameTemplate: 'Redis {version} (Caché)',
    description: 'Almacén en memoria auto-orquestado para caché y colas',
    iconKey: 'redis',
    serviceEngine: 'redis',
    subCategory: 'nosql',
    baseImage: 'redis:7.2-alpine',
    imageTemplate: 'redis:{version}-alpine',
    defaultVersion: '7.2',
    suggestedVersions: [
      { version: '7.2', label: 'Redis 7.2 (Recomendada)', isDefault: true },
      { version: '7.4', label: 'Redis 7.4 (Reciente)' }
    ],
    ramDeltaMB: 64
  },
  {
    id: 'postgres-16',
    name: 'PostgreSQL 18 (Aula)',
    nameTemplate: 'PostgreSQL {version}',
    description: 'Motor relacional compartido del aula, sintonizado para laboratorio',
    iconKey: 'postgres',
    serviceEngine: 'postgres',
    subCategory: 'relational',
    baseImage: 'postgres:18-alpine',
    imageTemplate: 'postgres:{version}-alpine',
    defaultVersion: '18',
    suggestedVersions: [
      { version: '18', label: 'PostgreSQL 18 (Recomendada)', isDefault: true },
      { version: '16', label: 'PostgreSQL 16' }
    ],
    ramDeltaMB: 50
  },
  {
    id: 'sqlite-3',
    name: 'SQLite 3 (En archivo)',
    nameTemplate: 'SQLite {version} (En archivo)',
    description: 'Base de datos embebida en archivo dentro del contenedor del alumno',
    iconKey: 'sqlite',
    serviceEngine: 'sqlite',
    subCategory: 'relational',
    baseImage: '',
    imageTemplate: '',
    defaultVersion: '3',
    suggestedVersions: [],
    ramDeltaMB: 0
  }
];

export const CATALOG_KIND_OPTIONS: ComboboxOption<ComponentCatalogKind>[] = [
  { id: 'backend', label: 'Columna 1 · Lenguaje Principal (C, C++, Python, Java, Go, Rust...)', value: 'backend' },
  { id: 'web', label: 'Columna 1 · Interfaz Web / Frontend (Angular, React, Vue...)', value: 'web' },
  { id: 'addon', label: 'Columna 2 · Complemento, Framework o Librería', value: 'addon' },
  { id: 'database', label: 'Columna 3 · Motor de Base de Datos o Caché', value: 'database' }
];

export const LAYER_OPTIONS: ComboboxOption<ComponentLayer>[] = [
  { id: 'runtime', label: '1. Tecnología Principal o Interfaz Web', value: 'runtime' },
  { id: 'addon', label: '2. Complemento / Framework / Librería', value: 'addon' },
  { id: 'database', label: '3. Motor de Base de Datos o Caché', value: 'database' }
];

export const SUB_CATEGORY_OPTIONS: ComboboxOption<string>[] = [
  { id: 'backend', label: 'Lenguaje Principal (Backend / General)', value: 'backend' },
  { id: 'web', label: 'Plataforma Web / Frontend', value: 'web' }
];

export const ADDON_SUB_CATEGORY_OPTIONS: ComboboxOption<string>[] = [
  { id: 'framework', label: 'Framework Principal (máx. 1)', value: 'framework' },
  { id: 'orm', label: 'ORM / Conector de Datos', value: 'orm' },
  { id: 'toolkit', label: 'Kit / Herramienta Académica', value: 'toolkit' }
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
  { id: 'postgres', label: 'PostgreSQL (Aula compartida)', value: 'postgres' },
  { id: 'mysql', label: 'MySQL (Auto-orquestado)', value: 'mysql' },
  { id: 'mariadb', label: 'MariaDB (Auto-orquestado)', value: 'mariadb' },
  { id: 'sqlite', label: 'SQLite (Archivo local)', value: 'sqlite' },
  { id: 'mongodb', label: 'MongoDB (Auto-orquestado)', value: 'mongodb' },
  { id: 'redis', label: 'Redis (Caché auto-orquestada)', value: 'redis' },
  { id: 'none', label: 'Ninguno (solo archivos del alumno)', value: 'none' }
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
