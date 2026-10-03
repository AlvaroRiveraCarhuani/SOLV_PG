import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  OnInit, 
  OnDestroy, 
  inject, 
  signal, 
  computed,
  ElementRef, 
  ViewChild, 
  HostListener 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideTerminal, 
  LucideX, 
  LucideTrash2, 
  LucideCopy, 
  LucideCheck, 
  LucideMaximize2, 
  LucideMinimize2, 
  LucideSend,
  LucideLock,
  LucideUnlock,
  LucideDownload,
  LucideSearch,
  LucidePlay
} from '@lucide/angular';
import { LiveWorkspaceSession } from '../../models/teacher.models';
import { TeacherLiveService } from '../../services/teacher-live.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe, formatSolvDate } from '@shared/pipes/date-text.pipe';

export interface TerminalLogEntry {
  id: string;
  timestamp: string;
  text: string;
  type: 'system' | 'prompt' | 'stdout' | 'stderr' | 'error' | 'success';
}

export interface MacroPreset {
  label: string;
  command: string;
  category: 'diag' | 'test' | 'sys';
  description?: string;
}

@Component({
  selector: 'shadow-terminal-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideTerminal,
    LucideX,
    LucideTrash2,
    LucideCopy,
    LucideCheck,
    LucideMaximize2,
    LucideMinimize2,
    LucideSend,
    LucideLock,
    LucideUnlock,
    LucideDownload,
    LucideSearch,
    LucidePlay,
    MachineDataDirective,
    DateTextPipe
  ],
  templateUrl: './shadow-terminal-modal.component.html',
  styleUrl: './shadow-terminal-modal.component.scss'
})
export class ShadowTerminalModalComponent implements OnInit, OnDestroy {
  @Input({ required: true }) session!: LiveWorkspaceSession;
  @Output() close = new EventEmitter<void>();

  private liveService = inject(TeacherLiveService);

  @ViewChild('terminalOutput') private terminalContainer?: ElementRef<HTMLDivElement>;
  @ViewChild('promptInputField') private promptInputRef?: ElementRef<HTMLInputElement>;

  // Operating Modes: 'mirror' (Read-Only Observer) vs 'tutor' (Interactive Read/Write Intervention)
  readonly mode = signal<'mirror' | 'tutor'>('mirror');

  // Logs & Streaming
  readonly logEntries = signal<TerminalLogEntry[]>([]);
  readonly isConnected = signal<boolean>(false);
  readonly autoScroll = signal<boolean>(true);
  readonly isFullscreen = signal<boolean>(false);
  readonly isCopied = signal<boolean>(false);
  readonly isExported = signal<boolean>(false);
  readonly showTimestamps = signal<boolean>(false);
  readonly searchQuery = signal<string>('');

  // Command Execution & History
  readonly commandInput = signal<string>('');
  readonly isExecuting = signal<boolean>(false);
  readonly commandHistory = signal<string[]>([]);
  readonly historyIndex = signal<number>(-1);

  // Macro Presets Toolbelt
  readonly activeCategory = signal<'diag' | 'test' | 'sys'>('diag');
  readonly macroPresets: MacroPreset[] = [
    { label: 'git status', command: 'git status', category: 'diag', description: 'Estado de archivos modificados' },
    { label: 'git diff', command: 'git diff', category: 'diag', description: 'Diferencias no commiteadas' },
    { label: 'ls -la', command: 'ls -la', category: 'diag', description: 'Árbol detallado del workspace' },
    { label: 'pytest', command: 'pytest -v', category: 'test', description: 'Ejecutar suite de tests Python' },
    { label: 'python3 main.py', command: 'python3 main.py', category: 'test', description: 'Ejecutar script principal' },
    { label: 'npm test', command: 'npm test', category: 'test', description: 'Ejecutar tests JS/TS' },
    { label: 'free -m', command: 'free -m', category: 'sys', description: 'Consumo de memoria en MB' },
    { label: 'ps aux', command: 'ps aux --sort=-%mem | head -n 10', category: 'sys', description: 'Procesos con mayor consumo' },
    { label: 'cat /proc/meminfo', command: 'cat /proc/meminfo | head -n 10', category: 'sys', description: 'Telemetría cgroup RAM' }
  ];

  readonly currentCategoryPresets = computed(() => {
    const cat = this.activeCategory();
    return this.macroPresets.filter(p => p.category === cat);
  });

