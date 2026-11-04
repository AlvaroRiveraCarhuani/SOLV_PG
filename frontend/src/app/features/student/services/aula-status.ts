export type AulaStatusTone = 'success' | 'warning' | 'neutral';

/**
 * Estado del aula en lenguaje humano. La portada y el aula muestran el mismo
 * texto y el mismo tono para no describir la misma sesión de dos formas.
 */
export function aulaStatusLabel(status: string): string {
  switch (status) {
    case 'open':
      return 'Abierta';
    case 'frozen':
      return 'Congelada';
    case 'closed':
      return 'Cerrada';
    default:
      return 'Sin estado';
  }
}

export function aulaStatusTone(status: string): AulaStatusTone {
  switch (status) {
    case 'open':
      return 'success';
    case 'frozen':
      return 'warning';
    default:
      return 'neutral';
  }
}
