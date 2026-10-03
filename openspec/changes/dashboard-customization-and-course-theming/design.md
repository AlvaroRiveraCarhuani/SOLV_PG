# Technical Design: Personalización Libre de Dashboards y Tematización Híbrida

## Architecture Overview

```
[DashboardLayoutService]
   ├── loadLayout(dashboardKey, defaultWidgets)
   ├── saveLayout(dashboardKey, widgets)
   └── resetLayout(dashboardKey, defaultWidgets)

[TeacherDashboardComponent / StudentDashboardComponent]
   ├── State Signals:
   │    ├── isCustomizing = signal<boolean>(false)
   │    └── widgets = signal<WidgetLayoutItem[]>([...])
   │
   ├── Grid Container:
   │    <div cdkDropList class="bento-grid-12" (cdkDropListDropped)="onDropWidget($event)">
   │      @for (widget of visibleWidgets(); track widget.id) {
   │        <div cdkDrag [class]="'col-span-' + widget.colSpan">
   │          <!-- Controles de Edición si isCustomizing() -->
   │          <!-- Contenido del Widget -->
   │        </div>
   │      }
   │    </div>
```

## Data Contracts

```typescript
export type WidgetColSpan = 4 | 6 | 8 | 12;

export interface WidgetLayoutItem {
  id: string;
  title: string;
  colSpan: WidgetColSpan;
  visible: boolean;
  minColSpan?: WidgetColSpan;
}
```

## CSS Grid Layout

```scss
.bento-grid-12 {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: var(--space-4);

  @media (max-width: 768px) {
    display: flex;
    flex-direction: column;
  }
}

.col-span-12 { grid-column: span 12; }
.col-span-8  { grid-column: span 8; }
.col-span-6  { grid-column: span 6; }
.col-span-4  { grid-column: span 4; }
```
