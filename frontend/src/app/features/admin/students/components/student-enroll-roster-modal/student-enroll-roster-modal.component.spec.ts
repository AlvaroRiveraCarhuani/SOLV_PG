import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StudentEnrollRosterModalComponent } from './student-enroll-roster-modal.component';
import { RosterUploadComponent } from '@shared/components/roster-upload/roster-upload.component';
import { ParsedRosterResult } from '@core/models/admin.model';

const SAMPLE_ROSTER: ParsedRosterResult = {
  total_detected: 2,
  valid_emails: ['estudiante1@uab.edu.bo', 'estudiante2@uab.edu.bo'],
  duplicate_emails: [],
  invalid_emails: []
};

describe('StudentEnrollRosterModalComponent', () => {
  let component: StudentEnrollRosterModalComponent;
  let fixture: ComponentFixture<StudentEnrollRosterModalComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StudentEnrollRosterModalComponent],
      providers: [provideHttpClientTesting()]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentEnrollRosterModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('subjects', [
      { id: 's1', code: 'SIS-211', name: 'Estructuras de Datos' },
      { id: 's2', code: 'SIS-312', name: 'Sistemas Operativos' }
    ]);
    fixture.detectChanges();
  });

  function rosterUpload(): RosterUploadComponent {
    return fixture.debugElement.query(By.directive(RosterUploadComponent)).componentInstance as RosterUploadComponent;
  }

  it('no emite rosterSubmit sin materia seleccionada', () => {
    let emitted: unknown;
    component.rosterSubmit.subscribe(p => { emitted = p; });
    rosterUpload().parsedRoster.set(SAMPLE_ROSTER);
    component.confirm();
    expect(emitted).toBeUndefined();
  });

  it('no emite rosterSubmit sin correos válidos', () => {
    let emitted: unknown;
    component.rosterSubmit.subscribe(p => { emitted = p; });
    component.selectedSubjectId.set('s1');
    component.confirm();
    expect(emitted).toBeUndefined();
  });

  it('emite rosterSubmit con materia y correos', () => {
    let emitted: { subjectId: string; emails: string[] } | undefined;
    component.rosterSubmit.subscribe(p => { emitted = p; });

    component.selectedSubjectId.set('s1');
    rosterUpload().parsedRoster.set(SAMPLE_ROSTER);
    component.confirm();

    expect(emitted).toEqual({
      subjectId: 's1',
      emails: SAMPLE_ROSTER.valid_emails
    });
  });
});
