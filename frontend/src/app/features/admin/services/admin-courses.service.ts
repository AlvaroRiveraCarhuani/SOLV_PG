import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, forkJoin, of } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';
import { TeacherItem } from '@core/models/admin.model';

export interface AcademicPeriod {
  id: string;
  tenant_id: string;
  name: string;
  code: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface CreateAcademicPeriodDTO {
  name: string;
  code: string;
  start_date: string;
  end_date: string;
}

export interface UpdateAcademicPeriodDTO {
  name?: string;
  code?: string;
  start_date?: string;
  end_date?: string;
  is_active?: boolean;
}

export interface CourseLoadMetric {
  id: string;
  course_name: string;
  teacher_name: string;
  active_students: number;
  hibernated_students: number;
  ram_used_mb: number;
}

export interface SubjectRaw {
  id: string;
  tenant_id: string;
  name: string;
  code: string;
  teacher_id?: string;
  teacher_name?: string;
  academic_period_id?: string;
  is_archived: boolean;
  classroom_course_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AdminCourseItem {
  id: string;
  name: string;
  code: string;
  teacher_id?: string;
  teacher_name: string;
  academic_period_id?: string;
  is_archived: boolean;
  active_students: number;
  hibernated_students: number;
  ram_used_mb: number;
  template_name?: string;
}

export interface CreateCoursePayload {
  name: string;
  code: string;
  teacher_id?: string;
  academic_period_id?: string;
  classroom_course_id?: string;
}

export interface ReassignCoursePayload {
  new_teacher_id: string;
  reason: string;
}

export interface DockerTemplateItem {
  id: string;
  name: string;
  display_name?: string;
  image: string;
  base_ram_mb?: number;
}

@Injectable({
  providedIn: 'root'
})
export class AdminCoursesService {
  private http = inject(HttpClient);

  // Estados reactivos con Signals
  periods = signal<AcademicPeriod[]>([]);
  activePeriod = signal<AcademicPeriod | null>(null);
  courses = signal<AdminCourseItem[]>([]);
  teachers = signal<TeacherItem[]>([]);
  templates = signal<DockerTemplateItem[]>([]);

  isLoading = signal<boolean>(false);
  error = signal<string | null>(null);

  /**
   * Carga inicial completa de periodos, cursos y metadatos
   */
  loadAll(): void {
    this.isLoading.set(true);
    this.error.set(null);

    forkJoin({
      periods: this.fetchPeriods(),
      subjects: this.fetchSubjects(),
      loadMetrics: this.fetchCoursesLoadMetrics(),
      teachers: this.fetchTeachers(),
      templates: this.fetchTemplates()
    }).subscribe({
      next: ({ periods, subjects, loadMetrics, teachers, templates }) => {
        const validPeriods = Array.isArray(periods) ? periods : [];
        const validSubjects = Array.isArray(subjects) ? subjects : [];
        const validMetrics = Array.isArray(loadMetrics) ? loadMetrics : [];

        this.periods.set(validPeriods);
        
        // Seleccionar periodo activo actual si existe o el primero
        const currentActive = validPeriods.find(p => p.is_active) || validPeriods[0] || null;
        this.activePeriod.set(currentActive);

        // Mapear métricas en vivo sobre la lista de materias
        const loadMap = new Map<string, CourseLoadMetric>();
        for (const metric of validMetrics) {
          if (metric && metric.id) {
            loadMap.set(metric.id, metric);
          }
        }

        const enrichedCourses: AdminCourseItem[] = validSubjects.map(s => {
          const metric = loadMap.get(s.id);
          return {
            id: s.id,
            name: s.name,
            code: s.code,
            teacher_id: s.teacher_id,
            teacher_name: s.teacher_name || metric?.teacher_name || 'Sin asignar',
            academic_period_id: s.academic_period_id,
            is_archived: !!s.is_archived,
            active_students: metric ? metric.active_students : 0,
            hibernated_students: metric ? metric.hibernated_students : 0,
            ram_used_mb: metric ? metric.ram_used_mb : 0,
            template_name: 'Linux / Docker Base'
          };
        });

        this.courses.set(enrichedCourses);
        this.teachers.set(Array.isArray(teachers) ? teachers : []);
        this.templates.set(Array.isArray(templates) ? templates : []);
        this.isLoading.set(false);
      },
      error: () => {
        this.error.set('No se pudo conectar con el servicio de cursos y periodos.');
        this.isLoading.set(false);
      }
    });
  }

  // ---------------------------------------------------------------------------
  // Periodos Académicos
  // ---------------------------------------------------------------------------
  fetchPeriods(): Observable<AcademicPeriod[]> {
    return this.http.get<{ data?: AcademicPeriod[] } | AcademicPeriod[]>('/api/v1/admin/academic-periods').pipe(
      map(res => {
        if (Array.isArray(res)) return res;
        return res.data || [];
      }),
      catchError(() => of([]))
    );
  }

  createPeriod(dto: CreateAcademicPeriodDTO): Observable<AcademicPeriod> {
    return this.http.post<{ data?: AcademicPeriod } | AcademicPeriod>('/api/v1/admin/academic-periods', dto).pipe(
      map(res => {
        const period = (res as { data?: AcademicPeriod }).data || (res as AcademicPeriod);
        this.periods.update(prev => [...prev, period]);
        return period;
      })
    );
  }

