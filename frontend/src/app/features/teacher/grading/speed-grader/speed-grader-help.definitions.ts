import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';
import { ReviewMode } from '../../models/teacher.models';

/**
 * Contexto reactivo para evaluar el estado real de la mesa de revisión de entregas.
 * La progresión del tutorial interactivo evalúa estas señales en tiempo real.
 */
export interface SpeedGraderTutorialContext {
  /** Modalidad de revisión (judge o project) */
  mode: () => ReviewMode;
  /** Indica si la entrega tiene código fuente cargado para revisar */
  hasCodeLoaded: () => boolean;
  /** Indica si el comparador de diff está desplegado */
  isDiffActive: () => boolean;
  /** Indica si el docente ya seleccionó un nivel de rúbrica, nota o tiene veredicto listo */
  hasEvaluationInput: () => boolean;
  /** Indica si la entrega quedó calificada */
  isGraded: () => boolean;
}

/**
 * Construye el recorrido visual (tour) de Revisión de Entrega adaptado a la modalidad.
 */
export function buildSpeedGraderTour(mode: ReviewMode, hasRubric: boolean): TourDefinition {
  const isJudge = mode === 'judge';

  return {
    id: `speed-grader-tour-${mode}`,
    title: 'Mesa de Revisión y Calificación',
    steps: [
      {
        selector: '[data-tour="review-header-nav"]',
        title: 'Navegación e Identidad de la Entrega',
        description:
          'Identificá al estudiante, la práctica y la modalidad. Utilizá las flechas o los atajos J (siguiente) y K (anterior) para recorrer las entregas de tus alumnos con fluidez.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-code-viewer"]',
        title: 'Inspección de Código y Comentarios',
        description:
          'Examiná el código enviado por el alumno. Podés hacer clic en cualquier número de línea (o presionar C) para dejar una retroalimentación puntual sobre esa parte del código.',
        side: 'right'
      },
      {
        selector: '[data-tour="review-diff-tool"]',
        title: 'Comparador con la Base',
        description:
          'Alterná la vista de diferencias (atajo D) para contrastar el código del estudiante contra la plantilla inicial o los archivos base del proyecto, identificando qué cambió con claridad.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-eval-panel"]',
        title: isJudge
          ? 'Resultados del Juez y Casos de Prueba'
          : hasRubric
            ? 'Evaluación por Rúbrica Ponderada'
            : 'Calificación Directa y Devolución',
        description: isJudge
          ? 'Observá el veredicto pedagógico («Cumple», «No cumple», etc.), el tiempo de ejecución y los casos de prueba ejecutados con sus entradas y salidas esperadas.'
          : hasRubric
            ? 'Seleccioná el nivel de desempeño alcanzado en cada criterio. El sistema calcula el puntaje ponderado automáticamente en tiempo real.'
            : 'Ingresá la calificación numérica (0 a 100 pts) y redactá tu retroalimentación general para la entrega de forma directa.',
        side: 'left'
      },
      {
        selector: '[data-tour="review-forensic-tools"]',
        title: 'Herramientas Forenses y Evolución',
        description:
          'En el menú «Herramientas» podés activar la Evolución Temporal (atajo T) para ver cómo fue creciendo el código paso a paso, o abrir la Auditoría de Escritura para inspeccionar la cadencia de tipeo.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-grade-action"]',
        title: isJudge ? 'Convalidar o Ajustar Nota' : 'Guardar Calificación',
        description: isJudge
          ? 'Pulsá «Calificar» (o atajo A) para abrir el modal de ajuste manual si necesitás convalidar un caso especial, corregir un veredicto o ingresar una nota personalizada.'
          : 'Confirmá la nota asignada y tu devolución pulsando «Guardar Calificación». El resultado queda registrado en la libreta de la materia de forma inmediata.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial interactivo guiado para corregir y evaluar una entrega.
 */
export function buildSpeedGraderTutorial(
  context: SpeedGraderTutorialContext
): TutorialDefinition {
  return {
    id: 'speed-grader-tutorial',
    title: 'Paso a paso: Cómo calificar una entrega',
    manualFragment: null,
    steps: [
      {
        id: 'inspect-code',
        title: '1. Inspeccionar el código',
        body:
          'Revisá el código fuente del estudiante en el visor central. Podés explorar los archivos del proyecto o pulsar sobre una línea para insertar un comentario puntual con el atajo C.',
        selector: '[data-tour="review-code-viewer"]',
        side: 'right',
        check: {
          isComplete: () => context.hasCodeLoaded(),
          pendingHint: 'Esperá a que cargue el código de la entrega.'
        }
      },
      {
        id: 'check-diff',
        title: '2. Contrastar con la base',
        body:
          'Pulsá la tecla D o hacé clic en «Herramientas > Comparar» para contrastar las modificaciones del estudiante respecto al código inicial de la cátedra.',
        selector: '[data-tour="review-diff-tool"]',
        side: 'bottom',
        check: {
          isComplete: () => context.isDiffActive() || context.hasCodeLoaded(),
          pendingHint: 'Probá activar el visor de diferencias con la tecla D.'
        }
      },
      {
        id: 'evaluate-submission',
        title: '3. Evaluar el desempeño',
        body:
          context.mode() === 'judge'
            ? 'Examiná el veredicto pedagógico y los casos de prueba ejecutados en el panel derecho. Si encontrás discrepancias, podés rectificar la nota con el atajo A.'
            : 'Completá los criterios de la rúbrica o ingresá la nota directa junto con tus comentarios pedagógicos en el panel lateral.',
        selector: '[data-tour="review-eval-panel"]',
        side: 'left',
        check: {
          isComplete: () => context.hasEvaluationInput(),
          pendingHint: 'Revisá los criterios o el estado de la evaluación en el panel derecho.'
        }
      },
      {
        id: 'finalize-grade',
        title: '4. Asentar la calificación',
        body:
          'Guardá la nota asignada para que quede registrada en la libreta de la materia. Si estás en modo de evaluación, podés aplicar el ajuste manual si fue necesario.',
        selector: '[data-tour="review-grade-action"]',
        side: 'left',
        check: {
          isComplete: () => context.isGraded(),
          pendingHint: 'Pulsá el botón para guardar o convalidar la calificación.'
        }
      }
    ]
  };
}
