import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  OnInit, 
  OnDestroy, 
  inject, 
  signal, 
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
  LucideSend
} from '@lucide/angular';
import { LiveWorkspaceSession } from '../../models/teacher.models';
import { TeacherLiveService } from '../../services/teacher-live.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';

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

  terminalLogs = signal<string[]>([]);
  isConnected = signal<boolean>(false);
  autoScroll = signal<boolean>(true);
  isFullscreen = signal<boolean>(false);
  isCopied = signal<boolean>(false);
  commandInput = signal<string>('');
  isExecuting = signal<boolean>(false);

  private ws?: WebSocket;
  private pingInterval?: ReturnType<typeof setInterval>;

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.onClose();
  }

  ngOnInit(): void {
    this.connectWebSocket();
  }

  ngOnDestroy(): void {
    this.disconnectWebSocket();
  }

  private connectWebSocket(): void {
    if (!this.session.container_id) {
      this.appendLog('[SOLV Shadow Mode] Sin contenedor asignado o en proceso de inicio.');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/teacher/live-sessions/${this.session.container_id}/terminal`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.isConnected.set(true);
        this.appendLog(`[SOLV Shadow Mode] Conexión establecida con el contenedor: ${this.session.container_id}`);
        this.appendLog(`[Terminal Mirror] Estudiante: ${this.session.student_name} (${this.session.student_email})`);
        this.appendLog('─'.repeat(60));

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
            this.appendLog(data.buffer);
          } else if (data.type === 'stdout' && data.output) {
            this.appendLog(data.output);
          } else if (data.type === 'error') {
            this.appendLog(`[Error]: ${data.error}`);
          }
        } catch {
          this.appendLog(event.data);
        }
      };

      this.ws.onclose = () => {
        this.isConnected.set(false);
        this.appendLog('[SOLV Shadow Mode] Conexión finalizada.');
      };

      this.ws.onerror = () => {
        this.isConnected.set(false);
        this.appendLog('[SOLV Shadow Mode] Fallback: Simulador de terminal activo (modo seguro).');
      };
    } catch {
      this.isConnected.set(false);
      this.appendLog('[SOLV Shadow Mode] Simulación de terminal activa.');
    }
  }

  private disconnectWebSocket(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
    }
    if (this.ws) {
      this.ws.close();
      this.ws = undefined;
    }
  }

  appendLog(line: string): void {
    this.terminalLogs.update(logs => [...logs, line]);
    if (this.autoScroll()) {
      setTimeout(() => this.scrollToBottom(), 30);
    }
  }

  clearTerminal(): void {
    this.terminalLogs.set([]);
  }

  toggleAutoScroll(): void {
    this.autoScroll.update(val => !val);
  }

  toggleFullscreen(): void {
    this.isFullscreen.update(val => !val);
  }

  copyLogs(): void {
    const text = this.terminalLogs().join('\n');
    navigator.clipboard.writeText(text).then(() => {
      this.isCopied.set(true);
      setTimeout(() => this.isCopied.set(false), 2000);
    });
  }

  runDiagnostic(cmd: string): void {
    this.commandInput.set(cmd);
    this.sendCommand();
  }

  sendCommand(): void {
    const cmd = this.commandInput().trim();
    if (!cmd || this.isExecuting()) return;

    this.appendLog(`\n$ ${cmd}`);
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
        if (res.output) {
          this.appendLog(res.output);
        }
      },
      error: (err) => {
        this.isExecuting.set(false);
        this.appendLog(`[Error al ejecutar]: ${err?.error?.message || 'Fallo de ejecución'}`);
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
