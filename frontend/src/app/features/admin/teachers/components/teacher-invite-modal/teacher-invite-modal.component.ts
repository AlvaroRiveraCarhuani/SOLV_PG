import { Component, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TeacherInvitationPayload, TeacherRoleType } from '@core/models/admin.model';
import { LucideX, LucideUserPlus, LucideMail, LucideShieldCheck } from '@lucide/angular';

@Component({
  selector: 'solv-teacher-invite-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideX, LucideUserPlus, LucideMail, LucideShieldCheck],
  template: `
    <div class="modal-backdrop" (click)="close.emit()">
      <div class="modal-dialog" (click)="$event.stopPropagation()">
        <!-- Header -->
        <div class="modal-header">
          <div class="header-left">
            <div class="icon-circle">
              <svg lucideUserPlus class="icon"></svg>
            </div>
            <div>
              <h2 class="modal-title">Invitar Nuevo Docente a la Institución</h2>
              <p class="modal-subtitle">Emisión de token transaccional de un solo uso con vigencia de 72h (ADR-025).</p>
            </div>
          </div>
          <button class="btn-close" (click)="close.emit()" title="Cerrar modal">
            <svg lucideX class="icon"></svg>
          </button>
        </div>

        <!-- Body Form -->
        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">Correo Institucional del Profesor *</label>
            <div class="input-with-icon">
              <svg lucideMail class="field-icon"></svg>
              <input 
                type="email" 
                placeholder="ejemplo@uab.edu.bo"
                [ngModel]="email()" 
                (ngModelChange)="email.set($event)"
                class="form-input"
                autofocus
              />
            </div>
            <span class="field-hint">El docente completará su alta mediante Google Workspace SSO con esta cuenta.</span>
          </div>

          <div class="form-group">
            <label class="form-label">Rol Institucional en la Cátedra</label>
            <div class="radio-group">
              <label class="radio-label">
                <input 
                  type="radio" 
                  name="roleType" 
                  value="titular" 
                  [checked]="roleType() === 'titular'"
                  (change)="roleType.set('titular')" 
                />
                <span class="radio-text">Profesor Titular (Gestión de laboratorios y evaluaciones)</span>
              </label>
              <label class="radio-label">
                <input 
                  type="radio" 
                  name="roleType" 
                  value="auxiliar" 
                  [checked]="roleType() === 'auxiliar'"
                  (change)="roleType.set('auxiliar')" 
                />
                <span class="radio-text">Auxiliar / Ayudante (Soporte y monitoreo de sesiones)</span>
              </label>
            </div>
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input 
                type="checkbox" 
                [checked]="sendEmail()" 
                (change)="sendEmail.set(!sendEmail())" 
              />
              <span class="checkbox-text">
                Enviar enlace profundo por correo electrónico con token transaccional (TTL 72 horas)
              </span>
            </label>
          </div>
        </div>

        <!-- Footer -->
        <div class="modal-footer">
          <button class="btn-cancel" (click)="close.emit()">
            <span>Cancelar</span>
          </button>
          <button 
            class="btn-submit" 
            [disabled]="!isValidEmail()" 
            (click)="onSubmit()">
            <svg lucideShieldCheck class="btn-icon"></svg>
            <span>Generar Token e Invitar</span>
          </button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .modal-backdrop {
      position: fixed;
      top: 0;
      left: 0;
      width: 100vw;
      height: 100vh;
      background-color: rgba(15, 23, 42, 0.5);
      backdrop-filter: blur(2px);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 1000;
      padding: var(--space-4, 16px);
    }

    .modal-dialog {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1);
      width: 100%;
      max-width: 580px;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      animation: modalEnter 150ms ease-out;
    }

    @keyframes modalEnter {
      from { opacity: 0; transform: scale(0.98); }
      to { opacity: 1; transform: scale(1); }
    }

    .modal-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      padding: 16px 20px;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
    }

    .header-left {
      display: flex;
      gap: 12px;
      align-items: flex-start;
    }

    .icon-circle {
      width: 36px;
      height: 36px;
      border-radius: 50%;
      background-color: var(--tenant-primary-subtle, rgba(37, 99, 235, 0.08));
      color: var(--tenant-primary, #2563EB);
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;

      .icon {
        width: 18px;
        height: 18px;
      }
    }

    .modal-title {
      font-size: 15px;
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .modal-subtitle {
      font-size: 12px;
      color: var(--text-muted, #64748B);
      margin: 2px 0 0 0;
    }

    .btn-close {
      background: transparent;
      border: none;
      cursor: pointer;
      padding: 4px;
      border-radius: 4px;
      color: var(--text-muted, #94A3B8);

      &:hover {
        background-color: #F1F5F9;
        color: #0F172A;
      }

      .icon {
        width: 18px;
        height: 18px;
      }
    }

    .modal-body {
      padding: 20px;
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    .form-group {
      display: flex;
      flex-direction: column;
      gap: 6px;
    }

    .form-label {
      font-size: 12px;
      font-weight: 600;
      color: var(--text-primary, #1E293B);
    }

    .input-with-icon {
      position: relative;
    }

    .field-icon {
      position: absolute;
      left: 10px;
      top: 50%;
      transform: translateY(-50%);
      width: 15px;
      height: 15px;
      color: var(--text-muted, #94A3B8);
    }

    .form-input {
      width: 100%;
      padding: 8px 12px 8px 34px;
      font-size: 13px;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      outline: none;

      &:focus {
        border-color: var(--tenant-primary, #2563EB);
      }
    }

    .field-hint {
      font-size: 11px;
      color: var(--text-muted, #64748B);
    }

    .radio-group {
      display: flex;
      flex-direction: column;
      gap: 8px;
      background-color: #F8FAFC;
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-md, 6px);
      padding: 10px 14px;
    }

    .radio-label {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: var(--text-secondary, #334155);
      cursor: pointer;
    }

    .checkbox-group {
      margin-top: 4px;
    }

    .checkbox-label {
      display: flex;
      align-items: flex-start;
      gap: 8px;
      font-size: 12px;
      color: var(--text-secondary, #475569);
      cursor: pointer;
    }

    .modal-footer {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      padding: 14px 20px;
      border-top: 1px solid var(--border-subtle, #E2E8F0);
      background-color: #F8FAFC;
    }

    .btn-cancel {
      padding: 7px 14px;
      background-color: #FFFFFF;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: 13px;
      font-weight: 500;
      color: var(--text-secondary, #475569);
      cursor: pointer;

      &:hover {
        background-color: #F1F5F9;
      }
    }

    .btn-submit {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 7px 16px;
      background-color: var(--tenant-primary, #2563EB);
      border: 1px solid transparent;
      border-radius: var(--radius-md, 6px);
      font-size: 13px;
      font-weight: 600;
      color: #FFFFFF;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        opacity: 0.92;
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }

      .btn-icon {
        width: 14px;
        height: 14px;
      }
    }
  `]
})
export class TeacherInviteModalComponent {
  close = output<void>();
  submit = output<TeacherInvitationPayload>();

  email = signal<string>('');
  roleType = signal<TeacherRoleType>('titular');
  sendEmail = signal<boolean>(true);

  isValidEmail(): boolean {
    const e = this.email().trim();
    return e.length > 5 && e.includes('@') && e.includes('.');
  }

  onSubmit(): void {
    if (!this.isValidEmail()) return;
    this.submit.emit({
      email: this.email().trim(),
      role: 'teacher',
      role_type: this.roleType(),
      send_email: this.sendEmail()
    });
  }
}
