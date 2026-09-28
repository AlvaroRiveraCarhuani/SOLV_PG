import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MachineDataDirective } from './machine-data.directive';

@Component({
  standalone: true,
  imports: [MachineDataDirective],
  template: '<span machineData>82%</span>'
})
class MachineDataHostComponent {}

describe('MachineDataDirective', () => {
  let fixture: ComponentFixture<MachineDataHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MachineDataHostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(MachineDataHostComponent);
    fixture.detectChanges();
  });

  it('marks the existing element as machine data without adding a wrapper', () => {
    const host = fixture.nativeElement as HTMLElement;
    const value = host.querySelector('span');

    expect(value?.classList.contains('font-mono')).toBe(true);
    expect(host.children).toHaveLength(1);
    expect(value?.children).toHaveLength(0);
  });
});