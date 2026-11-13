import { TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial del catálogo de plantillas: recorre la bandeja de revisión y valida
 * que el administrador esté realmente en la pestaña que toca.
 *
 * Las plantillas se revisan en un modal que queda por encima del panel, así que
 * la guía se queda en la página y explica la ficha sin exigir abrirla.
 */

export interface CatalogTutorialContext {
  /** Pestaña visible del catálogo. */
  activeTab: () => 'pending' | 'catalog' | 'rejected';
  /** Plantillas que esperan aprobación; la pestaña se oculta cuando son 0. */
  pendingCount: () => number;
}

const MANUAL_FRAGMENT = '3-2-ciclo-de-vida-y-gobernanza-de-plantillas';

export function buildCatalogTutorial(context: CatalogTutorialContext): TutorialDefinition {
  return {
    id: 'catalog-review-first-template',
    title: 'Revisar y aprobar una plantilla',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'purpose',
        title: 'Cómo funciona esta bandeja',
        body:
          'Al publicar desde el constructor, la plantilla entra activa de inmediato. La pestaña «En aprobación» aparece sólo cuando queda algo pendiente de auditoría, por ejemplo una copia duplicada o una plantilla heredada.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom'
      },
      {
        id: 'pending',
        title: 'Plantillas en aprobación',
        body:
          'La pestaña sólo se muestra cuando hay algo que aprobar. El contador indica cuántas esperan tu decisión; si está en cero, no hay nada pendiente hoy.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom',
        check: {
          // Sin pendientes la pestaña no existe: el paso se da por cumplido.
          isComplete: () => context.activeTab() === 'pending' || context.pendingCount() === 0,
          pendingHint: 'Abra la pestaña «En aprobación» para continuar.'
        }
      },
      {
        id: 'card',
        title: 'Qué mirar en cada tarjeta',
        body:
          'Quién la creó, tecnologías pedidas, memoria por alumno e imagen. La justificación pedagógica explica para qué se usará en clase y conviene leerla antes de decidir.',
        selector: '[data-tour="templates-list"]',
        side: 'top'
      },
      {
        id: 'review',
        title: 'Revisar y fijar cuotas',
        body:
          '«Revisar y Aprobar» abre la ficha, donde se confirma la tecnología y se ajusta la memoria por alumno antes de aprobar. «Rechazar» pide un motivo de al menos 10 caracteres, que el docente verá para corregir.',
        selector: '[data-tour="templates-list"]',
        side: 'top'
      },
      {
        id: 'catalog',
        title: 'Ver la plantilla publicada',
        body:
          'Al aprobar, la plantilla pasa a «Catálogo Activo» y los docentes pueden asignarla a sus materias. Si algo falla después, puede suspenderse sin borrar el historial de los cursos anteriores.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom',
        check: {
          isComplete: () => context.activeTab() === 'catalog',
          pendingHint: 'Abra la pestaña «Catálogo Activo» para terminar.'
        }
      }
    ]
  };
}
