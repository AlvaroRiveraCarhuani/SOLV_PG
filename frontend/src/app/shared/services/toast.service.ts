import { Injectable, signal } from '@angular/core';

export type ToastType = 'success' | 'error' | 'info';

export interface ToastMessage {
  id: number;
  message: string;
  type: ToastType;
}

/**
 * Toast global de la aplicación. Vive en la raíz para que cualquier servicio
 * de larga duración (por ejemplo el Copiloto IA) pueda avisar aunque el
 * componente que originó el pedido ya no esté montado.
 */
@Injectable({
  providedIn: 'root'
})
export class ToastService {
  private sequence = 0;
  private timer?: ReturnType<typeof setTimeout>;

  private readonly _toast = signal<ToastMessage | null>(null);
  readonly toast = this._toast.asReadonly();

  show(message: string, type: ToastType = 'success', durationMs = 5000): void {
    this.clearTimer();
    this.sequence += 1;
    this._toast.set({ id: this.sequence, message, type });
    const id = this.sequence;
    this.timer = setTimeout(() => this.dismiss(id), durationMs);
  }

  dismiss(id: number): void {
    if (this._toast()?.id !== id) return;
    this.clearTimer();
    this._toast.set(null);
  }

  clear(): void {
    this.clearTimer();
    this._toast.set(null);
  }

  private clearTimer(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = undefined;
    }
  }
}