  updatePeriod(id: string, dto: UpdateAcademicPeriodDTO): Observable<AcademicPeriod> {
    return this.http.put<{ data?: AcademicPeriod } | AcademicPeriod>(`/api/v1/admin/academic-periods/${id}`, dto).pipe(
      map(res => {
        const updated = (res as { data?: AcademicPeriod }).data || (res as AcademicPeriod);
        this.periods.update(prev => prev.map(p => p.id === id ? updated : p));
        if (this.activePeriod()?.id === id) {
          this.activePeriod.set(updated);
        }
        return updated;
      })
    );
  }

  deletePeriod(id: string): Observable<void> {
    return this.http.delete<void>(`/api/v1/admin/academic-periods/${id}`).pipe(
      tap(() => {
        this.periods.update(prev => prev.filter(p => p.id !== id));
        if (this.activePeriod()?.id === id) {
          this.activePeriod.set(this.periods()[0] || null);
        }
      })
    );
  }

  setActivePeriod(period: AcademicPeriod | null): void {
    this.activePeriod.set(period);
  }

  // ---------------------------------------------------------------------------
  // Materias / Cursos
  // ---------------------------------------------------------------------------
  fetchSubjects(): Observable<SubjectRaw[]> {
    return this.http.get<any>('/api/v1/subjects').pipe(
      map(res => {
        if (Array.isArray(res)) return res;
        if (res && Array.isArray(res.data)) return res.data;
        if (res && Array.isArray(res.subjects)) return res.subjects;
        return [];
      }),
      catchError(() => of([]))
    );
  }

  fetchCoursesLoadMetrics(): Observable<CourseLoadMetric[]> {
    return this.http.get<any>('/api/v1/admin/dashboard/courses-load').pipe(
      map(res => {
        if (Array.isArray(res)) return res;
        if (res && Array.isArray(res.data)) return res.data;
        return [];
      }),
      catchError(() => of([]))
    );
  }

  fetchTeachers(): Observable<TeacherItem[]> {
    return this.http.get<{ teachers?: TeacherItem[]; data?: TeacherItem[] }>('/api/v1/teachers').pipe(
      map(res => res.teachers || res.data || []),
      catchError(() => of([]))
    );
  }

  fetchTemplates(): Observable<DockerTemplateItem[]> {
    return this.http.get<{ templates?: DockerTemplateItem[]; data?: DockerTemplateItem[] } | DockerTemplateItem[]>('/api/v1/admin/templates').pipe(
      map(res => {
        if (Array.isArray(res)) return res;
        return (res as { templates?: DockerTemplateItem[] }).templates || (res as { data?: DockerTemplateItem[] }).data || [];
      }),
      catchError(() => of([
        { id: 'c-lang', name: 'c-lang-base', display_name: 'C / Linux Kernel', image: 'solv-c-base:latest', base_ram_mb: 512 },
        { id: 'python', name: 'python-base', display_name: 'Python 3.12 / Data', image: 'solv-python:latest', base_ram_mb: 512 },
        { id: 'java', name: 'java-base', display_name: 'Java 21 LTS', image: 'solv-java:latest', base_ram_mb: 1024 }
      ]))
    );
  }

  createCourse(payload: CreateCoursePayload): Observable<AdminCourseItem> {
    return this.http.post<SubjectRaw>('/api/v1/subjects', payload).pipe(
      map(raw => {
        const teacher = this.teachers().find(t => t.id === payload.teacher_id);
        const newCourse: AdminCourseItem = {
          id: raw.id,
          name: raw.name,
          code: raw.code,
          teacher_id: raw.teacher_id,
          teacher_name: teacher?.full_name || 'Sin asignar',
          academic_period_id: raw.academic_period_id,
          is_archived: false,
          active_students: 0,
          hibernated_students: 0,
          ram_used_mb: 0,
          template_name: 'Linux / Docker Base'
        };
        this.courses.update(prev => [newCourse, ...prev]);
        return newCourse;
      })
    );
  }

  reassignCourseTeacher(courseId: string, payload: ReassignCoursePayload): Observable<void> {
    return this.http.post<void>(`/api/v1/admin/courses/${courseId}/reassign`, payload).pipe(
      tap(() => {
        const teacher = this.teachers().find(t => t.id === payload.new_teacher_id);
        const teacherName = teacher ? teacher.full_name : 'Docente Asignado';
        this.courses.update(prev => prev.map(c => {
          if (c.id === courseId) {
            return { ...c, teacher_id: payload.new_teacher_id, teacher_name: teacherName };
          }
          return c;
        }));
      })
    );
  }

  toggleArchiveCourse(courseId: string, isArchived: boolean): Observable<void> {
    return this.http.put<void>(`/api/v1/subjects/${courseId}/archive`, { is_archived: isArchived }).pipe(
      tap(() => {
        this.courses.update(prev => prev.map(c => {
          if (c.id === courseId) {
            return { ...c, is_archived: isArchived };
          }
          return c;
        }));
      })
    );
  }

  updateCourse(courseId: string, payload: { name: string; code: string }): Observable<void> {
    return this.http.put<void>(`/api/v1/subjects/${courseId}`, payload).pipe(
      tap(() => {
        this.courses.update(prev => prev.map(c => {
          if (c.id === courseId) {
            return { ...c, name: payload.name, code: payload.code };
          }
          return c;
        }));
      })
    );
  }
}
