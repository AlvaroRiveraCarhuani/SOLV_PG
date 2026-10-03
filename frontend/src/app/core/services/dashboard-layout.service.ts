import { Injectable } from '@angular/core';

export type WidgetColSpan = 4 | 6 | 8 | 12;

export interface WidgetLayoutItem {
  id: string;
  title: string;
  colSpan: WidgetColSpan;
  visible: boolean;
  minColSpan?: WidgetColSpan;
}

@Injectable({
  providedIn: 'root'
})
export class DashboardLayoutService {
  loadLayout(storageKey: string, defaults: WidgetLayoutItem[]): WidgetLayoutItem[] {
    try {
      if (typeof window === 'undefined' || !window.localStorage) {
        return defaults.map(d => ({ ...d }));
      }

      const raw = localStorage.getItem(storageKey);
      if (!raw) {
        return defaults.map(d => ({ ...d }));
      }

      const saved = JSON.parse(raw) as WidgetLayoutItem[];
      if (!Array.isArray(saved)) {
        return defaults.map(d => ({ ...d }));
      }

      // Merge inteligente: Conservar orden y spans guardados, agregando widgets nuevos de defaults
      const defaultsMap = new Map<string, WidgetLayoutItem>(defaults.map(d => [d.id, d]));
      const result: WidgetLayoutItem[] = [];
      const seenIds = new Set<string>();

      for (const item of saved) {
        const def = defaultsMap.get(item.id);
        if (def) {
          result.push({
            id: item.id,
            title: def.title,
            colSpan: item.colSpan || def.colSpan,
            visible: item.visible !== undefined ? item.visible : def.visible,
            minColSpan: def.minColSpan
          });
          seenIds.add(item.id);
        }
      }

      // Agregar los que no estaban en el storage guardado
      for (const def of defaults) {
        if (!seenIds.has(def.id)) {
          result.push({ ...def });
        }
      }

      return result;
    } catch {
      return defaults.map(d => ({ ...d }));
    }
  }

  saveLayout(storageKey: string, items: WidgetLayoutItem[]): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        localStorage.setItem(storageKey, JSON.stringify(items));
      }
    } catch {
      // Ignorar excepciones en entornos con storage deshabilitado
    }
  }

  resetLayout(storageKey: string, defaults: WidgetLayoutItem[]): WidgetLayoutItem[] {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        localStorage.removeItem(storageKey);
      }
    } catch {}
    return defaults.map(d => ({ ...d }));
  }
}
