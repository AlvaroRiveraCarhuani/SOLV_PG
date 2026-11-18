import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado y acciones del docente en la ficha de práctica.
 */
export interface PracticeDetailTutorialContext {
  /** Indica si la modalidad es IDE */
  isIde: () => boolean;
  /** Indica si la sección de consigna está desplegada */
  isConsignaExpanded: () => boolean;
  /** Indica si la sección de criterios / rúbrica está desplegada */
  isEvalExpanded: () => boolean;
  /** Indica si la práctica cuenta con bases de datos satélite */
  hasDatabase: () => boolean;
  /** Indica si la sección de bases de datos está desplegada */
  isDatabasesExpanded: () => boolean;
}

/**
 * Construye el recorrido visual guiado (tour) para la ficha de detalle de práctica.
 */
export function buildPracticeDetailTour(isIde: boolean, hasDatabase: boolean): TourDefinition {
  const steps = [
    {
      selector: '[data-tour="practice-header-actions"]',
      title: 'Acciones y Espacio de Trabajo',
      description: isIde
        ? 'Abrí tu propio espacio de trabajo en el IDE para explorar el proyecto, realizá una prueba rápida previa a clase o llevá la actividad al aula en vivo.'
        : 'Probá la solución del ejercicio, ajustá casos de prueba o llevalo al aula para trabajar con tus alumnos.',
      side: 'bottom' as const
    },
    {
      selector: '[data-tour="practice-env-status"]',
      title: 'Validación Previa al Aula',
      description:
        'Verificá el estado del entorno base antes de la clase. El semáforo te confirma que el repositorio y sus servicios satélites están listos para los estudiantes.',
      side: 'bottom' as const
    },
    {
      selector: '[data-tour="practice-params-grid"]',
      title: 'Parámetros y Políticas de la Práctica',
      description:
        'Consultá la plantilla asignada, las tecnologías detectadas, la política de re-entregas y los límites de ejecución configurados.',
      side: 'bottom' as const
    },
    {
      selector: '[data-tour="practice-consigna"]',
      title: 'Consigna y Estructura del Proyecto',
      description:
        'Leé las instrucciones y examiná la estructura esperada de carpetas con íconos visuales de directorios y archivos. Hacé clic en la cabecera para colapsar o desplegar la sección.',
      side: 'top' as const
    },
    {
      selector: '[data-tour="practice-eval-criteria"]',
      title: isIde ? 'Rúbrica de Calificación' : 'Casos de Prueba',
      description: isIde
        ? 'Revisá los criterios de evaluación ponderados y sus porcentajes de puntaje para corregir las entregas con transparencia.'
        : 'Inspeccioná las entradas (STDIN) y salidas esperadas (STDOUT) de cada caso de prueba con su respectivo puntaje.',
      side: 'top' as const
    }
  ];

  if (isIde && hasDatabase) {
    steps.push({
      selector: '[data-tour="practice-databases"]',
      title: 'Bases de Datos de los Alumnos',
      description:
        'Monitoreá las bases de datos asignadas a cada estudiante. Si un alumno rompe sus datos o esquemas, podés reiniciarla con un solo clic al estado inicial.',
      side: 'top' as const
    });
  }

  return {
    id: 'practice-detail-tour',
    title: 'Ficha de Detalle de Práctica',
    steps
  };
}

/**
 * Construye el tutorial interactivo reactivo de la ficha de práctica.
 */
export function buildPracticeDetailTutorial(
  context: PracticeDetailTutorialContext
): TutorialDefinition {
  const steps = [
    {
      id: 'toggle-consigna',
      title: '1. Desplegar y colapsar secciones',
      body:
        'Hacé clic en el encabezado de la Consigna de la Práctica para alternar entre vista completa y modo compacto.',
      selector: '[data-tour="practice-consigna"] .section-toggle-head',
      check: {
        isComplete: () => !context.isConsignaExpanded(),
        pendingHint: 'Hacé clic en el encabezado de la consigna para colapsarla.'
      }
    },
    {
      id: 'inspect-criteria',
      title: '2. Examinar los criterios de evaluación',
      body:
        'Hacé clic en el encabezado de la Rúbrica o Casos de Prueba para verificar los puntos de ponderación.',
      selector: '[data-tour="practice-eval-criteria"] .section-toggle-head',
      check: {
        isComplete: () => context.isEvalExpanded(),
        pendingHint: 'Hacé clic en el encabezado de criterios para desplegarlos.'
      }
    },
    {
      id: 'explore-actions',
      title: '3. Espacio de trabajo docente',
      body:
        'Utilizá los botones superiores para abrir el entorno de desarrollo o llevar la práctica directamente a tu clase en vivo.',
      selector: '[data-tour="practice-header-actions"]'
    }
  ];

  return {
    id: 'practice-detail-tutorial',
    title: 'Cómo explorar la ficha de una práctica',
    manualFragment: null,
    steps
  };
}
