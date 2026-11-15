import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías de Gestión de Docentes.
 *
 * El tutorial acompaña el alta individual hasta que el docente queda
 * registrado. Reenviar, renovar y revocar solo se explican: cada una cambia el
 * estado de una persona y conviene que el administrador la decida.
 */

export interface TeachersTutorialContext {
  /** El modal de alta está abierto. */
  inviteModalOpen: () => boolean;
  /** El correo escrito en el modal es válido. */
  emailValid: () => boolean;
  /** Se registró un docente en esta visita. */
  hasRegistered: () => boolean;
}

const MANUAL_FRAGMENT = '4-4-invitaciones-a-docentes';

/**
 * Recorrido de la pantalla: contadores, alta, pestañas de estado, filtros y
 * acciones por fila.
 */
export function buildTeachersTour(): TourDefinition {
  return {
    id: 'teachers-tour-v1',
    title: 'Recorrido por Gestión de Docentes',
    steps: [
      {
        selector: '[data-tour="teachers-summary"]',
        title: 'El resumen del plantel',
        description:
          'La línea bajo el título cuenta docentes registrados, activos, sin materias, pendientes y expirados. «Pendientes (72h)» son invitaciones vigentes sin aceptar; «Expirados» son las que ya vencieron.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teachers-create"]',
        title: 'Un alta, dos caminos',
        description:
          '«Registrar Docente» abre el alta individual o la importación de una planilla. El registro individual deja la cuenta habilitada de inmediato con Google SSO.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teachers-table"]',
        title: 'La tabla y sus pestañas',
        description:
          'Las pestañas filtran por estado; «Sin materias» muestra solo docentes activos que todavía no tienen ninguna asignada. La columna de materias abre el detalle y permite reasignar el titular.',
        side: 'top'
      },
      {
        selector: '[data-tour="teachers-filters"]',
        title: 'Búsqueda y origen',
        description:
          'La búsqueda cubre nombre y correo institucional. El filtro de origen separa las altas manuales de las sincronizadas desde Google Classroom.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teachers-row-actions"]',
        title: 'Cada estado ofrece acciones distintas',
        description:
          'Una invitación pendiente se reenvía o se revoca; una expirada se renueva por 72 horas más. Un docente activo se da de baja con confirmación, lo que desasigna sus materias.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial del alta individual: exige el modal abierto, un correo válido y el
 * registro confirmado. Termina explicando la vigencia y las acciones por fila.
 */
export function buildTeachersTutorial(context: TeachersTutorialContext): TutorialDefinition {
  return {
    id: 'teachers-invite-flow',
    title: 'Registrar un docente',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'summary',
        title: 'Dónde se ve el estado del plantel',
        body:
          'Antes de registrar, el resumen dice cuántos docentes hay y cuántas invitaciones siguen abiertas. Una invitación vence a las 72 horas y, pasado ese plazo, el enlace deja de servir.',
        selector: '[data-tour="teachers-summary"]',
        side: 'bottom'
      },
      {
        id: 'open',
        title: 'Abra el alta',
        body:
          '«Registrar Docente» abre el formulario. El alta individual habilita la cuenta con Google SSO; la importación de planilla registra varios correos a la vez y se explica en el manual.',
        selector: '[data-tour="teachers-create"]',
        side: 'bottom',
        check: {
          isComplete: () => context.inviteModalOpen(),
          pendingHint: 'Pulse «Registrar Docente» para continuar.'
        }
      },
      {
        id: 'tabs',
        title: 'Registro individual o planilla',
        body:
          'Esta guía acompaña el registro individual. La segunda pestaña importa una lista desde Excel o CSV y conviene usarla cuando el alta es masiva.',
        selector: '[data-tour="teacher-invite-tabs"]',
        side: 'bottom'
      },
      {
        id: 'email',
        title: 'El correo institucional',
        body:
          'El correo debe pertenecer al dominio de la institución. Si pertenece a otro dominio, la pantalla lo advierte: con Google Workspace restringido, esa persona no podrá iniciar sesión.',
        selector: '[data-tour="teacher-invite-email"]',
        side: 'left',
        check: {
          isComplete: () => context.emailValid(),
          pendingHint: 'Escriba un correo institucional válido para continuar.'
        }
      },
      {
        id: 'submit',
        title: 'Confirme el registro',
        body:
          'El botón permanece deshabilitado hasta que el correo es válido. Al confirmar, el docente queda habilitado y aparece en la tabla como activo; la acción queda registrada.',
        selector: '[data-tour="teacher-invite-submit"]',
        side: 'top',
        check: {
          isComplete: () => context.hasRegistered(),
          pendingHint: 'Confirme el registro para continuar.'
        }
      },
      {
        id: 'lifecycle',
        title: 'Reenviar, renovar y revocar',
        body:
          'Reenviar genera una invitación nueva con otras 72 horas; renovar hace lo mismo sobre una invitación expirada. Revocar anula el enlace vigente y la baja de un docente activo desasigna sus materias. Aquí termina la guía: estas acciones quedan en sus manos.',
        selector: '[data-tour="teachers-row-actions"]',
        side: 'left'
      }
    ]
  };
}
