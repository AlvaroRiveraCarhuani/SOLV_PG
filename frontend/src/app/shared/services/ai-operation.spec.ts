import { TestBed } from '@angular/core/testing';
import { Observable, Subject, throwError } from 'rxjs';
import { AiOperation, AiOperations, AiOperationOptions } from './ai-operation';
import { ToastService } from './toast.service';

function createToastSpy(): Pick<ToastService, 'show'> {
  return { show: vi.fn() };
}

function createOptions(
  overrides: Partial<AiOperationOptions<number>> = {}
): AiOperationOptions<number> {
  return {
    resolveError: () => 'falló',
    ...overrides
  };
}

describe('AiOperation', () => {
  it('pasa a running al lanzar y a ready al recibir el resultado', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const source = new Subject<number>();

    operation.run(source);

    expect(operation.isRunning()).toBe(true);
    expect(operation.state()).toBe('running');
    expect(operation.result()).toBeNull();

    source.next(42);
    source.complete();

    expect(operation.state()).toBe('ready');
    expect(operation.isRunning()).toBe(false);
    expect(operation.result()).toBe(42);
  });

  it('invoca onSuccess de forma sincrónica con el resultado', () => {
    const toast = createToastSpy();
    const onSuccess = vi.fn();
    const operation = new AiOperation<number>(
      toast as ToastService,
      createOptions({ onSuccess })
    );
    const source = new Subject<number>();

    operation.run(source);
    source.next(7);

    expect(onSuccess).toHaveBeenCalledWith(7);
  });

  it('marca error con el mensaje resuelto e invoca onError', () => {
    const toast = createToastSpy();
    const onError = vi.fn();
    const operation = new AiOperation<number>(
      toast as ToastService,
      createOptions({
        resolveError: () => 'cuota agotada',
        onError
      })
    );

    operation.run(throwError(() => new Error('boom')));

    expect(operation.state()).toBe('error');
    expect(operation.errorMessage()).toBe('cuota agotada');
    expect(onError).toHaveBeenCalledWith('cuota agotada');
    expect(operation.isRunning()).toBe(false);
  });

  it('cancelar aborta la suscripción y deja el estado en idle', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(
      toast as ToastService,
      createOptions({ notify: { cancel: 'Cancelado.' } })
    );
    const teardown = vi.fn();
    const source = new Observable<number>(() => teardown);

    operation.run(source);
    operation.cancel();

    expect(teardown).toHaveBeenCalled();
    expect(operation.state()).toBe('idle');
    expect(operation.isRunning()).toBe(false);
    expect(toast.show).toHaveBeenCalledWith('Cancelado.', 'info');
  });

  it('descarta una respuesta tardía del intento cancelado', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const source = new Subject<number>();

    operation.run(source);
    operation.cancel();
    source.next(99);

    expect(operation.result()).toBeNull();
    expect(operation.state()).toBe('idle');
  });

  it('reset limpia el resultado pero no interrumpe una llamada en vuelo', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const source = new Subject<number>();

    operation.run(source);
    source.next(5);
    operation.reset();

    expect(operation.result()).toBeNull();
    expect(operation.state()).toBe('idle');

    const running = new Subject<number>();
    operation.run(running);
    operation.reset();

    expect(operation.isRunning()).toBe(true);
    expect(operation.state()).toBe('running');
  });

  it('no relanza si ya hay una llamada en vuelo', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const first = new Subject<number>();
    const second = new Subject<number>();
    const secondSpy = vi.spyOn(second, 'subscribe');

    operation.run(first);
    operation.run(second);

    expect(secondSpy).not.toHaveBeenCalled();
  });

  it('emite el toast de éxito con el mensaje configurado', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(
      toast as ToastService,
      createOptions({ notify: { success: (result: number) => `Listo: ${result}` } })
    );
    const source = new Subject<number>();

    operation.run(source);
    source.next(3);

    expect(toast.show).toHaveBeenCalledWith('Listo: 3');
  });

  it('take entrega el resultado una sola vez y deja la operación en idle', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const source = new Subject<number>();

    operation.run(source);
    source.next(11);

    expect(operation.take()).toBe(11);
    expect(operation.state()).toBe('idle');
    expect(operation.result()).toBeNull();
    expect(operation.take()).toBeNull();
  });

  it('take no entrega nada mientras la llamada está en vuelo', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(toast as ToastService, createOptions());
    const source = new Subject<number>();

    operation.run(source);

    expect(operation.take()).toBeNull();
    expect(operation.isRunning()).toBe(true);
  });

  it('dispose corta la llamada sin emitir el toast de cancelación', () => {
    const toast = createToastSpy();
    const operation = new AiOperation<number>(
      toast as ToastService,
      createOptions({ notify: { cancel: 'Cancelado.' } })
    );
    const teardown = vi.fn();
    const source = new Observable<number>(() => teardown);

    operation.run(source);
    operation.dispose();

    expect(teardown).toHaveBeenCalled();
    expect(toast.show).not.toHaveBeenCalled();
  });
});

describe('AiOperations', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({});
  });

  it('devuelve la misma instancia para la misma clave', () => {
    const registry = TestBed.inject(AiOperations);
    const first = registry.get<number>('k', createOptions());
    const second = registry.get<number>('k', createOptions());

    expect(first).toBe(second);
  });

  it('devuelve instancias distintas para claves distintas', () => {
    const registry = TestBed.inject(AiOperations);
    const first = registry.get<number>('a', createOptions());
    const second = registry.get<number>('b', createOptions());

    expect(first).not.toBe(second);
  });
});
