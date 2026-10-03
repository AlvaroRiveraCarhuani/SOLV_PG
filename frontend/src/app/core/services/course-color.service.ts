import { Injectable, signal } from '@angular/core';

export interface CourseColorPreset {
  id: string;
  name: string;
  hex: string;
}

export interface CourseThemeStyle {
  accent: string;
  accentBg: string;
  accentBorder: string;
  accentText: string;
}

export const CURATED_COURSE_PALETTE: CourseColorPreset[] = [
  { id: 'indigo', name: 'Índigo', hex: '#6366F1' },
  { id: 'emerald', name: 'Esmeralda', hex: '#10B981' },
  { id: 'purple', name: 'Púrpura', hex: '#8B5CF6' },
  { id: 'amber', name: 'Ámbar', hex: '#F59E0B' },
  { id: 'rose', name: 'Rosa Carmín', hex: '#F43F5E' },
  { id: 'cyan', name: 'Cian', hex: '#06B6D4' },
  { id: 'teal', name: 'Verde Azulado', hex: '#14B8A6' },
  { id: 'slate', name: 'Pizarra', hex: '#64748B' }
];

@Injectable({
  providedIn: 'root'
})
export class CourseColorService {
  private readonly storagePrefix = 'solv_course_color_';
  private customColors = signal<Record<string, string>>({});

  constructor() {
    this.loadFromStorage();
  }

  getPresets(): CourseColorPreset[] {
    return CURATED_COURSE_PALETTE;
  }

  /**
   * Retrieves the assigned or deterministic color for a given course ID and optional code.
   */
  getCourseColor(courseId: string, courseCode?: string): string {
    const custom = this.customColors()[courseId];
    if (custom) {
      return custom;
    }
    return this.getDeterministicColor(courseCode || courseId || 'solv');
  }

  /**
   * Computes safe UI styling variables for the course accent color.
   */
  getCourseThemeStyle(hexColor: string): CourseThemeStyle {
    const cleanHex = this.normalizeHex(hexColor);
    const { r, g, b } = this.hexToRgb(cleanHex);

    return {
      accent: cleanHex,
      accentBg: `rgba(${r}, ${g}, ${b}, 0.12)`,
      accentBorder: `rgba(${r}, ${g}, ${b}, 0.35)`,
      accentText: cleanHex
    };
  }

  /**
   * Persists a custom or preset color for a course.
   */
  setCourseColor(courseId: string, colorHex: string): void {
    const cleanHex = this.normalizeHex(colorHex);
    this.customColors.update(prev => ({
      ...prev,
      [courseId]: cleanHex
    }));
    this.saveToStorage();
  }

  /**
   * Resets course color to its deterministic default.
   */
  resetCourseColor(courseId: string): void {
    this.customColors.update(prev => {
      const next = { ...prev };
      delete next[courseId];
      return next;
    });
    this.saveToStorage();
  }

  /**
   * Generates a deterministic curated preset color from a string key.
   */
  getDeterministicColor(key: string): string {
    let hash = 0;
    for (let i = 0; i < key.length; i++) {
      hash = key.charCodeAt(i) + ((hash << 5) - hash);
    }
    const index = Math.abs(hash) % CURATED_COURSE_PALETTE.length;
    return CURATED_COURSE_PALETTE[index].hex;
  }

  private normalizeHex(hex: string): string {
    if (!hex) return CURATED_COURSE_PALETTE[0].hex;
    let clean = hex.trim();
    if (!clean.startsWith('#')) {
      clean = '#' + clean;
    }
    if (/^#[0-9A-Fa-f]{6}$/.test(clean)) {
      return clean.toUpperCase();
    }
    if (/^#[0-9A-Fa-f]{3}$/.test(clean)) {
      return (
        '#' +
        clean[1] + clean[1] +
        clean[2] + clean[2] +
        clean[3] + clean[3]
      ).toUpperCase();
    }
    return CURATED_COURSE_PALETTE[0].hex;
  }

  private hexToRgb(hex: string): { r: number; g: number; b: number } {
    const clean = hex.replace('#', '');
    const bigint = parseInt(clean, 16);
    return {
      r: (bigint >> 16) & 255,
      g: (bigint >> 8) & 255,
      b: bigint & 255
    };
  }

  private loadFromStorage(): void {
    try {
      const raw = localStorage.getItem(this.storagePrefix + 'map');
      if (raw) {
        const parsed = JSON.parse(raw);
        if (typeof parsed === 'object' && parsed !== null) {
          this.customColors.set(parsed);
        }
      }
    } catch {
      // Ignored in non-browser or corrupted storage
    }
  }

  private saveToStorage(): void {
    try {
      localStorage.setItem(this.storagePrefix + 'map', JSON.stringify(this.customColors()));
    } catch {
      // Ignored in non-browser
    }
  }
}
