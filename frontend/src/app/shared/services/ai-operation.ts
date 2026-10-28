import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, Subscription } from 'rxjs';
import { ToastService } from './toast.service';

export type AiOperationState = 'idle' | 'running' | 'ready' | 'error';

/**
 * Mensajes de aviso de una operación de IA. Cada campo es opcional: cuando no
 * se define, el feature muestra el resultado en su propia UI (inline) y no se
 * emite toast global.
 */
export interface AiOperationNotify<T> {
  success?: string | ((result: T) => string);
  error?: (message: string) => string;
  cancel?: string;
}

export interface AiOperationOptions<T> {
  notify?: AiOperationNotify<T>;
  resolveError: (err: unknown) => string;
  /** Se invoca de forma síncrona al recibir el resultado. */
  onSuccess?: (result: T) => void;
  /** Se invoca de forma síncrona al recibir un error, con el mensaje resuelto. */
  onError?: (message: string) => void;
}

/**
 * Ciclo de vida de una llamada a IA: idle -> running -> ready | error.
 *
 * Es una clase plana, sin DI, para poder testearla aislada. El dueño del
 * estado es quien la instancia; ver {@link AiOperations} para las operaciones
 * que deben sobrevivir a la navegación.
 */
export class AiOperation<T> {
  readonly state = signal<AiOperationState>('idle');
  readonly result = signal<T | null>(null);
  readonly errorMessage = signal<string | null>(null);
  readonly elapsedSeconds = signal<string>('0.0');
  readonly isRunning = computed<boolean>(() => this.state() === 'running');

  private subscription?: Subscription;
  private timer?: ReturnType<typeof setInterval>;
  private startedAt = 0;
  /** Descarta respuestas de intentos anteriores tras cancelar o relanzar. */
  private attempt = 0;

  constructor(
    private readonly toastService: ToastService,
    private readonly options: AiOperationOptions<T>
  ) {}

  run(source: Observable<T>): void {
    if (this.isRunning()) return;

    this.attempt += 1;
    const attempt = this.attempt;

    this.state.set('running');
    this.errorMessage.set(null);
    this.elapsedSeconds.set('0.0');
    this.startTimer();

    this.subscription?.unsubscribe();
    this.subscription = source.subscribe({
      next: result => {
        if (attempt !== this.attempt) return;
        this.stopTimer();
        this.subscription = undefined;
        this.result.set(result);
        this.state.set('ready');
        this.options.onSuccess?.(result);
        this.notifySuccess(result);
      },
      error: (err: unknown) => {
        if (attempt !== this.attempt) return;
        this.stopTimer();
        this.subscription = undefined;
        const message = this.options.resolveError(err);
        this.errorMessage.set(message);
        this.state.set('error');
        this.options.onError?.(message);
        this.notifyError(message);
      }
    });
  }

  /** Aborta la llamada en vuelo. Deja el resultado previo intacto. */
  cancel(): void {
    const wasRunning = this.isRunning();
    this.attempt += 1;
    this.subscription?.unsubscribe();
    this.subscription = undefined;
    this.stopTimer();
    this.state.set('idle');
    if (wasRunning && this.options.notify?.cancel) {
      this.toastService.show(this.options.notify.cancel, 'info');
    }
  }

  /**
   * Limpia el resultado y vuelve a `idle`. No interrumpe una llamada en vuelo:
   * eso es responsabilidad de {@link cancel}.
   */
  reset(): void {
    if (this.isRunning()) return;
    this.result.set(null);
    this.errorMessage.set(null);
    this.elapsedSeconds.set('0.0');
    this.state.set('idle');
  }

  /**
   * Reclama el resultado listo y lo consume: devuelve el valor una sola vez y
   * deja la operación en `idle`. Es un *claim*, no un *peek*, así dos caminos
   * distintos de restauración no pueden aplicar el mismo resultado dos veces.
   */
  take(): T | null {
    const result = this.result();
    if (result === null || this.state() !== 'ready') return null;
    this.reset();
    return result;
  }

  /**
   * Libera recursos al morir el dueño de la operación, sin avisar al usuario.
   * A diferencia de {@link cancel}, no emite el toast de cancelación.
   */
  dispose(): void {
    this.attempt += 1;
    this.subscription?.unsubscribe();
    this.subscription = undefined;
    this.stopTimer();
  }

  private notifySuccess(result: T): void {
    const success = this.options.notify?.success;
    if (!success) return;
    const message = typeof success === 'function' ? success(result) : success;
    if (message) {
      this.toastService.show(message);
    }
  }

  private notifyError(message: string): void {
    const error = this.options.notify?.error;
    if (!error) return;
    this.toastService.show(error(message), 'error');
  }

  private startTimer(): void {
    this.stopTimer();
    this.startedAt = performance.now();
    this.timer = setInterval(() => {
      this.elapsedSeconds.set(((performance.now() - this.startedAt) / 1000).toFixed(1));
    }, 100);
  }

  private stopTimer(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = undefined;
    }
  }
}

/**
 * Registro de operaciones de IA con dueño en la raíz de la aplicación.
 *
 * Una operación obtenida de acá no muere con el componente que la lanzó: la
 * llamada sigue, el resultado queda disponible y el aviso llega por el toast
 * global aunque el usuario haya navegado a otra vista.
 */
@Injectable({
  providedIn: 'root'
})
export class AiOperations {
  private readonly toastService = inject(ToastService);
  private readonly operations = new Map<string, AiOperation<unknown>>();

  get<T>(key: string, options: AiOperationOptions<T>): AiOperation<T> {
    const existing = this.operations.get(key);
    if (existing) {
      return existing as AiOperation<T>;
    }
    const operation = new AiOperation<T>(this.toastService, options);
    this.operations.set(key, operation as AiOperation<unknown>);
    return operation;
  }
}