  readonly filteredLogs = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const logs = this.logEntries();
    if (!query) return logs;
    return logs.filter(entry => entry.text.toLowerCase().includes(query));
  });

  readonly memoryPercent = computed(() => {
    const limit = this.session?.memory_limit_mb || 512;
    const used = this.session?.memory_used_mb || 0;
    return Math.min(100, Math.round((used / limit) * 100));
  });

  private ws?: WebSocket;
  private pingInterval?: ReturnType<typeof setInterval>;

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.onClose();
  }

  @HostListener('document:keydown', ['$event'])
  onGlobalKeyDown(event: KeyboardEvent): void {
    if (event.ctrlKey && event.key.toLowerCase() === 'l') {
      event.preventDefault();
      this.clearTerminal();
    }
  }

  ngOnInit(): void {
    this.connectWebSocket();
  }

  ngOnDestroy(): void {
    this.disconnectWebSocket();
  }

  setMode(m: 'mirror' | 'tutor'): void {
    this.mode.set(m);
    if (m === 'tutor') {
      this.appendLogEntry(
        '[SOLV Shadow Mode] Modo de Intervención Tutorial activado. Los comandos se ejecutarán en el contenedor del alumno.',
        'system'
      );
      setTimeout(() => this.promptInputRef?.nativeElement?.focus(), 50);
    } else {
      this.appendLogEntry(
        '[SOLV Shadow Mode] Modo Espejo (Solo Lectura) activado. Visualización pasiva de actividad.',
        'system'
      );
    }
  }

  private connectWebSocket(): void {
    if (!this.session?.container_id) {
      this.appendLogEntry('[SOLV Shadow Mode] Sin contenedor asignado o en proceso de inicio.', 'system');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/teacher/live-sessions/${this.session.container_id}/terminal`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.isConnected.set(true);
        this.appendLogEntry(`[SOLV Shadow Mode] Conexión establecida con el contenedor: ${this.session.container_id}`, 'system');
        this.appendLogEntry(`[Terminal Mirror] Estudiante: ${this.session.student_name} (${this.session.student_email})`, 'system');
        this.appendLogEntry('────────────────────────────────────────────────────────────────────────', 'system');

        this.pingInterval = setInterval(() => {
          if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'ping' }));
          }
        }, 15000);
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'init' && data.buffer) {
            this.parseAndAppendOutput(data.buffer);
          } else if (data.type === 'stdout' && data.output) {
            this.parseAndAppendOutput(data.output);
          } else if (data.type === 'stderr' && data.output) {
            this.parseAndAppendOutput(data.output, 'stderr');
          } else if (data.type === 'error') {
            this.appendLogEntry(`[Error]: ${data.error}`, 'error');
          }
        } catch {
          this.parseAndAppendOutput(event.data);
        }
      };

      this.ws.onclose = () => {
        this.isConnected.set(false);
        this.appendLogEntry('[SOLV Shadow Mode] Conexión finalizada.', 'system');
      };

      this.ws.onerror = () => {
        this.isConnected.set(false);
        this.appendLogEntry('[SOLV Shadow Mode] Fallback: Simulador de terminal activo (modo seguro).', 'system');
      };
    } catch {
      this.isConnected.set(false);
      this.appendLogEntry('[SOLV Shadow Mode] Simulación de terminal activa.', 'system');
    }
  }

  private disconnectWebSocket(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = undefined;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = undefined;
    }
  }

  private parseAndAppendOutput(raw: string, defaultType: TerminalLogEntry['type'] = 'stdout'): void {
    if (!raw) return;
    const lines = raw.split('\n');
    for (const line of lines) {
      if (!line && lines.length > 1) continue;
      let type: TerminalLogEntry['type'] = defaultType;
      const trimmed = line.trim();
      if (trimmed.startsWith('$ ') || trimmed.startsWith('>>> ')) {
        type = 'prompt';
      } else if (trimmed.includes('Error') || trimmed.includes('FAIL') || trimmed.includes('Traceback')) {
        type = 'error';
      } else if (trimmed.includes('PASS') || trimmed.includes('OK') || trimmed.includes('Successfully')) {
        type = 'success';
      }
      this.appendLogEntry(line, type);
    }
  }

  appendLogEntry(text: string, type: TerminalLogEntry['type'] = 'stdout'): void {
    const entry: TerminalLogEntry = {
      id: `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
      timestamp: formatSolvDate(new Date(), 'time') || '',
      text,
      type
    };

    this.logEntries.update(logs => [...logs, entry]);
    if (this.autoScroll()) {
      setTimeout(() => this.scrollToBottom(), 30);
    }
  }

  appendLog(line: string): void {
    this.parseAndAppendOutput(line);
  }

  clearTerminal(): void {
    this.logEntries.set([]);
  }

  toggleAutoScroll(): void {
    this.autoScroll.update(val => !val);
  }

  toggleFullscreen(): void {
    this.isFullscreen.update(val => !val);
  }

  toggleTimestamps(): void {
    this.showTimestamps.update(val => !val);
  }

  copyLogs(): void {
    const text = this.logEntries().map(e => this.showTimestamps() ? `[${e.timestamp}] ${e.text}` : e.text).join('\n');
    navigator.clipboard.writeText(text).then(() => {
      this.isCopied.set(true);
      setTimeout(() => this.isCopied.set(false), 2000);
    });
  }

  exportSessionSnapshot(): void {
    const lines: string[] = [
      '========================================================================',
      'SOLV VIRTUAL LABS — SHADOW MODE TERMINAL TRANSCRIPT',
      `Fecha: ${new Date().toISOString()}`,
      `Estudiante: ${this.session.student_name} (${this.session.student_email})`,
      `Contenedor: ${this.session.container_id}`,
      `Materia: ${this.session.subject_name}`,
      `Límite RAM: ${this.session.memory_limit_mb} MB`,
      '========================================================================\n'
    ];

    for (const entry of this.logEntries()) {
      lines.push(`[${entry.timestamp}] [${entry.type.toUpperCase()}] ${entry.text}`);
    }

    const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `solv-shadow-${this.session.student_name.toLowerCase().replace(/\s+/g, '_')}-${Date.now()}.log`;
    link.click();
    URL.revokeObjectURL(url);

    this.isExported.set(true);
    setTimeout(() => this.isExported.set(false), 2000);
  }

  runDiagnostic(cmd: string): void {
    if (this.mode() === 'mirror') {
      this.setMode('tutor');
    }
    this.commandInput.set(cmd);
    this.sendCommand();
  }

  onInputKeyDown(event: KeyboardEvent): void {
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      const history = this.commandHistory();
      if (history.length === 0) return;
      
      const nextIdx = this.historyIndex() < history.length - 1 ? this.historyIndex() + 1 : this.historyIndex();
      this.historyIndex.set(nextIdx);
      this.commandInput.set(history[history.length - 1 - nextIdx] || '');
    } else if (event.key === 'ArrowDown') {
      event.preventDefault();
      const history = this.commandHistory();
      if (this.historyIndex() > 0) {
        const nextIdx = this.historyIndex() - 1;
        this.historyIndex.set(nextIdx);
        this.commandInput.set(history[history.length - 1 - nextIdx] || '');
      } else if (this.historyIndex() === 0) {
        this.historyIndex.set(-1);
        this.commandInput.set('');
      }
    }
  }

  sendCommand(): void {
    const cmd = this.commandInput().trim();
    if (!cmd || this.isExecuting()) return;

    // Push to history
    this.commandHistory.update(hist => [...hist, cmd]);
    this.historyIndex.set(-1);

    this.appendLogEntry(`$ ${cmd}`, 'prompt');
    this.commandInput.set('');

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'exec', command: cmd }));
      return;
    }

    // Fallback REST execution
    this.isExecuting.set(true);
    this.liveService.executeTutorCommand(this.session.container_id, cmd).subscribe({
      next: (res) => {
        this.isExecuting.set(false);
        if (res?.output) {
          this.parseAndAppendOutput(res.output);
        }
      },
      error: (err) => {
        this.isExecuting.set(false);
        this.appendLogEntry(`[Error al ejecutar]: ${err?.error?.message || 'Fallo de ejecución en contenedor'}`, 'error');
      }
    });
  }

  private scrollToBottom(): void {
    if (this.terminalContainer) {
      const el = this.terminalContainer.nativeElement;
      el.scrollTop = el.scrollHeight;
    }
  }

  onClose(): void {
    this.close.emit();
  }
}
