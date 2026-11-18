import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado real del banco de laboratorios y del modal de duplicación.
 * La progresión del tutorial lee este contrato directamente desde Angular Signals.
 */
export interface TeacherExercisesTutorialContext {
  /** Indica si el docente tiene al menos una materia asignada */
  hasCourses: () => boolean;
  /** Cantidad total de materias asignadas al docente */
  coursesCount: () => number;
  /** Cantidad total de laboratorios creados en el banco */
  totalExercisesCount: () => number;
  /** Indica si el modal de duplicación está abierto */
  isDuplicateModalOpen: () => boolean;
  /** Indica si se seleccionó una materia destino válida en el modal */
  hasSelectedTargetCourse: () => boolean;
  /** Indica si el proceso de duplicación concluyó y el modal se cerró */
  isDuplicationFinished: () => boolean;
}

/**
 * Recorrido visual del Catálogo de Laboratorios del docente.
 * Señala métricas de cabecera, modalidades de ejecución, stack tecnológico, casos límite y duplicación.
 */
export function buildTeacherExercisesTour(): TourDefinition {
  return {
    id: 'teacher-exercises-tour-v1',
    title: 'Recorrido por el Catálogo de Laboratorios',
    steps: [
      {
        selector: '[data-tour="exercises-summary"]',
        title: 'Resumen y filtros rápidos',
        description:
          'La cabecera condensa las métricas del catálogo. Cada cifra actúa como filtro rápido para aislar publicados o borradores con un solo clic.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="exercises-modality-filter"]',
        title: 'Entorno de Evaluación vs Entorno de Desarrollo',
        description:
          'Filtrá según la necesidad: el Entorno de Evaluación ejecuta runners efímeros para calificar código con casos de prueba discretos, mientras que el Entorno de Desarrollo provee un espacio interactivo con terminal para proyectos continuos y APIs.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="exercises-filters"]',
        title: 'Búsqueda y materias',
        description:
          'Buscá por título, tecnología o lenguaje, o restringí la lista a una materia asignada para gestionar sus prácticas específicas.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="exercise-card"]',
        title: 'Ficha de práctica y stack',
        description:
          'Cada tarjeta detalla la materia a la que pertenece, la modalidad de ejecución, las tecnologías detectadas y el estado de publicación.',
        side: 'top'
      },
      {
        selector: '[data-tour="exercise-action-tests"]',
        title: 'Casos límite automáticos',
        description:
          'En Entornos de Evaluación, este botón analiza los contratos y genera casos de prueba para condiciones de borde (entradas vacías, números negativos, desbordes) sin tener que cargarlos a mano.',
        side: 'top'
      },
      {
        selector: '[data-tour="exercise-action-duplicate"]',
        title: 'Reutilizar en otra materia',
        description:
          '¿Querés usar este laboratorio en otra comisión o semestre? Pulsá «Duplicar» para crear una copia limpia como borrador, sin arrastrar notas ni entregas de alumnos previos.',
        side: 'top'
      }
    ]
  };
}

/**
 * Tutorial interactivo para reutilizar y duplicar un laboratorio entre materias.
 * Acompaña al docente desde la selección de la práctica base hasta la confirmación de la copia en la materia destino.
 */
export function buildTeacherExercisesTutorial(
  context: TeacherExercisesTutorialContext
): TutorialDefinition {
  return {
    id: 'teacher-exercises-duplicate-tutorial',
    title: 'Cómo reutilizar un laboratorio en otra materia',
    manualFragment: null,
    steps: [
      {
        id: 'select-source-lab',
        title: 'Elegir el laboratorio a reutilizar',
        body:
          'Identificá en el catálogo el laboratorio que querés llevar a otra de tus materias. Si aún no tenés laboratorios creados, primero creá tu primera práctica con «Nuevo Laboratorio».',
        selector: '[data-tour="exercise-card"]',
        side: 'top',
        check: {
          isComplete: () => context.totalExercisesCount() > 0,
          pendingHint: 'Necesitás tener al menos un laboratorio en el catálogo para poder duplicarlo.'
        }
      },
      {
        id: 'open-duplicate-dialog',
        title: 'Abrir el asistente de duplicación',
        body:
          'Hacé clic en el botón «Duplicar» en la tarjeta del ejercicio seleccionado para iniciar la copia limpia.',
        selector: '[data-tour="exercise-action-duplicate"]',
        side: 'top',
        check: {
          isComplete: () => context.isDuplicateModalOpen(),
          pendingHint: 'Hacé clic en «Duplicar» en la tarjeta del laboratorio.'
        }
      },
      {
        id: 'select-target-course',
        title: 'Seleccionar la materia destino',
        body:
          'Elegí en el selector a qué otra materia asignada querés copiar esta práctica. La materia original queda excluida para evitar duplicados en el mismo curso.',
        selector: '[data-tour="duplicate-course-select"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSelectedTargetCourse(),
          pendingHint: 'Seleccioná una materia destino en el menú desplegable.'
        }
      },
      {
        id: 'confirm-duplication',
        title: 'Confirmar la duplicación',
        body:
          'Pulsá «Duplicar» para generar la copia. Se creará como borrador independiente sin entregas de alumnos.',
        selector: '[data-tour="duplicate-confirm-btn"]',
        side: 'top',
        check: {
          isComplete: () => context.isDuplicationFinished(),
          pendingHint: 'Pulsá el botón «Duplicar» para confirmar la copia.'
        }
      },
      {
        id: 'duplication-completed',
        title: 'Laboratorio listo para personalizar',
        body:
          'El nuevo laboratorio ya está disponible como borrador en la materia elegida. Podés editar sus fechas, consignas o casos de prueba sin modificar el laboratorio original.',
        selector: '[data-tour="exercises-summary"]',
        side: 'bottom',
        check: {
          isComplete: () => true,
          pendingHint: ''
        }
      }
    ]
  };
}
