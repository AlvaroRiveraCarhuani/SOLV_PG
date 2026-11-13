import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías del Constructor de Plantillas.
 *
 * Los selectores apuntan a elementos reales de la pantalla. Las comprobaciones
 * del tutorial leen el estado del constructor: nunca se marca un paso como
 * cumplido por el simple hecho de pulsar «Siguiente».
 */

const MANUAL_FRAGMENT = '7-constructor-de-plantillas-de-entornos';

export function buildStudioTour(): TourDefinition {
  return {
    id: 'studio-tour-v1',
    title: 'Recorrido por el Constructor de Plantillas',
    steps: [
      {
        selector: '[data-tour="studio-purpose"]',
        title: 'Qué es una plantilla oficial',
        description:
          'Una plantilla define el entorno que los docentes podrán elegir al crear un ejercicio: modalidad, tecnologías y memoria. Aquí se registra una sola vez y luego se reutiliza en todas las materias.',
        side: 'bottom'
      },
      {
        selector: '#studio-tpl-name',
        title: 'Identidad para los docentes',
        description:
          'El nombre es lo que verá el docente en el catálogo y en el asistente. La categoría agrupa la plantilla y la descripción explica para qué sirve.',
        side: 'right'
      },
      {
        selector: '[data-tour="studio-stack"]',
        title: 'Tecnología del entorno',
        description:
          'Elija el lenguaje principal y, si corresponde, la interfaz web, los complementos y la base de datos. El Modo Avanzado permite revisar el Dockerfile y los scripts generados.',
        side: 'right'
      },
      {
        selector: '[data-tour="studio-capacity"]',
        title: 'Capacidad del aula',
        description:
          'La memoria por alumno determina cuántos estudiantes pueden trabajar a la vez. El panel muestra el mínimo que exigen las tecnologías y el máximo que permite el servidor.',
        side: 'top'
      },
      {
        selector: '[data-tour="studio-summary"]',
        title: 'Revise antes de guardar',
        description:
          'El resumen confirma nombre, imagen, tecnologías, memoria y programas incluidos. La pestaña Manifiesto muestra el contrato técnico que se guardará con la plantilla.',
        side: 'left'
      },
      {
        selector: '[data-tour="studio-publish"]',
        title: 'Comprobar y publicar',
        description:
          '«Comprobar funcionamiento» valida que las tecnologías respondan. Publicar registra la plantilla oficial: los docentes podrán usarla en sus materias. Si algo falta, el panel indica el motivo exacto.',
        side: 'left'
      }
    ]
  };
}

export interface StudioTutorialContext {
  /** Nombre actual de la plantilla en edición. */
  templateName: () => string;
  /** Indica si hay al menos una tecnología seleccionada. */
  hasSelectedStack: () => boolean;
  /** Indica si la memoria elegida respeta el mínimo y el máximo del servidor. */
  isCapacityValid: () => boolean;
  /** Motivo por el que la publicación está bloqueada, si existe. */
  publishBlocker: () => { reason: string } | null;
}

export function buildStudioTutorial(context: StudioTutorialContext): TutorialDefinition {
  const nameReady = () => context.templateName().trim().length >= 3;
  const stackReady = () => context.hasSelectedStack();

  return {
    id: 'studio-first-template',
    title: 'Crear y publicar una plantilla',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'purpose',
        title: 'Para qué sirve esta plantilla',
        body:
          'Una plantilla oficial es el entorno que el docente elige al preparar un ejercicio. Define si el estudiante trabaja con un editor o si el código se corrige automáticamente, y qué tecnologías tendrá disponibles.',
        selector: '[data-tour="studio-purpose"]',
        side: 'bottom'
      },
      {
        id: 'identity',
        title: 'Complete la identidad',
        body:
          'Elija la modalidad y escriba un nombre reconocible para el docente. La categoría agrupa la plantilla en el catálogo; la descripción es opcional y explica el uso previsto.',
        selector: '#studio-tpl-name',
        side: 'right',
        check: {
          isComplete: nameReady,
          pendingHint: 'Escriba un nombre de al menos 3 caracteres para continuar.'
        }
      },
      {
        id: 'stack',
        title: 'Seleccione la tecnología',
        body:
          'Elija el lenguaje principal del entorno. Puede añadir una interfaz web, complementos y una base de datos si la materia lo necesita. Sólo se ofrecen tecnologías del catálogo homologado.',
        selector: '[data-tour="studio-stack"]',
        side: 'right',
        check: {
          isComplete: stackReady,
          pendingHint: 'Seleccione al menos el lenguaje principal para continuar.'
        }
      },
      {
        id: 'capacity',
        title: 'Interprete las recomendaciones de recursos',
        body:
          'Cada opción muestra la memoria por alumno y cuántos estudiantes caben a la vez. El mínimo requerido lo fijan las tecnologías elegidas; el máximo, el servidor. Si la memoria supera la libre actual, el panel lo advierte sin bloquear la plantilla.',
        selector: '[data-tour="studio-capacity"]',
        side: 'top',
        check: {
          isComplete: () => stackReady() && context.isCapacityValid(),
          pendingHint: 'Ajuste la memoria dentro de los límites permitidos para continuar.'
        }
      },
      {
        id: 'summary',
        title: 'Revise la configuración',
        body:
          'El resumen del entorno reúne nombre, imagen, tecnologías, memoria y programas incluidos. Revise que coincida con lo que necesita la materia antes de continuar.',
        selector: '[data-tour="studio-summary"]',
        side: 'left',
        check: {
          isComplete: () => nameReady() && stackReady(),
          pendingHint: 'Complete el nombre y las tecnologías para que el resumen esté completo.'
        }
      },
      {
        id: 'publish',
        title: 'Comprobar y publicar',
        body:
          '«Comprobar funcionamiento» construye la imagen cuando hace falta y verifica que los programas respondan. Publicar registra la plantilla oficial para que los docentes la usen en sus materias; si algo falta, el panel indica el motivo y no permite publicar.',
        selector: '[data-tour="studio-publish"]',
        side: 'left'
      }
    ]
  };
}
