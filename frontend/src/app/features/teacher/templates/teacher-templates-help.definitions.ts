import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado real del catálogo y del modal de solicitud.
 * La progresión del tutorial lee este contrato directamente desde Angular.
 */
export interface TeacherTemplatesTutorialContext {
  /** Indica si el modal de solicitud de entorno está abierto en pantalla */
  isRequestModalOpen: () => boolean;
  /** Valida que la nota sea opcional con piezas del catálogo o tenga >= 10 caracteres cuando aplica */
  descriptionValid: () => boolean;
  /** Indica si se seleccionó o extrajo al menos una herramienta en el entorno */
  hasChips: () => boolean;
  /** Indica si se eligió al menos una modalidad de uso (desarrollo o evaluación) */
  hasUses: () => boolean;
}

/**
 * Recorrido visual del Catálogo de Entornos del docente.
 * Señala pestañas, filtros, tarjetas, pedido de variantes y seguimiento de solicitudes.
 */
export function buildTeacherTemplatesTour(): TourDefinition {
  return {
    id: 'teacher-templates-tour-v1',
    title: 'Recorrido por el Catálogo de Entornos',
    steps: [
      {
        selector: '[data-tour="teacher-templates-tabs"]',
        title: 'Catálogo y Mis Solicitudes',
        description:
          'En «Catálogo» podés explorar todos los entornos aprobados. En «Mis solicitudes» hacés el seguimiento de los pedidos que enviaste a la administración.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-templates-filters"]',
        title: 'Búsqueda y filtros por modalidad',
        description:
          'Buscá por tecnología o materia, y filtrá por modalidad: Entorno de Desarrollo (con IDE y terminal para clase) o Entorno de Evaluación (calificación automática).',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-templates-header-actions"]',
        title: 'Solicitar un entorno nuevo',
        description:
          'Si necesitás una tecnología que no está en el catálogo, usá este botón para redactar tu práctica y pedir que el administrador prepare el entorno.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-templates-card"]',
        title: 'Ficha del entorno y sus herramientas',
        description:
          'Cada tarjeta muestra el propósito pedagógico, las tecnologías incluidas y si es un entorno oficial de la institución o una propuesta docente.',
        side: 'top'
      },
      {
        selector: '[data-tour="teacher-templates-variant-btn"]',
        title: 'Pedir una variante',
        description:
          '¿Te sirve esta base pero necesitás sumar una librería o base de datos? «Pedir una variante» abre la solicitud precargando estas herramientas para no empezar de cero.',
        side: 'top'
      },
      {
        selector: '[data-tour="teacher-templates-requests-tab"]',
        title: 'Seguimiento de pedidos',
        description:
          'El contador indica solicitudes en revisión. Al ser aprobada una solicitud, podés verla directamente en el catálogo mediante un clic.',
        side: 'bottom'
      }
    ]
  };
}

/**
 * Tutorial interactivo para solicitar un entorno o variante.
 * Guía al docente desde el catálogo hasta la preparación completa de la solicitud en el modal.
 */
export function buildTeacherTemplateRequestTutorial(
  context: TeacherTemplatesTutorialContext
): TutorialDefinition {
  return {
    id: 'teacher-template-request-tutorial',
    title: 'Cómo solicitar un entorno o variante',
    manualFragment: null,
    steps: [
      {
        id: 'open-request',
        title: 'El punto de partida',
        body:
          'Para pedir un entorno completamente nuevo pulse «Solicitar entorno». Si prefiere partir de una plantilla parecida, puede pulsar «Pedir una variante» en cualquier tarjeta.',
        selector: '[data-tour="teacher-templates-header-actions"]',
        side: 'bottom',
        check: {
          isComplete: () => context.isRequestModalOpen(),
          pendingHint: 'Abra el formulario pulsando «Solicitar entorno» o «Pedir una variante».'
        }
      },
      {
        id: 'tools-and-ai',
        title: 'Herramientas o asistencia con IA',
        body:
          'Pulsa «Agregar herramienta» para elegir lenguajes, frameworks o bases de datos del catálogo. Si no sabes qué elegir, describe tu clase y usa «Armar la lista con IA»; las piezas faltantes se marcan como «la pedimos».',
        selector: '[data-tour="template-request-tools"]',
        side: 'bottom',
        check: {
          isComplete: () => context.hasChips(),
          pendingHint: 'Agrega al menos una herramienta del catálogo o usa la asistencia con IA.'
        }
      },
      {
        id: 'select-uses',
        title: '¿Cómo lo van a usar?',
        body:
          'Elige si el entorno es para que los alumnos programen en clase (Entorno de Desarrollo), para entregas y pruebas automáticas (Entorno de Evaluación) o ambos.',
        selector: '[data-tour="template-request-uses"]',
        side: 'top',
        check: {
          isComplete: () => context.hasUses(),
          pendingHint: 'Seleccione al menos una opción de uso.'
        }
      },
      {
        id: 'describe-practice',
        title: 'Nombre, materia y nota',
        body:
          'Si ya elegiste herramientas del catálogo, la nota para administración es opcional. Si faltan piezas fuera del catálogo, explica en pocas palabras qué necesitan los estudiantes.',
        selector: '[data-tour="template-request-practice"]',
        side: 'top',
        check: {
          isComplete: () => context.descriptionValid(),
          pendingHint: 'Agrega herramientas del catálogo o escribe una nota de al menos 10 caracteres.'
        }
      },
      {
        id: 'review-summary',
        title: 'Revisión y envío',
        body:
          'Encima de los botones verás si ya existe un entorno igual o si otra materia pidió lo mismo. Pulsa «Enviar solicitud» cuando quieras registrarla, o «Cancelar» para salir.',
        selector: '[data-tour="template-request-actions"]',
        side: 'top'
      }
    ]
  };
}
