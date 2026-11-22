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
          'Identifique al estudiante, la práctica y la modalidad. Utilice las flechas o los atajos J (siguiente) y K (anterior) para recorrer las entregas de sus alumnos con fluidez.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-code-viewer"]',
        title: 'Inspección de Código y Comentarios',
        description:
          'Examine el código enviado por el alumno. Puede hacer clic en cualquier número de línea (o presionar C) para dejar una observación puntual sobre esa parte del código.',
        side: 'right'
      },
      {
        selector: '[data-tour="review-diff-tool"]',
        title: 'Comparador con la Base',
        description:
          'Alterne la vista de diferencias (atajo D) para contrastar el código del estudiante con la plantilla inicial o los archivos base del proyecto e identificar qué cambió.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-eval-panel"]',
        title: isJudge
          ? 'Resultados de la evaluación y casos de prueba'
          : hasRubric
            ? 'Evaluación por rúbrica'
            : 'Nota directa y devolución',
        description: isJudge
          ? 'Observe el veredicto humanizado, el tiempo de ejecución y los casos de prueba con sus entradas y salidas esperadas.'
          : hasRubric
            ? 'Seleccione el nivel alcanzado en cada criterio. El sistema calcula la calificación ponderada en tiempo real.'
            : 'Ingrese la calificación numérica (0 a 100) y redacte su retroalimentación general para la entrega.',
        side: 'left'
      },
      {
        selector: '[data-tour="review-forensic-tools"]',
        title: 'Herramientas de inspección y evolución temporal',
        description:
          'En el menú «Herramientas» puede activar la Evolución temporal (atajo T) para ver cómo creció el código paso a paso, o abrir la Auditoría de escritura para revisar la actividad de tipeo.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="review-grade-action"]',
        title: isJudge ? 'Convalidar o ajustar la nota' : 'Guardar Calificación',
        description: isJudge
          ? 'Pulse «Calificar» (atajo A) para abrir el ajuste manual si necesita corregir un veredicto o registrar una nota personalizada.'
          : 'Confirme la nota y su devolución con «Guardar Calificación». El resultado queda registrado en la libreta de la materia.',
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
          'Revise el código fuente del estudiante en el visor central. Puede explorar los archivos del proyecto o pulsar sobre una línea para insertar un comentario puntual con el atajo C.',
        selector: '[data-tour="review-code-viewer"]',
        side: 'right',
        check: {
          isComplete: () => context.hasCodeLoaded(),
          pendingHint: 'Espere a que cargue el código de la entrega.'
        }
      },
      {
        id: 'check-diff',
        title: '2. Contrastar con la base',
        body:
          'Pulse la tecla D o haga clic en «Herramientas > Comparar» para contrastar las modificaciones del estudiante con el código inicial de la cátedra.',
        selector: '[data-tour="review-diff-tool"]',
        side: 'bottom',
        check: {
          isComplete: () => context.isDiffActive() || context.hasCodeLoaded(),
          pendingHint: 'Pruebe activar el visor de diferencias con la tecla D.'
        }
      },
      {
        id: 'evaluate-submission',
        title: '3. Evaluar el desempeño',
        body:
          context.mode() === 'judge'
            ? 'Examine el veredicto y los casos de prueba del panel derecho. Si encuentra discrepancias, puede rectificar la nota con el atajo A.'
            : 'Complete los criterios de la rúbrica o ingrese la nota directa junto con sus comentarios en el panel lateral.',
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
          'Guarde la nota asignada para que quede registrada en la libreta de la materia. En Entorno de Evaluación puede aplicar el ajuste manual si fue necesario.',
        selector: '[data-tour="review-grade-action"]',
        side: 'left',
        check: {
          isComplete: () => context.isGraded(),
          pendingHint: 'Pulse el botón para guardar o convalidar la calificación.'
        }
      }
    ]
  };
}
