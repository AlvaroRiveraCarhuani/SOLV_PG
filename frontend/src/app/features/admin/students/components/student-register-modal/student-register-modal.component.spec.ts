import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { signal } from '@angular/core';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StudentRegisterModalComponent } from './student-register-modal.component';
import { RosterUploadComponent } from '@shared/components/roster-upload/roster-upload.component';
import { TenantService } from '@core/services/tenant.service';
import { CreateStudentDTO } from '../../../services/admin-students.service';
import { ParsedRosterResult } from '@core/models/admin.model';

const SAMPLE_ROSTER: ParsedRosterResult = {
  total_detected: 2,
  valid_emails: ['estudiante1@uab.edu.bo', 'estudiante2@uab.edu.bo'],
  duplicate_emails: [],
  invalid_emails: []
};

describe('StudentRegisterModalComponent', () => {
  let component: StudentRegisterModalComponent;
  let fixture: ComponentFixture<StudentRegisterModalComponent>;

  beforeEach(async () => {
    const configSignal = signal({ support_email: 'soporte@uab.edu.bo', base_domain: 'uab.edu.bo' });

    await TestBed.configureTestingModule({
      imports: [StudentRegisterModalComponent],
      providers: [
        provideHttpClientTesting(),
        { provide: TenantService, useValue: { config: configSignal } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentRegisterModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  function rosterUpload(): RosterUploadComponent {
    return fixture.debugElement.query(By.directive(RosterUploadComponent)).componentInstance as RosterUploadComponent;
  }

  it('inicia en la pestaña individual', () => {
    expect(component.currentTab()).toBe('individual');
  });

  it('emite studentCreated con datos limpios', () => {
    let payload: CreateStudentDTO | undefined;
    component.studentCreated.subscribe(p => { payload = p; });

    component.firstName.set('Ana Lucía');
    component.lastName.set('Flores Mamani');
    component.email.set('ana.flores@uab.edu.bo');

    component.onSubmitIndividual();

    expect(payload).toEqual({
      first_name: 'Ana Lucía',
      last_name: 'Flores Mamani',
      email: 'ana.flores@uab.edu.bo'
    });
  });

  it('no emite studentCreated con campos vacíos', () => {
    let emitted = false;
    component.studentCreated.subscribe(() => { emitted = true; });
    component.onSubmitIndividual();
    expect(emitted).toBe(false);
    expect(component.error()).toBeTruthy();
  });

  it('cambia a la pestaña de planilla y emite batchSubmit', () => {
    let emitted: string[] | undefined;
    component.batchSubmit.subscribe(p => { emitted = p; });

    component.setTab('batch');
    fixture.detectChanges();

    rosterUpload().parsedRoster.set(SAMPLE_ROSTER);
    component.confirmBatch();

    expect(emitted).toEqual(SAMPLE_ROSTER.valid_emails);
  });

  it('closed emite al cerrar', () => {
    let closed = false;
    component.closed.subscribe(() => { closed = true; });
    component.closed.emit();
    expect(closed).toBe(true);
  });
});
