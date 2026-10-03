import { Component, ChangeDetectionStrategy, input, signal, inject, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { 
  LucideBookOpen, 
  LucideArrowRight, 
  LucidePalette 
} from '@lucide/angular';
import { TeacherCourseSummary } from '../../../models/teacher.models';
import { CourseColorService, CourseThemeStyle } from '@core/services/course-color.service';
import { CourseColorPickerComponent } from '@shared/components/course-color-picker/course-color-picker.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ViewSwitcherComponent, ViewMode } from '@shared/components/view-switcher/view-switcher.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';

@Component({
  selector: 'dashboard-courses-widget',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideBookOpen,
    LucideArrowRight,
    LucidePalette,
    CourseColorPickerComponent,
    DismissibleDirective,
    MachineDataDirective,
    ViewSwitcherComponent,
    SkeletonLoaderComponent
  ],
  templateUrl: './dashboard-courses-widget.component.html',
  styleUrl: './dashboard-courses-widget.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DashboardCoursesWidgetComponent {
  private courseColorService = inject(CourseColorService);

  readonly courses = input<TeacherCourseSummary[]>([]);
  readonly isLoading = input<boolean>(false);

  readonly viewMode = signal<ViewMode>((localStorage.getItem('solv_teacher_view_mode') as ViewMode) || 'cards');
  readonly activeColorPickerCourseId = signal<string | null>(null);

  constructor() {
    effect(() => {
      const mode = this.viewMode();
      try {
        localStorage.setItem('solv_teacher_view_mode', mode);
      } catch {}
    });
  }

  getCourseColor(courseId: string, courseCode: string): string {
    return this.courseColorService.getCourseColor(courseId, courseCode);
  }

  getCourseThemeStyle(courseId: string, courseCode: string): CourseThemeStyle {
    const color = this.getCourseColor(courseId, courseCode);
    return this.courseColorService.getCourseThemeStyle(color);
  }

  toggleCourseColorPicker(courseId: string, event: Event): void {
    event.stopPropagation();
    if (this.activeColorPickerCourseId() === courseId) {
      this.activeColorPickerCourseId.set(null);
    } else {
      this.activeColorPickerCourseId.set(courseId);
    }
  }

  onCourseColorChanged(courseId: string, colorHex: string): void {
    this.courseColorService.setCourseColor(courseId, colorHex);
    this.activeColorPickerCourseId.set(null);
  }
}
