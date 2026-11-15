import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías de Cursos y Períodos Académicos.
 *
 * El tutorial acompaña el alta hasta el botón de confirmación y termina sin
 * crear la materia: sirve para capacitación sin dejar datos de prueba. La
 * reasignación y el archivado solo se explican.
 */

export interface CoursesTutorialContext {
  /** El modal de alta está abierto. */
  createModalOpen: () => boolean;
  /** Nombre y código escritos en el formulario. */
  identityComplete: () => boolean;
}

const MANUAL_FRAGMENT = '4-2-cursos-y-asignacion-de-docentes';

/**
 * Recorrido de la pantalla: resumen, selector de período, estados, tabla y
 * acciones por fila.
 */
export function buildCoursesTour(): TourDefinition {
  return {
    id: 'courses-tour-v1',
    title: 'Recorrido por Cursos y Períodos',
    steps: [
      {
        selector: '[data-tour="courses-summary"]',
        title: 'El resumen del período',
        description:
          'La línea bajo el título cuenta las materias del período visible, cuántas están activas, cuántas no tienen docente y cuántas quedaron archivadas.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="courses-period"]',
        title: 'El período decide qué se ve',
        description:
          'El selector acota la tabla y los contadores a un semestre. «Todos los períodos» muestra también las materias que no tienen período asignado.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="courses-tabs"]',
        title: 'Cuatro estados, un mismo catálogo',
        description:
          '«Sin Docente» es la bandeja que necesita atención: materias activas sin titular. «Archivados» muestra las de períodos cerrados, que se consultan pero no abren sesiones nuevas.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="courses-table"]',
        title: 'Qué dice cada fila',
        description:
          'Código y nombre, docente titular, plantilla base, estudiantes matriculados y estado. Una fila atenuada está archivada.',
        side: 'top'
      },
      {
        selector: '[data-tour="courses-actions"]',
        title: 'Reasignar, editar y archivar',
        description:
          'Reasignar cambia el titular sin perder el contenido. Editar corrige nombre y código. Archivar deja la materia en solo lectura y pide confirmación; desarchivar la devuelve al catálogo.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial del alta: exige el modal abierto y nombre y código escritos, y
 * termina antes de confirmar. Docente y período son opcionales y se explican
 * sin bloquear.
 */
export function buildCoursesTutorial(context: CoursesTutorialContext): TutorialDefinition {
  return {
    id: 'courses-create-course',
    title: 'Registrar una materia',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'summary',
        title: 'Dónde se ve el estado del catálogo',
        body:
          'Antes de crear, el resumen y las pestañas dicen cuántas materias hay en el período y cuántas no tienen docente. Una materia nueva aparece en la tabla al confirmar el alta.',
        selector: '[data-tour="courses-summary"]',
        side: 'bottom'
      },
      {
        id: 'open',
        title: 'Abra el alta',
        body:
          '«Nuevo Curso» abre el formulario de materia oficial. Desde aquí se define el código que verán docentes y estudiantes en todo el sistema.',
        selector: '[data-tour="courses-create"]',
        side: 'bottom',
        check: {
          isComplete: () => context.createModalOpen(),
          pendingHint: 'Pulse «Nuevo Curso» para continuar.'
        }
      },
      {
        id: 'identity',
        title: 'Nombre, código y sección',
        body:
          'El nombre y el código son obligatorios. La sección es opcional y se agrega al código: «SIS-311» con sección «1» se identifica como SIS-311-1. El formulario muestra el identificador final mientras escribe.',
        selector: '[data-tour="course-create-identity"]',
        side: 'bottom',
        check: {
          isComplete: () => context.createModalOpen() && context.identityComplete(),
          pendingHint: 'Complete el nombre y el código de la materia para continuar.'
        }
      },
      {
        id: 'period',
        title: 'El período académico',
        body:
          'El período agrupa la materia en su semestre. Si se deja sin período, la materia existe pero no aparece al filtrar por un semestre concreto. Se puede corregir después desde la edición.',
        selector: '[data-tour="course-create-period"]',
        side: 'top'
      },
      {
        id: 'teacher',
        title: 'El docente puede quedar pendiente',
        body:
          'Asignar el titular ahora evita que la materia caiga en «Sin Docente». Si todavía no se decidió, se deja sin asignar y se reasigna desde la fila cuando corresponda.',
        selector: '[data-tour="course-create-teacher"]',
        side: 'top'
      },
      {
        id: 'submit',
        title: 'El paso que cierra el alta',
        body:
          '«Crear Curso» registra la materia en el período elegido y la muestra en la tabla con su estado real; un código duplicado se rechaza con un mensaje del servidor. Esta guía no confirma el alta: revise el resumen y cierre el formulario si solo estaba practicando.',
        selector: '[data-tour="course-create-submit"]',
        side: 'top'
      },
      {
        id: 'lifecycle',
        title: 'Qué queda después del alta',
        body:
          'Desde la fila se reasigna el titular, se editan nombre y código y se archiva la materia. Archivar pide confirmación y deja todo en solo lectura; aquí termina la guía y esas decisiones quedan en sus manos.',
        selector: '[data-tour="courses-actions"]',
        side: 'left'
      }
    ]
  };
}
