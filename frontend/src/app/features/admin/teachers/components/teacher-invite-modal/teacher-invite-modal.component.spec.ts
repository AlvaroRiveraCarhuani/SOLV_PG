import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { TeacherInviteModalComponent } from './teacher-invite-modal.component';
import { TenantService } from '@core/services/tenant.service';
import { AdminTeachersService } from '@features/admin/services/admin-teachers.service';
import {
  DirectTeacherPayload,
  BatchDirectTeacherPayload,
  ParsedRosterResult
} from '@core/models/admin.model';

interface MockTenantService {
  config: ReturnType<typeof signal<{ support_email: string; base_domain: string }>>;
}

interface MockAdminTeachersService {
  parseRosterFile: (file?: File) => import('rxjs').Observable<ParsedRosterResult>;
}

describe('TeacherInviteModalComponent', () => {
  let component: TeacherInviteModalComponent;
  let mockTenantService: MockTenantService;
  let mockTeachersService: MockAdminTeachersService;

  beforeEach(async () => {
    const configSignal = signal({ support_email: 'soporte@uab.edu.bo', base_domain: 'uab.edu.bo' });
    mockTenantService = { config: configSignal };
    mockTeachersService = {
      parseRosterFile: () => of({
        total_detected: 2,
        valid_emails: ['docente1@uab.edu.bo', 'docente2@uab.edu.bo'],
        duplicate_emails: [],
        invalid_emails: []
      } as ParsedRosterResult)
    };

    await TestBed.configureTestingModule({
      imports: [TeacherInviteModalComponent],
      providers: [
        { provide: TenantService, useValue: mockTenantService },
        { provide: AdminTeachersService, useValue: mockTeachersService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(TeacherInviteModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente en la pestaña individual por defecto', () => {
    expect(component).toBeTruthy();
    expect(component.currentTab()).toBe('individual');
  });

  it('institutionDomain se extrae del support_email del tenant', () => {
    expect(component.institutionDomain()).toBe('uab.edu.bo');
  });

  it('isValidEmail valida correctamente emails', () => {
    component.email.set('');
    expect(component.isValidEmail()).toBe(false);

    component.email.set('a@b.c');
    expect(component.isValidEmail()).toBe(false);

    component.email.set('docente@uab.edu.bo');
    expect(component.isValidEmail()).toBe(true);
  });

  it('onSubmitIndividual emite directSubmit con datos limpios', () => {
    let payload: DirectTeacherPayload | undefined;
    component.directSubmit.subscribe(p => { payload = p; });

    component.email.set('alvaro.rivera@uab.edu.bo');
    component.firstName.set('Alvaro');
    component.lastName.set('Rivera');
    component.sendNotification.set(true);

    component.onSubmitIndividual();

    expect(payload).toEqual({
      email: 'alvaro.rivera@uab.edu.bo',
      first_name: 'Alvaro',
      last_name: 'Rivera',
      role_type: 'titular',
      send_notification: true
    });
    expect(component.isSubmitting()).toBe(true);
  });

  it('isExternalDomain detecta dominios no institucionales', () => {
    component.email.set('prof@uab.edu.bo');
    expect(component.isExternalDomain()).toBe(false);

    component.email.set('prof@gmail.com');
    expect(component.isExternalDomain()).toBe(true);
  });

  it('permite cambiar a pestaña de importar planilla y procesar archivo', () => {
    component.setTab('batch');
    expect(component.currentTab()).toBe('batch');

    const dummyFile = new File(['email\ndocente1@uab.edu.bo\ndocente2@uab.edu.bo'], 'roster.csv', { type: 'text/csv' });
    component.processFile(dummyFile);

    expect(component.selectedFile()).toBe(dummyFile);
    expect(component.parsedRoster()?.total_detected).toBe(2);
    expect(component.parsedRoster()?.valid_emails.length).toBe(2);
  });

  it('onSubmitBatch emite batchSubmit con correos parseados', () => {
    let batchPayload: BatchDirectTeacherPayload | undefined;
    component.batchSubmit.subscribe(p => { batchPayload = p; });

    component.parsedRoster.set({
      total_detected: 2,
      valid_emails: ['d1@uab.edu.bo', 'd2@uab.edu.bo'],
      duplicate_emails: [],
      invalid_emails: []
    });
    component.sendNotificationBatch.set(true);

    component.onSubmitBatch();

    expect(batchPayload).toEqual({
      emails: ['d1@uab.edu.bo', 'd2@uab.edu.bo'],
      send_notification: true
    });
    expect(component.isSubmittingBatch()).toBe(true);
  });

  it('closed emite evento al ser llamado', () => {
    let closed = false;
    component.closed.subscribe(() => { closed = true; });
    component.closed.emit();
    expect(closed).toBe(true);
  });
});
