import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado real de la cola de revisiones docentes.
 * La progresión del tutorial lee directamente este contrato desde Angular Signals.
 */
export interface TeacherReviewsTutorialContext {
  /** Indica si existen prácticas con entregas pendientes en la cola */
  hasGroups: () => boolean;
  /** Indica si el docente tiene una práctica seleccionada en el panel izquierdo */
  hasSelectedGroup: () => boolean;
  /** Cantidad de entregas que esperan corrección en la práctica seleccionada */
  submissionsCount: () => number;
  /** Indica si la cola está filtrada por una materia en particular */
  isFilterActive: () => boolean;
}

/**
 * Recorrido visual de la Cola «Por calificar».
 * Explica la cabecera, el agrupamiento por prácticas, los tiempos de espera y el acceso a SpeedGrader.
 */
export function buildTeacherReviewsTour(): TourDefinition {
  return {
    id: 'teacher-reviews-tour-v1',
    title: 'Recorrido por la Cola de Entregas',
    steps: [
      {
        selector: '[data-tour="reviews-header-summary"]',
        title: 'Bandeja de Corrección Docente',
        description:
          'Aquí se centralizan todas las entregas que esperan tu evaluación en tus materias, organizadas para corregir por lotes sin buscar materia por materia.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="reviews-queue-pane"]',
        title: 'Prácticas Agrupadas',
        description:
          'Cada tarjeta representa una práctica con entregas pendientes. Indica la cantidad por revisar y la fecha de la entrega más antigua para no postergar a ningún alumno.',
        side: 'right'
      },
      {
        selector: '[data-tour="reviews-work-header"]',
        title: 'Detalle de la Práctica Seleccionada',
        description:
          'Muestra el nombre de la materia y de la práctica activa. El enlace «Abrir práctica» te permite consultar la consigna o configuración si tenés dudas.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="reviews-table"]',
        title: 'Entregas por Orden de Llegada',
        description:
          'Las entregas se ordenan de la más antigua a la más reciente. La columna «Espera» calcula el tiempo transcurrido desde el envío del estudiante.',
        side: 'top'
      },
      {
        selector: '[data-tour="reviews-grade-action"]',
        title: 'Calificar con SpeedGrader',
        description:
          'El botón «Calificar» abre el visor de revisión de esa entrega con su rúbrica y código. Al finalizar la nota, volvés directamente al siguiente alumno de la cola.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial interactivo guiado para gestionar y despachar correcciones pendientes.
 */
export function buildTeacherReviewsTutorial(
  context: TeacherReviewsTutorialContext
): TutorialDefinition {
  return {
    id: 'teacher-reviews-tutorial',
    title: 'Cómo despachar entregas en espera',
    manualFragment: null,
    steps: [
      {
        id: 'select-practice',
        title: '1. Seleccionar una práctica',
        body:
          'Elegí una práctica de la columna izquierda para desplegar sus entregas pendientes. Corregir una misma práctica en bloque te permite mantener el mismo criterio fresco.',
        selector: '[data-tour="reviews-queue-pane"]',
        side: 'right',
        check: {
          isComplete: () => context.hasSelectedGroup(),
          pendingHint: 'Hacé clic en una de las prácticas de la lista izquierda.'
        }
      },
      {
        id: 'check-wait-time',
        title: '2. Priorizar por tiempo de espera',
        body:
          'Observá la columna «Espera» del primer alumno de la lista. Las entregas se presentan en orden cronológico estricto para dar pronta respuesta a quienes entregaron antes.',
        selector: '[data-tour="reviews-table"]',
        side: 'top',
        check: {
          isComplete: () => context.submissionsCount() > 0,
          pendingHint: 'Asegurate de que la práctica seleccionada tenga entregas visibles.'
        }
      },
      {
        id: 'check-review-status',
        title: '3. Identificar estado de revisión',
        body:
          'Si una entrega muestra la etiqueta «En revisión», significa que vos o un ayudante de cátedra ya abrieron la entrega. Esto evita correcciones duplicadas en el equipo.',
        selector: '[data-tour="reviews-table"]',
        side: 'top',
        check: {
          isComplete: () => context.submissionsCount() > 0,
          pendingHint: 'Verificá el estado de los alumnos en la tabla de entregas.'
        }
      },
      {
        id: 'start-grading',
        title: '4. Lanzar la calificación',
        body:
          'Pulsá «Calificar» en la fila del alumno para ingresar a SpeedGrader. Allí podrás ejecutar el código, aplicar la rúbrica y redactar la devolución pedagógica.',
        selector: '[data-tour="reviews-grade-action"]',
        side: 'left',
        check: {
          isComplete: () => context.hasSelectedGroup() && context.submissionsCount() > 0,
          pendingHint: 'Localizá el botón «Calificar» de la primera entrega.'
        }
      }
    ]
  };
}
