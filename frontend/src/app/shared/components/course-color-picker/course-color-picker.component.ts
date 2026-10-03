import { Component, EventEmitter, Input, Output, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CourseColorService, CourseColorPreset, CURATED_COURSE_PALETTE } from '@core/services/course-color.service';
import { LucideCheck, LucidePalette, LucideRotateCcw } from '@lucide/angular';

@Component({
  selector: 'course-color-picker',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCheck,
    LucidePalette,
    LucideRotateCcw
  ],
  templateUrl: './course-color-picker.component.html',
  styleUrls: ['./course-color-picker.component.scss']
})
export class CourseColorPickerComponent implements OnInit {
  private courseColorService = inject(CourseColorService);

  @Input() courseId?: string;
  @Input() courseCode?: string;
  @Input() selectedColor?: string;

  @Output() colorChange = new EventEmitter<string>();

  presets: CourseColorPreset[] = CURATED_COURSE_PALETTE;
  currentColor = signal<string>(CURATED_COURSE_PALETTE[0].hex);
  isCustom = computed<boolean>(() => {
    return !this.presets.some(p => p.hex.toUpperCase() === this.currentColor().toUpperCase());
  });

  ngOnInit(): void {
    if (this.selectedColor) {
      this.currentColor.set(this.selectedColor.toUpperCase());
    } else if (this.courseId) {
      this.currentColor.set(this.courseColorService.getCourseColor(this.courseId, this.courseCode));
    }
  }

  selectPreset(hex: string): void {
    this.applyColor(hex);
  }

  onCustomColorInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input && input.value) {
      this.applyColor(input.value);
    }
  }

  resetToDefault(): void {
    if (this.courseId) {
      this.courseColorService.resetCourseColor(this.courseId);
      const def = this.courseColorService.getCourseColor(this.courseId, this.courseCode);
      this.applyColor(def);
    } else {
      this.applyColor(CURATED_COURSE_PALETTE[0].hex);
    }
  }

  private applyColor(hex: string): void {
    const normalized = hex.toUpperCase();
    this.currentColor.set(normalized);
    if (this.courseId) {
      this.courseColorService.setCourseColor(this.courseId, normalized);
    }
    this.colorChange.emit(normalized);
  }
}
