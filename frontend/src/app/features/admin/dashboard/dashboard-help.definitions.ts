import { TourDefinition } from '@shared/help/help.types';

/**
 * Recorrido del dashboard de salud.
 *
 * No lleva tutorial: no hay un flujo por pasos que comprobar. «Hibernar todo»
 * es un clic con confirmación y las operaciones de infraestructura quedan en
 * manos del administrador.
 */
export function buildDashboardTour(): TourDefinition {
  return {
    id: 'dashboard-health-tour-v1',
    title: 'Recorrido por Salud y Recursos',
    steps: [
      {
        selector: '[data-tour="dashboard-health"]',
        title: 'El veredicto y sus motivos',
        description:
          'El indicador resume el estado del servidor y del motor Docker. Los motivos listados debajo explican por qué está degradado o crítico; sin motivos, el sistema está operativo.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="dashboard-actions"]',
        title: 'Actualizar, personalizar y hibernar',
        description:
          'El auto-refresco relee las métricas cada 10 segundos. «Personalizar» reordena los bloques. «Hibernar Todo» suspende los laboratorios activos y pide confirmación con el alcance real: las sesiones abiertas se cierran y los entornos quedan en reposo.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="dashboard-kpis"]',
        title: 'Los recursos del host',
        description:
          'Memoria usada sobre total, carga de vCPU, almacenamiento y concurrencia de contenedores. Cada tarjeta declara su umbral: un valor alto no siempre es una falla, pero explica una clase lenta.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="dashboard-chart"]',
        title: 'La carga en el tiempo',
        description:
          'La curva muestra el consumo de los últimos 60 minutos y la línea de capacidad. Sirve para distinguir un pico de una tendencia antes de abrir una clase.',
        side: 'top'
      },
      {
        selector: '[data-tour="dashboard-load"]',
        title: 'Materias e incidencias',
        description:
          'A la izquierda, la carga por materia con sus entornos abiertos. A la derecha, las incidencias técnicas con la acción disponible: reiniciar, detener o ver logs.',
        side: 'top'
      },
      {
        selector: '[data-tour="dashboard-containers"]',
        title: 'Las instancias activas',
        description:
          'La tabla lista alumno, materia, imagen, memoria, TTL y estado de red. Detener un contenedor pide confirmación y cierra la sesión del estudiante.',
        side: 'top'
      }
    ]
  };
}
