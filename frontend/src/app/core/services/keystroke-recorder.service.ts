import { Injectable, inject } from '@angular/core';
import { StudentService } from './student.service';

export interface KeystrokeBufferItem {
  timestamp_ms: number;
  event_type: 'insert' | 'delete' | 'paste';
  position: number;
  content: string;
  paste_source_detected: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class KeystrokeRecorderService {
  private studentService = inject(StudentService);
  private buffer: KeystrokeBufferItem[] = [];
  private sessionStartTime: number = Date.now();
  private activeSubmissionId: string | null = null;
  private syncIntervalId: ReturnType<typeof setInterval> | null = null;

  startSession(submissionId: string): void {
    this.activeSubmissionId = submissionId;
    this.sessionStartTime = Date.now();
    this.buffer = [];
    this.stopSyncInterval();
    this.syncIntervalId = setInterval(() => this.flush(), 30000);
  }

  recordEvent(eventType: 'insert' | 'delete' | 'paste', position: number, content: string): void {
    const isPaste = eventType === 'paste' || content.length > 50;
    const event: KeystrokeBufferItem = {
      timestamp_ms: Date.now() - this.sessionStartTime,
      event_type: eventType,
      position,
      content,
      paste_source_detected: isPaste
    };
    this.buffer.push(event);
  }

  async flush(): Promise<void> {
    if (!this.activeSubmissionId || this.buffer.length === 0) return;
    const eventsToSend = [...this.buffer];
    this.buffer = [];
    await this.studentService.saveKeystrokeEvents(this.activeSubmissionId, eventsToSend);
  }

  stopSession(): void {
    this.flush();
    this.stopSyncInterval();
    this.activeSubmissionId = null;
  }

  private stopSyncInterval(): void {
    if (this.syncIntervalId) {
      clearInterval(this.syncIntervalId);
      this.syncIntervalId = null;
    }
  }
}
