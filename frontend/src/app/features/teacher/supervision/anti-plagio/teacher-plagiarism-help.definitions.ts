import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado real de la auditoría de similitud.
 * La progresión del tutorial lee directamente este contrato desde Angular Signals.
 */
export interface TeacherPlagiarismTutorialContext {
  /** Indica si se cuenta con un reporte de similitud cargado */
  hasReport: () => boolean;
  /** Cantidad de coincidencias visibles con el filtro actual */
  matchesCount: () => number;
  /** Indica si el docente seleccionó un par de alumnos para analizar */
  isMatchSelected: () => boolean;
  /** Indica si los archivos del par seleccionado ya cargaron en el visor */
  hasPairLoaded: () => boolean;
  /** Nivel de riesgo activo en el filtro ('all', 'critical', 'warning', 'info') */
  currentRiskFilter: () => string;
  /** Indica si la práctica pertenece a un Entorno de Desarrollo con múltiples archivos */
  isMultiFile: () => boolean;
}

/**
 * Recorrido visual de la Auditoría de Similitud.
 * Guía la lectura del resumen compacto, re-escaneo, pares, visor diff y telemetría forense.
 */
export function buildTeacherPlagiarismTour(): TourDefinition {
  return {
    id: 'teacher-plagiarism-tour-v1',
    title: 'Recorrido por la Auditoría de Similitud',
    steps: [
      {
        selector: '[data-tour="plagiarism-summary"]',
        title: 'Resumen Compacto y Filtros Rápidos',
        description:
          'Muestra el balance de pares analizados, coincidencias críticas y alumnos involucrados. Podés pulsar cualquiera de los valores para filtrar la lista al instante.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="plagiarism-actions"]',
        title: 'Re-escanear Entregas',
        description:
          'Si llegaron entregas recientes o se realizaron cambios en las notas, «Re-escanear Práctica» vuelve a computar los árboles de sintaxis restando el código base.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="plagiarism-pairs-list"]',
        title: 'Pares de Alumnos con Coincidencias',
        description:
          'Lista de estudiantes donde se detectó afinidad estructural. Cada tarjeta exhibe el porcentaje de similitud y un indicador visual según su nivel de criticidad.',
        side: 'right'
      },
      {
        selector: '[data-tour="plagiarism-files-list"]',
        title: 'Archivos Afectados del Proyecto',
        description:
          'En prácticas con Entorno de Desarrollo, esta lista separa los archivos con tramos coincidentes para que no pierdas tiempo en archivos de configuración estándar.',
        side: 'right'
      },
      {
        selector: '[data-tour="plagiarism-diff-viewer"]',
        title: 'Visor Comparativo con Scroll Sincronizado',
        description:
          'Compara ambos códigos en paralelo destacando los bloques idénticos o renombrados. El desplazamiento sincronizado te permite seguir la lectura de ambos al mismo tiempo.',
        side: 'left'
      },
      {
        selector: '[data-tour="plagiarism-forensic-metrics"]',
        title: 'Telemetría y Evidencia Forense',
        description:
          'Aporta datos objetivos de autoría: cadencia de tipeo en palabras por minuto (WPM), tiempo en el editor y eventos de pegado masivo para respaldar tu valoración pedagógica.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial interactivo paso a paso para examinar coincidencias sospechosas con criterio docente.
 */
export function buildTeacherPlagiarismTutorial(
  context: TeacherPlagiarismTutorialContext
): TutorialDefinition {
  return {
    id: 'teacher-plagiarism-tutorial',
    title: 'Cómo analizar coincidencias de código',
    manualFragment: null,
    steps: [
      {
        id: 'filter-by-risk',
        title: '1. Enfocar por nivel de criticidad',
        body:
          'Filtrá por «Crítico» (>= 85%) o «Advertencia» (70-84%) usando la cabecera o el selector para priorizar los casos con mayor similitud estructural.',
        selector: '[data-tour="plagiarism-summary"]',
        side: 'bottom',
        check: {
          isComplete: () => context.currentRiskFilter() !== 'all' || context.matchesCount() > 0,
          pendingHint: 'Pulsá sobre una categoría de riesgo en el resumen de cabecera.'
        }
      },
      {
        id: 'select-match',
        title: '2. Seleccionar un par de alumnos',
        body:
          'Hacé clic en una tarjeta de la lista izquierda para cargar ambos envíos y sus métricas asociadas en el panel central.',
        selector: '[data-tour="plagiarism-pairs-list"]',
        side: 'right',
        check: {
          isComplete: () => context.isMatchSelected(),
          pendingHint: 'Seleccioná un par de estudiantes de la lista de coincidencias.'
        }
      },
      {
        id: 'review-code-diff',
        title: '3. Examinar tramos en el comparador',
        body:
          'Revisá los bloques coloreados en el visor. El motor ya restó la plantilla dada en clase, por lo que las marcas corresponden exclusivamente a lógica aportada por los alumnos.',
        selector: '[data-tour="plagiarism-diff-viewer"]',
        side: 'left',
        check: {
          isComplete: () => context.hasPairLoaded(),
          pendingHint: 'Asegurate de que los archivos del par estén cargados en el visor.'
        }
      },
      {
        id: 'evaluate-forensic-metrics',
        title: '4. Contrastar con la telemetría forense',
        body:
          'Mirá la columna de métricas: un código similar pero escrito a 40 WPM con tiempo razonable indica proceso personal; 0 WPM y pegado masivo aporta certeza objetiva para tu devolución.',
        selector: '[data-tour="plagiarism-forensic-metrics"]',
        side: 'left',
        check: {
          isComplete: () => context.hasPairLoaded(),
          pendingHint: 'Inspeccioná las métricas de tipeo y pegado en la columna derecha.'
        }
      }
    ]
  };
}
