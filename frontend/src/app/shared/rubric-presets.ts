export interface RubricLevelPreset {
  name: string;
  score: number;
  description: string;
}

export interface RubricCriterionPreset {
  name: string;
  description: string;
  weight: number;
  levels: RubricLevelPreset[];
}

export interface RubricPreset {
  title: string;
  description: string;
  criteria: RubricCriterionPreset[];
}

export const RUBRIC_PRESETS: Record<string, RubricPreset> = {
  'api-rest': {
    title: 'API REST Backend',
    description: 'Evaluación de arquitectura de endpoints REST, manejo de errores, documentación y código limpio.',
    criteria: [
      {
        name: 'Endpoints REST',
        description: 'Implementación y respuestas correctas de verbos GET, POST, PUT, DELETE.',
        weight: 30,
        levels: [
          { name: 'Excelente', score: 100, description: 'Todos los endpoints funcionan con verbos HTTP y códigos de estado semánticos.' },
          { name: 'Bueno', score: 75, description: 'La mayoría de los endpoints funcionan correctamente; errores menores en verbos o status.' },
          { name: 'Regular', score: 50, description: 'Faltan endpoints o varios retornan errores 500 inesperados.' },
          { name: 'Deficiente', score: 25, description: 'La API no responde o la mayoría de endpoints están incompletos.' }
        ]
      },
      {
        name: 'Manejo de errores y validación',
        description: 'Validación de payload de entrada y mensajes de error descriptivos.',
        weight: 25,
        levels: [
          { name: 'Excelente', score: 100, description: 'Validación rigurosa de entradas y respuestas de error 4xx bien estructuradas.' },
          { name: 'Bueno', score: 75, description: 'Valida entradas básicas pero algunos bordes devuelven errores no formateados.' },
          { name: 'Regular', score: 50, description: 'Validaciones mínimas; permite cargas malformadas sin manejo.' },
          { name: 'Deficiente', score: 25, description: 'Sin validación de entrada ni manejo de excepciones.' }
        ]
      },
      {
        name: 'Código limpio y arquitectura',
        description: 'Separación de capas (rutas, controladores, servicios) y nomenclatura.',
        weight: 25,
        levels: [
          { name: 'Excelente', score: 100, description: 'Arquitectura limpia, funciones de responsabilidad única y nomenclatura clara.' },
          { name: 'Bueno', score: 75, description: 'Estructura organizada en capas con acoplamiento menor en algunos controladores.' },
          { name: 'Regular', score: 50, description: 'Lógica de negocio mezclada en las rutas o código duplicado.' },
          { name: 'Deficiente', score: 25, description: 'Código monolítico sin estructura ni separación de responsabilidades.' }
        ]
      },
      {
        name: 'Documentación y tests',
        description: 'Documentación del proyecto (README) y pruebas funcionales.',
        weight: 20,
        levels: [
          { name: 'Excelente', score: 100, description: 'README completo con ejemplos cURL/Postman y tests pasando.' },
          { name: 'Bueno', score: 75, description: 'README básico con instrucciones de ejecución claras.' },
          { name: 'Regular', score: 50, description: 'Documentación incompleta o de difícil configuración.' },
          { name: 'Deficiente', score: 25, description: 'Sin documentación ni instrucciones de ejecución.' }
        ]
      }
    ]
  },
  'full-stack': {
    title: 'Aplicación Web Full-Stack',
    description: 'Evaluación integral de Frontend (UI/UX), Backend (API), Integración y Base de Datos.',
    criteria: [
      {
        name: 'Interfaz de Usuario (Frontend)',
        description: 'Diseño responsive, usabilidad y manejo de estado.',
        weight: 30,
        levels: [
          { name: 'Excelente', score: 100, description: 'UI profesional, completamente responsive y manejo de estado fluido.' },
          { name: 'Bueno', score: 75, description: 'UI funcional y limpia con detalles menores de alineación o estado.' },
          { name: 'Regular', score: 50, description: 'Interfaz básica, sin diseño responsive o con fallos visuales.' },
          { name: 'Deficiente', score: 25, description: 'UI rota o inoperable.' }
        ]
      },
      {
        name: 'Integración Client-Server',
        description: 'Consumo de API HTTP, estados de carga y manejo de errores en cliente.',
        weight: 30,
        levels: [
          { name: 'Excelente', score: 100, description: 'Integración asíncrona perfecta con feedback de carga y alertas de error.' },
          { name: 'Bueno', score: 75, description: 'Comunicación fluida con el backend; falta feedback de error en algunos casos.' },
          { name: 'Regular', score: 50, description: 'Peticiones bloqueantes o congelamiento visual al comunicarse con API.' },
          { name: 'Deficiente', score: 25, description: 'Falla la integración entre cliente y servidor.' }
        ]
      },
      {
        name: 'Persistencia y Modelo de Datos',
        description: 'Diseño de base de datos, relaciones e integridad de datos.',
        weight: 20,
        levels: [
          { name: 'Excelente', score: 100, description: 'Base de datos normalizada, migraciones limpias e índices adecuados.' },
          { name: 'Bueno', score: 75, description: 'Esquema funcional con relaciones correctas.' },
          { name: 'Regular', score: 50, description: 'Redundancia de datos o falta de claves foráneas.' },
          { name: 'Deficiente', score: 25, description: 'Errores graves de persistenica o datos corruptos.' }
        ]
      },
      {
        name: 'Calidad y Estructura',
        description: 'Buenas prácticas, modularidad y git commits.',
        weight: 20,
        levels: [
          { name: 'Excelente', score: 100, description: 'Estructura impecable, componentes reutilizables y commits semánticos.' },
          { name: 'Bueno', score: 75, description: 'Código limpio con estructura clara.' },
          { name: 'Regular', score: 50, description: 'Archivos muy extensos con lógica mezclada.' },
          { name: 'Deficiente', score: 25, description: 'Código desordenado e ininteligible.' }
        ]
      }
    ]
  },
  'proyecto-integrador': {
    title: 'Proyecto Integrador / Capstone',
    description: 'Evaluación holística de alcance de requisitos, innovación, calidad técnica y presentación.',
    criteria: [
      {
        name: 'Cumplimiento de Requisitos',
        description: 'Cobertura del alcance y casos de uso requeridos.',
        weight: 35,
        levels: [
          { name: 'Excelente', score: 100, description: 'Cumple el 100% de las funcionalidades requeridas y características adicionales.' },
          { name: 'Bueno', score: 75, description: 'Cumple con todos los requisitos esenciales del proyecto.' },
          { name: 'Regular', score: 50, description: 'Faltan varias funcionalidades clave acordadas.' },
          { name: 'Deficiente', score: 25, description: 'Cumple menos del 50% de los objetivos planteados.' }
        ]
      },
      {
        name: 'Sólida Arquitectura Técnica',
        description: 'Patrones de diseño, seguridad y escalabilidad.',
        weight: 30,
        levels: [
          { name: 'Excelente', score: 100, description: 'Aplicación de patrones Clean/Hexagonal, variables de entorno y seguridad.' },
          { name: 'Bueno', score: 75, description: 'Arquitectura sólida con buen aislamiento de dependencias.' },
          { name: 'Regular', score: 50, description: 'Vulnerabilidades menores o acoplamiento directo.' },
          { name: 'Deficiente', score: 25, description: 'Sin patrones reconocibles, credenciales expuestas.' }
        ]
      },
      {
        name: 'Testing y Despliegue',
        description: 'Configuración Docker, CI/CD o suite de pruebas.',
        weight: 20,
        levels: [
          { name: 'Excelente', score: 100, description: 'Docker compose completamente funcional y tests automatizados.' },
          { name: 'Bueno', score: 75, description: 'Dockerización limpia y scripts de inicialización.' },
          { name: 'Regular', score: 50, description: 'Docker requiere ajustes manuales para ejecutar.' },
          { name: 'Deficiente', score: 25, description: 'Imposible de ejecutar mediante contenedor o script.' }
        ]
      },
      {
        name: 'Presentación y Documentación',
        description: 'Claridad en la memoria técnica y demostración.',
        weight: 15,
        levels: [
          { name: 'Excelente', score: 100, description: 'Documentación técnica impecable y diagrama de arquitectura.' },
          { name: 'Bueno', score: 75, description: 'Documentación clara y completa.' },
          { name: 'Regular', score: 50, description: 'Documentación escasa con vacíos técnicos.' },
          { name: 'Deficiente', score: 25, description: 'Sin documentación ni manual.' }
        ]
      }
    ]
  }
};
