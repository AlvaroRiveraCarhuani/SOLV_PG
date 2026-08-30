# Vista 1: Dashboard del Estudiante (Centro de Mando)

> **Especificación Oficial de Interfaz y Wireframe**  
> **Rol:** Estudiante  
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System  

---

## 1. Diagrama de Arquitectura de Pantalla (Mermaid Visual HD)

```mermaid
graph TD
    subgraph ShellEstudiante ["Shell del Estudiante (Layout Principal Colapsable)"]
        Topbar["Topbar Superior: Toggle Sidebar [◀|▶] | Logo UAB (White-Label) | Notificaciones | Perfil Usuario"]
        Sidebar["Sidebar Izquierdo (Colapsable / Modo Iconos): Inicio | Laboratorios | Cursos | Evaluaciones | Historial | Ajustes"]
        
        subgraph MainContent ["Área Principal de Contenido (Main Content Grid)"]
            Greeting["HeaderGreeting: Saludo Dinámico por Prioridad (Crítica / Acción / Normal)"]
            
            subgraph ColCentral ["Columna Central - Continuidad Técnica (66%)"]
                LabsWidget["Widget: Mis Laboratorios (Modelo Cuaderno)<br/>- Programación II: Lab #04 [Abrir IDE]<br/>- Sistemas Operativos: Lab #02 [Reanudar]<br/>- Bases de Datos: Lab #01 [Ver Nota: 85/100]"]
                ProgresoWidget["Widget: Mi Progreso (Placeholder)"]
                RecientesWidget["Widget: Accesos Recientes"]
            end
            
            subgraph ColDerecha ["Columna Derecha - Urgencia Académica (33%)"]
                AgendaWidget["Widget: Para Hoy (Urgencia + Dualidad de Estado)<br/>- Entregas Próximas<br/>- Ejercicios Pendientes"]
            end
        end
    end

    Topbar --> MainContent
    Sidebar --> MainContent
    Greeting --> ColCentral
    LabsWidget --> ColCentral
    AgendaWidget --> ColDerecha
```

---

## 2. Especificación Visual de Componentes e Iconografía Lucide
- **Sidebar Colapsable (Lucide Icons):** Botón Toggle (`lucide:panel-left-close` / `lucide:panel-left-open`). Módulos: `lucide:home` (Inicio), `lucide:terminal` (Laboratorios), `lucide:book-open` (Cursos), `lucide:award` (Evaluaciones), `lucide:history` (Historial), `lucide:settings` (Ajustes). En estado colapsado muestra únicamente la columna de iconos con tooltips.
- **Dualidad de Estados Absorbida:** El botón de acción absorbe el estado técnico sin métricas pasivas de RAM/CPU:
  - `[ Abrir IDE ]` (Estado: Running / En ejecución).
  - `[ Reanudar ]` (Estado: Hibernated / Pausado).
  - `[ Reintentar ]` (Estado: Error / Falla de inicio).
  - `[ Ver Nota: 85/100 ]` (Estado: Entregado / Calificado).

---

## 3. Diagrama ASCII Técnico — Dashboard Estudiante

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ SOLV | Branding UAB                                                      [Notif (2)]  Alvaro v   │
├──────────────┬───────────────────────────────────────────────────────────┬───────────────────────┤
│              │ [ HeaderGreeting: Prioridad Dinámica ]                    │                       │
│ [x] Inicio   │  Buenos días, Álvaro. Tienes 2 entregas pendientes.       │ [!] Para hoy          │
│ [x] Labs     │                                                           │                       │
│ [x] Cursos   │  [ WIDGET: MIS LABORATORIOS (Modelo Cuaderno) ]           │ [!] Lab #04: Prog II  │
│ [x] Evaluac. │  +-----------------------------------------------------+  │ Entrega: Mañana 18:00 │
│ [x] Historial│  | Programación II                                       |  │                       │
│              │  | Lab #04: Estructuras          [ Abrir IDE ] (Activo) |  │ [o] Ejercicio Alg     │
│ [*] Ajustes  │  | Lab #03: Algoritmos           [ Reanudar ] (Pausado) |  │ Suma de Arrays        │
│              │  | Lab #02: Queries SQL          [ Reintentar ] (Error) |  │                       │
│              │  +-----------------------------------------------------+  │ [o] Lab #02: BD SQL   │
│              │  | Bases de Datos                                       |  │ Entrega: 25 Ago       │
│              │  | Lab #01: Modelo ER          [ Ver nota: 85/100 ]    |  │                       │
│              │  +-----------------------------------------------------+  +-----------------------+
│              │                                                           │                       │
│              │  Progreso (Placeholder)                                   │ [>] Recientes         │
│              │  Prog II: [========  ] 80%   | BD: [======    ] 60%       │ • Lab #04 (hace 2h)   │
└──────────────┴───────────────────────────────────────────────────────────┴───────────────────────┘
```
