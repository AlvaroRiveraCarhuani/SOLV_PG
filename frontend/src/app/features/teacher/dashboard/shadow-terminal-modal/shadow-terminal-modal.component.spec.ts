import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of, throwError } from 'rxjs';
import { vi, describe, beforeEach, it, expect } from 'vitest';
import { ShadowTerminalModalComponent } from './shadow-terminal-modal.component';
import { TeacherLiveService } from '../../services/teacher-live.service';
import { LiveWorkspaceSession, TutorCommandResponse } from '../../models/teacher.models';

describe('ShadowTerminalModalComponent', () => {
  let component: ShadowTerminalModalComponent;
  let fixture: ComponentFixture<ShadowTerminalModalComponent>;
  let liveService: TeacherLiveService;

  const mockSession: LiveWorkspaceSession = {
    workspace_id: 'ws-101',
    container_id: 'cnt-live-101',
    student_id: 'std-202',
    student_name: 'Estudiante Prueba',
    student_email: 'estudiante@universidad.edu',
    subject_id: 'sub-303',
    subject_name: 'Algoritmos y Estructuras',
    status: 'running',
    memory_limit_mb: 512,
    memory_used_mb: 180,
    cpu_percent: 25,
    oom_strikes: 0,
    last_heartbeat: '2026-10-03T14:30:00Z',
    is_attached: true,
    activity_state: 'typing'
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ShadowTerminalModalComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        TeacherLiveService
      ]
    }).compileComponents();

    liveService = TestBed.inject(TeacherLiveService);
    fixture = TestBed.createComponent(ShadowTerminalModalComponent);
    component = fixture.componentInstance;
    component.session = mockSession;
    fixture.detectChanges();
  });

  it('debe crearse correctamente en modo espejo inicial', () => {
    expect(component).toBeTruthy();
    expect(component.mode()).toBe('mirror');
    expect(component.session.container_id).toBe('cnt-live-101');
    expect(component.memoryPercent()).toBe(35); // 180 / 512 = 35%
  });

  it('debe alternar entre modo espejo y modo intervención tutorial', () => {
    component.setMode('tutor');
    expect(component.mode()).toBe('tutor');

    component.setMode('mirror');
    expect(component.mode()).toBe('mirror');
  });

  it('debe filtrar logs según la búsqueda en tiempo real', () => {
    component.appendLogEntry('Inicializando entorno...', 'system');
    component.appendLogEntry('$ git status', 'prompt');
    component.appendLogEntry('On branch main', 'stdout');
    component.appendLogEntry('Error: ModuleNotFoundError', 'error');

    component.searchQuery.set('ModuleNotFound');
    const filtered = component.filteredLogs();
    expect(filtered.length).toBe(1);
    expect(filtered[0].text).toContain('ModuleNotFoundError');

    component.searchQuery.set('');
    expect(component.filteredLogs().length).toBe(component.logEntries().length);
  });

  it('debe ejecutar comandos vía servicio REST y actualizar terminal al recibir respuesta', () => {
    const mockResponse: TutorCommandResponse = {
      container_id: 'cnt-live-101',
      command: 'git status',
      output: 'On branch main\nnothing to commit, working tree clean',
      exit_code: 0,
      executed_at: '2026-10-03T14:31:00Z'
    };

    vi.spyOn(liveService, 'executeTutorCommand').mockReturnValue(of(mockResponse));

    component.commandInput.set('git status');
    component.sendCommand();

    expect(liveService.executeTutorCommand).toHaveBeenCalledWith('cnt-live-101', 'git status');
    expect(component.commandInput()).toBe('');
    expect(component.commandHistory()).toContain('git status');
  });

  it('debe manejar errores de ejecución de comandos vía REST', () => {
    vi.spyOn(liveService, 'executeTutorCommand').mockReturnValue(
      throwError(() => ({ error: { message: 'Container unreachable' } }))
    );

    component.commandInput.set('python3 invalid.py');
    component.sendCommand();

    const lastLog = component.logEntries()[component.logEntries().length - 1];
    expect(lastLog.type).toBe('error');
    expect(lastLog.text).toContain('Container unreachable');
  });

  it('debe navegar por el historial de comandos con ArrowUp y ArrowDown', () => {
    component.commandHistory.set(['ls -la', 'git status', 'python3 main.py']);

    // ArrowUp: primer comando hacia atrás (python3 main.py)
    const upEvent1 = new KeyboardEvent('keydown', { key: 'ArrowUp' });
    component.onInputKeyDown(upEvent1);
    expect(component.commandInput()).toBe('python3 main.py');

    // ArrowUp: segundo comando hacia atrás (git status)
    const upEvent2 = new KeyboardEvent('keydown', { key: 'ArrowUp' });
    component.onInputKeyDown(upEvent2);
    expect(component.commandInput()).toBe('git status');

    // ArrowDown: volver adelante (python3 main.py)
    const downEvent1 = new KeyboardEvent('keydown', { key: 'ArrowDown' });
    component.onInputKeyDown(downEvent1);
    expect(component.commandInput()).toBe('python3 main.py');

    // ArrowDown: volver al input vacío
    const downEvent2 = new KeyboardEvent('keydown', { key: 'ArrowDown' });
    component.onInputKeyDown(downEvent2);
    expect(component.commandInput()).toBe('');
  });

  it('debe ejecutar presets de diagnóstico y cambiar automáticamente a modo tutor', () => {
    const sendSpy = vi.spyOn(component, 'sendCommand').mockImplementation(() => {});
    component.mode.set('mirror');

    component.runDiagnostic('free -m');
    expect(component.mode()).toBe('tutor');
    expect(component.commandInput()).toBe('free -m');
    expect(sendSpy).toHaveBeenCalled();
  });

  it('debe alternar fullscreen, timestamps y autoScroll', () => {
    expect(component.isFullscreen()).toBe(false);
    component.toggleFullscreen();
    expect(component.isFullscreen()).toBe(true);

    expect(component.showTimestamps()).toBe(false);
    component.toggleTimestamps();
    expect(component.showTimestamps()).toBe(true);

    expect(component.autoScroll()).toBe(true);
    component.toggleAutoScroll();
    expect(component.autoScroll()).toBe(false);
  });

  it('debe limpiar terminal y emitir cierre en escape', () => {
    component.appendLogEntry('Texto de prueba', 'stdout');
    expect(component.logEntries().length).toBeGreaterThan(0);

    component.clearTerminal();
    expect(component.logEntries().length).toBe(0);

    const closeSpy = vi.spyOn(component.close, 'emit');
    component.onEscape();
    expect(closeSpy).toHaveBeenCalled();
  });

  it('debe copiar logs al portapapeles y exportar snapshot', () => {
    const writeMock = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeMock },
      configurable: true
    });

    component.appendLogEntry('Log 1', 'stdout');
    component.appendLogEntry('Log 2', 'stdout');

    component.copyLogs();
    expect(writeMock).toHaveBeenCalled();

    // Export session snapshot
    if (typeof URL.createObjectURL !== 'function') {
      URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
      URL.revokeObjectURL = vi.fn();
    } else {
      vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test');
      vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
    }

    component.exportSessionSnapshot();
    expect(component.isExported()).toBe(true);
  });
});
