-- ============================================================================
-- SOLV — Seed del Panel Docente (Carlos Mendoza Ríos)
-- Objetivo: simular un entorno real de uso docente para validación visual.
-- Idempotente y re-ejecutable. Solo datos, sin cambios de esquema.
-- Convenciones de IDs: e...-0001x = labs SIS-420, e...-002x = labs SIS-211,
--   a3... = entregas, c2... = workspaces, 9x... = comentarios/notificaciones/etc.
-- ============================================================================

BEGIN;

-- ============================================================================
-- B0. Guardián de identidad: Carlos activo + período 2026-II activo
-- ============================================================================
UPDATE users
SET first_name = 'Carlos', last_name = 'Mendoza Ríos',
    role = 'teacher', status = 'active', origin = 'manual'
WHERE id = '0e5e6bde-1933-4084-82b0-d481b4309ca7';

UPDATE academic_periods
SET is_active = true
WHERE id = 'e68d7d15-13b6-40c5-9549-d0adf0cab077';

-- ============================================================================
-- B3a. Alumnos nuevos de la cohorte (Renata, Joaquín)
-- ============================================================================
INSERT INTO users (id, first_name, last_name, email, role, tenant_id, origin, status, created_at)
VALUES
  ('e30c0f6d-ff63-4392-adcf-6336d6132b01', 'Renata', 'Vargas Ortiz', 'renata.vargas@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '12 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132b02', 'Joaquín', 'Mamani Condori', 'joaquin.mamani@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO UPDATE
SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
    email = EXCLUDED.email, status = EXCLUDED.status, origin = 'manual';

-- ============================================================================
-- B3b. Matrícula: completar la cohorte de Carlos (9 alumnos por curso)
-- ============================================================================
INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at)
VALUES
  -- SIS-420 (Desarrollo Web Fullstack) — ampliar a 9
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afb', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '26 days'), -- Lucas
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afd', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '26 days'), -- Daniel
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132b01', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '12 days'), -- Renata
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132b02', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '10 days'), -- Joaquín
  -- SIS-211 (Estructuras de Datos) — ampliar a 9
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afc', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '36 days'), -- Sofía
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132b01', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '12 days'), -- Renata
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132b02', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '10 days')  -- Joaquín
ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;

-- ============================================================================
-- B1. Labs adicionales por curso (matriz con varias columnas + "en riesgo")
-- SIS-420: Práctica 2 (JOINs), Práctica 3 (Express), Examen Parcial (due +12h)
-- ============================================================================
INSERT INTO exercises (
  id, tenant_id, subject_id, template_id, title, description,
  type, environment_type, language, status, difficulty, purpose,
  time_limit_ms, memory_limit_mb, stale, per_student_seed, tags,
  due_date, config, db_config, comparator_spec, boilerplate, reference_solution, created_at
) VALUES
(
  'e0000000-0000-0000-0000-000000000011',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  'Práctica 2: Consultas con JOIN y Agregados',
  'Sobre la base de ventas institucionales, escribir consultas SQL que crucen clientes y pedidos con JOIN, y agreguen totales por cliente usando GROUP BY y HAVING. Exponer los resultados vía un endpoint de consulta en el servicio.',
  'code',
  'IDE_PERSISTENTE',
  'typescript',
  'published',
  'easy',
  'class',
  5000,
  512,
  false,
  false,
  ARRAY['fullstack', 'sql', 'joins', 'postgresql'],
  NOW() + INTERVAL '18 days',
  '{"allowed_languages": ["typescript"]}'::jsonb,
  '{"enabled": true, "engine": "postgres"}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'export function topClients(rows: ClientRow[]): string[] {
  // Filtrar clientes con total >= 1000, ordenar por total desc y devolver nombres
  return [];
}',
  'export function topClients(rows: ClientRow[]): string[] {
  return rows
    .filter((r) => r.total >= 1000)
    .sort((a, b) => b.total - a.total)
    .map((r) => r.client);
}',
  NOW() - INTERVAL '18 days'
),
(
  'e0000000-0000-0000-0000-000000000012',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  'Práctica 3: API Express con pruebas unitarias',
  'Construir un servidor Express con una ruta GET /health y POST /productos que valide el payload. Incluir al menos una prueba unitaria con supertest que verifique el código 201 y el cuerpo de la respuesta.',
  'code',
  'IDE_PERSISTENTE',
  'typescript',
  'published',
  'medium',
  'class',
  8000,
  512,
  false,
  false,
  ARRAY['express', 'rest', 'testing', 'typescript'],
  NOW() + INTERVAL '30 days',
  '{"allowed_languages": ["typescript"]}'::jsonb,
  '{"enabled": false}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => {
    res.json({ status: "ok" });
  });
  return app;
}',
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => res.json({ status: "ok" }));
  app.post("/productos", (req, res) => {
    const { nombre, precio } = req.body ?? {};
    if (!nombre || typeof precio !== "number") {
      return res.status(400).json({ error: "payload inválido" });
    }
    return res.status(201).json({ id: crypto.randomUUID(), nombre, precio });
  });
  return app;
}',
  NOW() - INTERVAL '12 days'
),
(
  'e0000000-0000-0000-0000-000000000013',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  'Examen Parcial: API REST con autenticación',
  'Implementar un middleware de autenticación por token que proteja la ruta /pedidos. El token debe viajar en el header Authorization con el prefijo "Bearer " y validarse contra la lista de tokens emitidos. Responder 401 ante tokens inválidos o ausentes.',
  'code',
  'JUEZ_EFIMERO',
  'typescript',
  'published',
  'hard',
  'exam',
  8000,
  512,
  false,
  true,
  ARRAY['auth', 'middleware', 'examen', 'rest'],
  NOW() + INTERVAL '12 hours',
  '{"allowed_languages": ["typescript"]}'::jsonb,
  '{"enabled": false}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'const tokens = new Set<string>();
export function emitirToken(userId: string): string {
  const t = "tok-" + userId + "-" + Math.random().toString(36).slice(2);
  tokens.add(t);
  return t;
}
export function requireAuth(req: { headers: Record<string, string> }): boolean {
  // Retornar true si el header Authorization trae un Bearer token válido
  return false;
}',
  'const tokens = new Set<string>();
export function emitirToken(userId: string): string {
  const t = "tok-" + userId + "-" + Math.random().toString(36).slice(2);
  tokens.add(t);
  return t;
}
export function requireAuth(req: { headers: Record<string, string> }): boolean {
  const auth = req.headers["authorization"] ?? "";
  if (!auth.startsWith("Bearer ")) return false;
  const token = auth.slice("Bearer ".length).trim();
  return token !== "" && tokens.has(token);
}',
  NOW() - INTERVAL '10 days'
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title, description = EXCLUDED.description,
    status = EXCLUDED.status, difficulty = EXCLUDED.difficulty,
    due_date = EXCLUDED.due_date;

-- SIS-211: Práctica 2 (Pilas y Colas), Práctica 3 (Ordenamiento)
INSERT INTO exercises (
  id, tenant_id, subject_id, template_id, title, description,
  type, environment_type, language, status, difficulty, purpose,
  time_limit_ms, memory_limit_mb, stale, per_student_seed, tags,
  due_date, config, db_config, comparator_spec, boilerplate, reference_solution, created_at
) VALUES
(
  'e0000000-0000-0000-0000-000000000021',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  'Práctica 2: Pilas y Colas con Listas Enlazadas',
  'Implementar una cola con dos pilas y una pila con lista enlazada. La operación peek no debe modificar la estructura. Procesar una secuencia de comandos push/pop/peek desde la entrada estándar y reportar el resultado de cada operación.',
  'code',
  'IDE_PERSISTENTE',
  'python',
  'published',
  'medium',
  'class',
  3000,
  256,
  false,
  false,
  ARRAY['estructuras', 'pilas', 'colas', 'python'],
  NOW() + INTERVAL '22 days',
  '{"allowed_languages": ["python"]}'::jsonb,
  '{"enabled": false}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'class ColaConDosPilas:
    """Cola implementada con dos pilas: ingreso y egreso."""
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        pass

    def pop(self):
        pass

    def peek(self):
        pass',
  'class ColaConDosPilas:
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        self.entrada.append(x)

    def _mover(self):
        if not self.salida:
            while self.entrada:
                self.salida.append(self.entrada.pop())

    def pop(self):
        self._mover()
        return self.salida.pop()

    def peek(self):
        self._mover()
        return self.salida[-1]',
  NOW() - INTERVAL '15 days'
),
(
  'e0000000-0000-0000-0000-000000000022',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  'Práctica 3: Ordenamiento por Mezcla (Merge Sort)',
  'Implementar merge sort sobre una lista de enteros en orden no decreciente. La solución debe ser estable y correr en O(n log n). Reportar la lista ordenada y el número de comparaciones realizadas.',
  'code',
  'JUEZ_EFIMERO',
  'python',
  'published',
  'hard',
  'class',
  4000,
  256,
  false,
  false,
  ARRAY['ordenamiento', 'divide_y_venceras', 'complejidad'],
  NOW() + INTERVAL '28 days',
  '{"allowed_languages": ["python"]}'::jsonb,
  '{"enabled": false}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'def merge_sort(lista):
    """Retorna (lista_ordenada, comparaciones)."""
    return lista, 0',
  'def merge_sort(lista):
    if len(lista) <= 1:
        return lista[:], 0

    medio = len(lista) // 2
    izq, c1 = merge_sort(lista[:medio])
    der, c2 = merge_sort(lista[medio:])

    resultado = []
    i = j = 0
    comparaciones = c1 + c2
    while i < len(izq) and j < len(der):
        comparaciones += 1
        if izq[i] <= der[j]:
            resultado.append(izq[i]); i += 1
        else:
            resultado.append(der[j]); j += 1
    resultado.extend(izq[i:])
    resultado.extend(der[j:])
    return resultado, comparaciones',
  NOW() - INTERVAL '10 days'
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title, description = EXCLUDED.description,
    status = EXCLUDED.status, difficulty = EXCLUDED.difficulty,
    due_date = EXCLUDED.due_date;

-- ============================================================================
-- B2. Rúbricas de los labs nuevos
-- ============================================================================
INSERT INTO rubrics (id, exercise_id, criteria, created_at, updated_at)
VALUES
(
  'f0000000-0000-0000-0000-000000000003',
  'e0000000-0000-0000-0000-000000000011',
  '[
    {"title": "Corrección de la consulta JOIN", "max_score": 45, "description": "La consulta cruza correctamente clientes y pedidos y agrupa por cliente."},
    {"title": "Agregados y HAVING", "max_score": 35, "description": "Usa SUM/GROUP BY y filtra con HAVING los totales mayores o iguales a 1000."},
    {"title": "Orden y presentación", "max_score": 20, "description": "Devuelve nombres ordenados descendentemente por total."}
  ]'::jsonb,
  NOW() - INTERVAL '18 days',
  NOW() - INTERVAL '18 days'
),
(
  'f0000000-0000-0000-0000-000000000004',
  'e0000000-0000-0000-0000-000000000012',
  '[
    {"title": "Rutas y métodos", "max_score": 40, "description": "GET /health y POST /productos implementados con el status esperado."},
    {"title": "Validación de payload", "max_score": 30, "description": "Rechaza payloads incompletos con 400 y crea con 201."},
    {"title": "Pruebas unitarias", "max_score": 30, "description": "Al menos una prueba con supertest que cubra el flujo feliz y el de error."}
  ]'::jsonb,
  NOW() - INTERVAL '12 days',
  NOW() - INTERVAL '12 days'
),
(
  'f0000000-0000-0000-0000-000000000005',
  'e0000000-0000-0000-0000-000000000013',
  '[
    {"title": "Middleware de autenticación", "max_score": 50, "description": "Protege /pedidos validando el Bearer token en el header."},
    {"title": "Manejo de errores", "max_score": 30, "description": "Devuelve 401 ante token ausente, inválido o expirado."},
    {"title": "Emisión de tokens", "max_score": 20, "description": "Emitir token único por usuario y verificar pertenencia."}
  ]'::jsonb,
  NOW() - INTERVAL '10 days',
  NOW() - INTERVAL '10 days'
),
(
  'f0000000-0000-0000-0000-000000000006',
  'e0000000-0000-0000-0000-000000000021',
  '[
    {"title": "Cola con dos pilas", "max_score": 45, "description": "push/pop/peek mantienen orden FIFO sin modificar en peek."},
    {"title": "Pila con lista enlazada", "max_score": 35, "description": "Operaciones en O(1) con nodos enlazados."},
    {"title": "Procesamiento de comandos", "max_score": 20, "description": "Interpreta la secuencia de entrada y reporta cada resultado."}
  ]'::jsonb,
  NOW() - INTERVAL '15 days',
  NOW() - INTERVAL '15 days'
),
(
  'f0000000-0000-0000-0000-000000000007',
  'e0000000-0000-0000-0000-000000000022',
  '[
    {"title": "Correctitud del ordenamiento", "max_score": 45, "description": "La lista resultante está en orden no decreciente."},
    {"title": "Complejidad O(n log n)", "max_score": 35, "description": "División recursiva con mezcla, sin usar sort() nativo."},
    {"title": "Conteo de comparaciones", "max_score": 20, "description": "Reporta el número exacto de comparaciones."}
  ]'::jsonb,
  NOW() - INTERVAL '10 days',
  NOW() - INTERVAL '10 days'
)
ON CONFLICT (id) DO UPDATE
SET criteria = EXCLUDED.criteria;

-- ============================================================================
-- B2b. Casos de prueba de los labs nuevos
-- ============================================================================
INSERT INTO exercise_test_cases (id, exercise_id, order_index, input, expected_output, visibility, weight, created_at)
VALUES
  -- Práctica 2 JOINs (011)
  ('d0000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000011', 1,
   '[{"client":"Aceros Bo","total":1500},{"client":"Textil Andina","total":820},{"client":"Lácteos Sur","total":1200}]',
   '["Aceros Bo","Lácteos Sur"]', 'public', 60.0, NOW() - INTERVAL '18 days'),
  ('d0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000011', 2,
   '[{"client":"Aceros Bo","total":999},{"client":"Andina","total":1000}]',
   '["Andina"]', 'hidden', 40.0, NOW() - INTERVAL '18 days'),
  -- Práctica 3 Express (012)
  ('d0000000-0000-0000-0000-000000000021', 'e0000000-0000-0000-0000-000000000012', 1,
   'GET /health', '{"status":"ok"}', 'public', 40.0, NOW() - INTERVAL '12 days'),
  ('d0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000012', 2,
   'POST /productos {"nombre":"Teclado","precio":180}', '201 + id', 'hidden', 60.0, NOW() - INTERVAL '12 days'),
  -- Examen Parcial (013)
  ('d0000000-0000-0000-0000-000000000031', 'e0000000-0000-0000-0000-000000000013', 1,
   'Authorization: Bearer tok-1-abcd', 'true', 'public', 40.0, NOW() - INTERVAL '10 days'),
  ('d0000000-0000-0000-0000-000000000032', 'e0000000-0000-0000-0000-000000000013', 2,
   'Authorization: Bearer tok-invalido', 'false', 'hidden', 30.0, NOW() - INTERVAL '10 days'),
  ('d0000000-0000-0000-0000-000000000033', 'e0000000-0000-0000-0000-000000000013', 3,
   'sin header', 'false', 'hidden', 30.0, NOW() - INTERVAL '10 days'),
  -- Práctica 2 Pilas y Colas (021)
  ('d0000000-0000-0000-0000-000000000041', 'e0000000-0000-0000-0000-000000000021', 1,
   'push 1\npush 2\npeek\npop\npeek', '2\n2\n1', 'public', 40.0, NOW() - INTERVAL '15 days'),
  ('d0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000021', 2,
   'push 5\npop\npop', '5\nerror', 'hidden', 30.0, NOW() - INTERVAL '15 days'),
  ('d0000000-0000-0000-0000-000000000043', 'e0000000-0000-0000-0000-000000000021', 3,
   'peek\npush 3\npeek', 'error\n3', 'hidden', 30.0, NOW() - INTERVAL '15 days'),
  -- Práctica 3 Ordenamiento (022)
  ('d0000000-0000-0000-0000-000000000051', 'e0000000-0000-0000-0000-000000000022', 1,
   '[4, 2, 9, 1, 5]', '([1, 2, 4, 5, 9], 7)', 'public', 40.0, NOW() - INTERVAL '10 days'),
  ('d0000000-0000-0000-0000-000000000052', 'e0000000-0000-0000-0000-000000000022', 2,
   '[1, 2, 3, 4]', '([1, 2, 3, 4], 4)', 'hidden', 30.0, NOW() - INTERVAL '10 days'),
  ('d0000000-0000-0000-0000-000000000053', 'e0000000-0000-0000-0000-000000000022', 3,
   '[9, 8, 7, 6, 5]', '([5, 6, 7, 8, 9], 10)', 'hidden', 30.0, NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO UPDATE
SET input = EXCLUDED.input, expected_output = EXCLUDED.expected_output;
-- ============================================================================
-- B4. Workspaces: refrescar los existentes y añadir la variedad de estados
-- "Activos ahora" = heartbeat < 2 min (Camilas, Sofía, Valeria)
-- ============================================================================
-- Camila SIS-420 sigue running y activa ahora
UPDATE workspaces
SET status = 'running',
    container_id = 'solv-ws-camila-420',
    last_heartbeat_at = NOW() - INTERVAL '1 minute',
    updated_at = NOW()
WHERE id = 'c1000000-0000-0000-0000-000000000002';

-- Valeria SIS-211: hibernada -> running y activa ahora
UPDATE workspaces
SET status = 'running',
    container_id = 'solv-ws-valeria-211',
    last_heartbeat_at = NOW() - INTERVAL '40 seconds',
    updated_at = NOW()
WHERE id = '62e0854d-76b0-4a4e-abc1-d4086bebcc4f';

-- Daniel SIS-211: se mantiene hibernado, heartbeat hace 3h (no cuenta como activo)
UPDATE workspaces
SET last_heartbeat_at = NOW() - INTERVAL '3 hours',
    updated_at = NOW()
WHERE id = 'c1000000-0000-0000-0000-000000000005';

-- Mateo SIS-211: hibernado hace 5h
UPDATE workspaces
SET last_heartbeat_at = NOW() - INTERVAL '5 hours',
    updated_at = NOW()
WHERE id = 'c1000000-0000-0000-0000-000000000003';

-- Workspaces nuevos de la cohorte de Carlos
INSERT INTO workspaces (
  id, student_id, subject_id, template_id, tenant_id, status, type,
  container_id, access_url, memory_limit_mb, oom_strike_count,
  last_heartbeat_at, last_oom_killed_at, created_at, updated_at
) VALUES
(
  'c2000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía, SIS-420
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'running', 'IDE_PERSISTENTE',
  'solv-ws-sofia-420',
  'https://workspace-sofia.solv.uab.edu.bo',
  512, 0,
  NOW() - INTERVAL '1 minute', null,
  NOW() - INTERVAL '6 hours', NOW() - INTERVAL '1 minute'
),
(
  'c2000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía, SIS-211
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'running', 'IDE_PERSISTENTE',
  'solv-ws-sofia-211',
  'https://workspace-sofia.solv.uab.edu.bo',
  512, 0,
  NOW() - INTERVAL '40 seconds', null,
  NOW() - INTERVAL '5 hours', NOW() - INTERVAL '40 seconds'
),
(
  'c2000000-0000-0000-0000-000000000003',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas, SIS-211 (OOM crítico)
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'oom_killed', 'IDE_PERSISTENTE',
  'solv-ws-lucas-211',
  'https://workspace-lucas.solv.uab.edu.bo',
  256, 2,
  NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours',
  NOW() - INTERVAL '1 day', NOW() - INTERVAL '2 hours'
),
(
  'c2000000-0000-0000-0000-000000000004',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo, SIS-420
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'hibernated', 'IDE_PERSISTENTE',
  'solv-ws-mateo-420',
  'https://workspace-mateo.solv.uab.edu.bo',
  1024, 0,
  NOW() - INTERVAL '5 hours', null,
  NOW() - INTERVAL '4 days', NOW() - INTERVAL '5 hours'
),
(
  'c2000000-0000-0000-0000-000000000005',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila, SIS-211
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'hibernated', 'IDE_PERSISTENTE',
  'solv-ws-camila-211',
  'https://workspace-camila.solv.uab.edu.bo',
  512, 0,
  NOW() - INTERVAL '3 hours', null,
  NOW() - INTERVAL '2 days', NOW() - INTERVAL '3 hours'
),
(
  'c2000000-0000-0000-0000-000000000006',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel, SIS-420
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'hibernated', 'IDE_PERSISTENTE',
  'solv-ws-daniel-420',
  'https://workspace-daniel.solv.uab.edu.bo',
  512, 0,
  NOW() - INTERVAL '6 hours', null,
  NOW() - INTERVAL '3 days', NOW() - INTERVAL '6 hours'
),
(
  'c2000000-0000-0000-0000-000000000007',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas, SIS-420 (contenedor caído)
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'failed', 'IDE_PERSISTENTE',
  null,
  'https://workspace-lucas-420.solv.uab.edu.bo',
  512, 0,
  NOW() - INTERVAL '90 minutes', null,
  NOW() - INTERVAL '5 days', NOW() - INTERVAL '90 minutes'
)
ON CONFLICT (id) DO UPDATE
SET status = EXCLUDED.status,
    container_id = EXCLUDED.container_id,
    last_heartbeat_at = EXCLUDED.last_heartbeat_at,
    last_oom_killed_at = EXCLUDED.last_oom_killed_at,
    oom_strike_count = EXCLUDED.oom_strike_count,
    updated_at = EXCLUDED.updated_at;
-- ============================================================================
-- B5. Entregas de la cohorte (mezcla de veredictos y perfiles por estudiante)
-- Veredictos: AC=verde, WA=rojo, TLE=ámbar, RE=naranja, AST_BLOCKED=rojo intenso
-- ============================================================================

-- --- SIS-211 Práctica 1: Búsqueda Binaria (exercise 001) -------------------
-- (ya existen Valeria AC100, Camila AC95, Mateo WA50, Lucas TLE25, Daniel AC90 pendiente)
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000002',
  'def busqueda_binaria(arr, objetivo):
    izq, der = 0, len(arr) - 1
    while izq <= der:
        medio = (izq + der) // 2
        if arr[medio] == objetivo:
            return medio
        if arr[medio] < objetivo:
            izq = medio + 1
        else:
            der = medio - 1
    return -1',
  'AC',
  '{"score": 97, "similarity": 0.04, "complexity": "O(log n)", "structure_hash": "a1b2c3d4"}'::jsonb,
  36, 17, 97,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '5 days'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result;

-- --- SIS-211 Práctica 2: Pilas y Colas (exercise 021) ----------------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000011',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '62e0854d-76b0-4a4e-abc1-d4086bebcc4f',
  'class Nodo:
    def __init__(self, valor):
        self.valor = valor
        self.siguiente = None

class ColaConDosPilas:
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        self.entrada.append(x)

    def _mover(self):
        if not self.salida:
            while self.entrada:
                self.salida.append(self.entrada.pop())

    def pop(self):
        self._mover()
        return self.salida.pop()

    def peek(self):
        self._mover()
        return self.salida[-1]

class PilaEnlazada:
    def __init__(self):
        self.tope = None

    def push(self, x):
        nuevo = Nodo(x)
        nuevo.siguiente = self.tope
        self.tope = nuevo

    def pop(self):
        if self.tope is None:
            raise IndexError("pila vacia")
        valor = self.tope.valor
        self.tope = self.tope.siguiente
        return valor',
  'AC',
  '{"score": 100, "similarity": 0.06, "complexity": "O(1) amortizado", "structure_hash": "f1e2d3c4"}'::jsonb,
  88, 21, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '3 days'
),
(
  'a3000000-0000-0000-0000-000000000012',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000002',
  'class ColaConDosPilas:
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        self.entrada.append(x)

    def pop(self):
        if not self.salida:
            while self.entrada:
                self.salida.append(self.entrada.pop())
        return self.salida.pop()

    def peek(self):
        if not self.salida:
            while self.entrada:
                self.salida.append(self.entrada.pop())
        return self.salida[-1]

class PilaEnlazada:
    class Nodo:
        def __init__(self, valor):
            self.valor = valor
            self.siguiente = None

    def __init__(self):
        self.tope = None

    def push(self, x):
        nodo = self.Nodo(x)
        nodo.siguiente = self.tope
        self.tope = nodo

    def pop(self):
        if not self.tope:
            raise IndexError("vacia")
        valor = self.tope.valor
        self.tope = self.tope.siguiente
        return valor',
  'AC',
  '{"score": 96, "similarity": 0.10, "complexity": "O(1) amortizado", "structure_hash": "a2b2c2d2"}'::jsonb,
  91, 22, 96,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '3 days'
),
(
  'a3000000-0000-0000-0000-000000000013',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila
  'c2000000-0000-0000-0000-000000000005',
  'class ColaConDosPilas:
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        self.entrada.append(x)

    def pop(self):
        if self.salida:
            return self.salida.pop()
        while self.entrada:
            self.salida.append(self.entrada.pop())
        return self.salida.pop()

    def peek(self):
        if self.salida:
            return self.salida[-1]
        while self.entrada:
            self.salida.append(self.entrada.pop())
        return self.salida[-1]

class PilaEnlazada:
    def __init__(self):
        self.items = []

    def push(self, x):
        self.items.append(x)

    def pop(self):
        if not self.items:
            raise IndexError("vacia")
        return self.items.pop()',
  'AC',
  '{"score": 90, "similarity": 0.30, "complexity": "O(1) amortizado", "structure_hash": "b3c3d3e3"}'::jsonb,
  95, 23, 90,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '2 days'
),
(
  'a3000000-0000-0000-0000-000000000014',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo
  'c1000000-0000-0000-0000-000000000003',
  'class ColaConDosPilas:
    def __init__(self):
        self.items = []

    def push(self, x):
        self.items.append(x)

    def pop(self):
        return self.items.pop(0)

    def peek(self):
        return self.items[0]

class PilaEnlazada:
    def __init__(self):
        self.items = []

    def push(self, x):
        self.items.append(x)

    def pop(self):
        return self.items.pop()',
  'AC',
  '{"score": 78, "similarity": 0.22, "complexity": "O(n) en pop", "structure_hash": "c4d4e4f4"}'::jsonb,
  1220, 25, 78,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '2 days'
),
(
  'a3000000-0000-0000-0000-000000000015',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel
  'c1000000-0000-0000-0000-000000000005',
  'class ColaConDosPilas:
    def __init__(self):
        self.entrada = []
        self.salida = []

    def push(self, x):
        self.entrada.append(x)

    def pop(self):
        if not self.salida:
            while self.entrada:
                self.salida.append(self.entrada.pop())
        return self.salida.pop()

    def peek(self):
        return self.entrada[-1] if self.entrada else None

class PilaEnlazada:
    def __init__(self):
        self.tope = None

    def push(self, x):
        self.tope = [x, self.tope]

    def pop(self):
        if self.tope is None:
            raise IndexError("vacia")
        valor, restante = self.tope
        self.tope = restante
        return valor',
  'AC',
  '{"score": 88, "similarity": 0.45, "complexity": "O(1) amortizado", "structure_hash": "d5e5f5a6"}'::jsonb,
  97, 22, 88,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '1 day'
),
(
  'a3000000-0000-0000-0000-000000000016',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000021',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas (AST bloqueado)
  'c2000000-0000-0000-0000-000000000003',
  'import os
from collections import deque

class ColaConDosPilas:
    def __init__(self):
        self.entrada = deque()
        self.salida = deque()

    def push(self, x):
        self.entrada.append(x)

    def pop(self):
        while not self.salida:
            self.salida.append(self.entrada.popleft())
        return self.salida.popleft()

    def peek(self):
        return self.salida[0] if self.salida else self.entrada[0]

# procesar comandos de la terminal del laboratorio
comando = input()
os.system(comando)',
  'AST_BLOCKED',
  '{"score": 0, "rule_id": "python.import-forbidden", "violation": "uso de os.system prohibido en entorno evaluador", "similarity": 0.02, "structure_hash": "e6f6a1b7"}'::jsonb,
  0, 0, null,
  null, false,
  NOW() - INTERVAL '6 hours'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result,
    manual_override = EXCLUDED.manual_override;

-- --- SIS-211 Práctica 3: Ordenamiento Merge Sort (exercise 022) ------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000021',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000022',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '62e0854d-76b0-4a4e-abc1-d4086bebcc4f',
  'def merge_sort(lista):
    if len(lista) <= 1:
        return lista[:], 0

    medio = len(lista) // 2
    izq, c1 = merge_sort(lista[:medio])
    der, c2 = merge_sort(lista[medio:])

    resultado = []
    i = j = 0
    comparaciones = c1 + c2
    while i < len(izq) and j < len(der):
        comparaciones += 1
        if izq[i] <= der[j]:
            resultado.append(izq[i]); i += 1
        else:
            resultado.append(der[j]); j += 1
    resultado.extend(izq[i:])
    resultado.extend(der[j:])
    return resultado, comparaciones',
  'AC',
  '{"score": 100, "similarity": 0.05, "complexity": "O(n log n)", "structure_hash": "a7b7c7d7"}'::jsonb,
  152, 20, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '1 day'
),
(
  'a3000000-0000-0000-0000-000000000022',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000022',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000002',
  'def merge_sort(lista):
    if len(lista) <= 1:
        return lista, 0

    medio = len(lista) // 2
    izquierda, c1 = merge_sort(lista[:medio])
    derecha, c2 = merge_sort(lista[medio:])

    ordenada = []
    i = j = 0
    comparaciones = c1 + c2 + 1
    while i < len(izquierda) and j < len(derecha):
        if izquierda[i] <= derecha[j]:
            ordenada.append(izquierda[i]); i += 1
        else:
            ordenada.append(derecha[j]); j += 1
    ordenada += izquierda[i:] + derecha[j:]
    return ordenada, comparaciones',
  'AC',
  '{"score": 94, "similarity": 0.08, "complexity": "O(n log n)", "structure_hash": "b8c8d8e8"}'::jsonb,
  160, 21, 94,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '1 day'
),
(
  'a3000000-0000-0000-0000-000000000023',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000022',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila
  'c2000000-0000-0000-0000-000000000005',
  'def merge_sort(lista):
    if len(lista) <= 1:
        return lista, 0

    medio = len(lista) // 2
    izq, c1 = merge_sort(lista[:medio])
    der, c2 = merge_sort(lista[medio:])

    combinada = sorted(izq + der)
    return combinada, c1 + c2 + len(izq)',
  'AC',
  '{"score": 85, "similarity": 0.35, "complexity": "O(n log n)", "structure_hash": "c9d9e9f9"}'::jsonb,
  175, 22, 85,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '20 hours'
),
(
  'a3000000-0000-0000-0000-000000000024',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000022',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel (pendiente de revisión)
  'c1000000-0000-0000-0000-000000000005',
  'def merge_sort(lista):
    if len(lista) <= 1:
        return lista, 0

    medio = len(lista) // 2
    izq, c1 = merge_sort(lista[:medio])
    der, c2 = merge_sort(lista[medio:])

    resultado = []
    i = j = 0
    comparaciones = c1 + c2
    while i < len(izq) and j < len(der):
        comparaciones += 1
        if izq[i] < der[j]:
            resultado.append(izq[i]); i += 1
        else:
            resultado.append(der[j]); j += 1
    resultado.extend(izq[i:])
    resultado.extend(der[j:])
    return resultado, comparaciones',
  'AC',
  '{"score": 90, "similarity": 0.60, "complexity": "O(n log n)", "structure_hash": "d0e0f0a1", "plagiarism_flag": true, "match_student": "Camila Rojas"}'::jsonb,
  158, 20, null,
  null, false,
  NOW() - INTERVAL '5 hours'
),
(
  'a3000000-0000-0000-0000-000000000025',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000022',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas (error de ejecución)
  'c2000000-0000-0000-0000-000000000003',
  'def merge_sort(lista):
    medio = len(lista) // 2
    izq = merge_sort(lista[:medio])
    der = merge_sort(lista[medio:])
    return izq + der',
  'RE',
  '{"score": 15, "similarity": 0.03, "complexity": "desconocida", "structure_hash": "e1f1a2b2", "error": "RecursionError: falta caso base"}'::jsonb,
  998, 18, 15,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '4 hours'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result;
-- ============================================================================
-- B5b. Entregas de SIS-420 (Desarrollo Web Fullstack)
-- ============================================================================

-- --- SIS-420 Práctica 1: Endpoints REST (exercise 002) ----------------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override,
  override_reason, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000031',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '6ed99b6e-fd02-4256-9bf1-5fd2232ac932',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    const subtotal = items.reduce((acc, item) => acc + item.price, 0);
    return Number((subtotal * 1.13).toFixed(2));
  }
}',
  'AC',
  '{"score": 100, "similarity": 0.04, "complexity": "O(n)", "structure_hash": "a1a2a3a4"}'::jsonb,
  34, 17, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false, '',
  NOW() - INTERVAL '8 days'
),
(
  'a3000000-0000-0000-0000-000000000032',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000001',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    let subtotal = 0;
    for (const item of items) {
      subtotal += item.price;
    }
    return Math.round(subtotal * 1.13 * 100) / 100;
  }
}',
  'AC',
  '{"score": 100, "similarity": 0.09, "complexity": "O(n)", "structure_hash": "b2b3b4b5"}'::jsonb,
  36, 17, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false, '',
  NOW() - INTERVAL '8 days'
),
(
  'a3000000-0000-0000-0000-000000000033',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila
  'c1000000-0000-0000-0000-000000000002',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    let subtotal = 0;
    for (const item of items) {
      subtotal = subtotal + item.price * 1.13;
    }
    return subtotal;
  }
}',
  'AC',
  '{"score": 92, "similarity": 0.31, "complexity": "O(n)", "structure_hash": "c3c4c5c6"}'::jsonb,
  41, 18, 92,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false, '',
  NOW() - INTERVAL '7 days'
),
(
  'a3000000-0000-0000-0000-000000000034',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo (auditada manualmente)
  'c2000000-0000-0000-0000-000000000004',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    let subtotal = 0;
    for (const item of items) {
      subtotal += item.price;
    }
    // FALTA aplicar el 13% de IVA sobre el subtotal
    return subtotal;
  }
}',
  'WA',
  '{"score": 40, "similarity": 0.18, "complexity": "O(n)", "structure_hash": "d4d5d6d7"}'::jsonb,
  33, 16, 40,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', true,
  'Se audita: la lógica omite el IVA del 13%; se otorga puntaje parcial por estructura correcta.',
  NOW() - INTERVAL '6 days'
),
(
  'a3000000-0000-0000-0000-000000000035',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas (error de ejecución)
  'c2000000-0000-0000-0000-000000000007',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    const subtotal = items.reduce((acc, item) => acc + item.price, 0);
    const iva = subtotal * 1.13;
    return iva.toFixed(2) / 1; // TypeError: toFixed devuelve string
  }
}',
  'RE',
  '{"score": 30, "similarity": 0.05, "complexity": "O(n)", "structure_hash": "e5e6e7e8", "error": "TypeError: división de string"}'::jsonb,
  12, 15, 30,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false, '',
  NOW() - INTERVAL '6 days'
),
(
  'a3000000-0000-0000-0000-000000000036',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel (auditada manualmente)
  'c2000000-0000-0000-0000-000000000006',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    const subtotal = items.reduce((acc, item) => acc + item.price, 0);
    return Math.round(subtotal * 1.13 * 100) / 100;
  }
}',
  'AC',
  '{"score": 96, "similarity": 0.25, "complexity": "O(n)", "structure_hash": "f6f7f8f9"}'::jsonb,
  39, 18, 96,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', true,
  'Se confirma AC: redondeo por Math.round aceptable para el rango de precios.',
  NOW() - INTERVAL '5 days'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result,
    manual_override = EXCLUDED.manual_override, override_reason = EXCLUDED.override_reason;

-- --- SIS-420 Práctica 2: JOINs (exercise 011) -------------------------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000041',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000011',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '6ed99b6e-fd02-4256-9bf1-5fd2232ac932',
  'export function topClients(rows: { client: string; total: number }[]): string[] {
  return rows
    .filter((r) => r.total >= 1000)
    .sort((a, b) => b.total - a.total)
    .map((r) => r.client);
}',
  'AC',
  '{"score": 100, "similarity": 0.05, "complexity": "O(n log n)", "structure_hash": "a1b2c3"}'::jsonb,
  28, 16, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '5 days'
),
(
  'a3000000-0000-0000-0000-000000000042',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000011',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000001',
  'export function topClients(rows: { client: string; total: number }[]): string[] {
  const elegibles: { client: string; total: number }[] = [];
  for (const r of rows) {
    if (r.total >= 1000) elegibles.push(r);
  }
  elegibles.sort((a, b) => b.total - a.total);
  return elegibles.map((r) => r.client);
}',
  'AC',
  '{"score": 95, "similarity": 0.12, "complexity": "O(n log n)", "structure_hash": "b2c3d4"}'::jsonb,
  31, 17, 95,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '5 days'
),
(
  'a3000000-0000-0000-0000-000000000043',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000011',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila
  'c1000000-0000-0000-0000-000000000002',
  'export function topClients(rows: { client: string; total: number }[]): string[] {
  return rows
    .filter((r) => r.total >= 1000)
    .sort((a, b) => a.total - b.total)
    .map((r) => r.client);
}',
  'AC',
  '{"score": 88, "similarity": 0.28, "complexity": "O(n log n)", "structure_hash": "c3d4e5"}'::jsonb,
  34, 17, 88,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '4 days'
),
(
  'a3000000-0000-0000-0000-000000000044',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000011',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo
  'c2000000-0000-0000-0000-000000000004',
  'export function topClients(rows: { client: string; total: number }[]): string[] {
  return rows.filter((r) => r.total >= 1000).map((r) => r.client);
}',
  'AC',
  '{"score": 75, "similarity": 0.20, "complexity": "O(n)", "structure_hash": "d4e5f6"}'::jsonb,
  27, 15, 75,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '4 days'
),
(
  'a3000000-0000-0000-0000-000000000045',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000011',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel
  'c2000000-0000-0000-0000-000000000006',
  'export function topClients(rows: { client: string; total: number }[]): string[] {
  return rows
    .filter((r) => r.total > 999)
    .sort((a, b) => b.total - a.total)
    .map((r) => r.client);
}',
  'AC',
  '{"score": 90, "similarity": 0.42, "complexity": "O(n log n)", "structure_hash": "e5f6a7"}'::jsonb,
  33, 17, 90,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result;

-- --- SIS-420 Práctica 3: Express + tests (exercise 012) ---------------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000051',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000012',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '6ed99b6e-fd02-4256-9bf1-5fd2232ac932',
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => res.json({ status: "ok" }));
  app.post("/productos", (req, res) => {
    const { nombre, precio } = req.body ?? {};
    if (!nombre || typeof precio !== "number") {
      return res.status(400).json({ error: "payload inválido" });
    }
    return res.status(201).json({ id: crypto.randomUUID(), nombre, precio });
  });
  return app;
}',
  'AC',
  '{"score": 100, "similarity": 0.05, "complexity": "O(1)", "structure_hash": "a1c2e3"}'::jsonb,
  45, 19, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '3 days'
),
(
  'a3000000-0000-0000-0000-000000000052',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000012',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000001',
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => res.json({ status: "ok" }));
  app.post("/productos", (req, res) => {
    const body = req.body ?? {};
    if (!body.nombre) return res.status(400).json({ error: "falta nombre" });
    return res.status(201).json({ id: "p-" + Date.now(), nombre: body.nombre, precio: body.precio });
  });
  return app;
}',
  'AC',
  '{"score": 92, "similarity": 0.14, "complexity": "O(1)", "structure_hash": "b2d4f6"}'::jsonb,
  48, 20, 92,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '3 days'
),
(
  'a3000000-0000-0000-0000-000000000053',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000012',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila
  'c1000000-0000-0000-0000-000000000002',
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => res.json({ status: "ok" }));
  app.post("/productos", (_req, res) => {
    return res.status(201).json({ id: "p-1" });
  });
  return app;
}',
  'AC',
  '{"score": 80, "similarity": 0.22, "complexity": "O(1)", "structure_hash": "c3e5a7"}'::jsonb,
  44, 19, 80,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '2 days'
),
(
  'a3000000-0000-0000-0000-000000000054',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000012',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel
  'c2000000-0000-0000-0000-000000000006',
  'import express from "express";
export function createApp() {
  const app = express();
  app.use(express.json());
  app.get("/health", (_req, res) => res.json({ status: "ok" }));
  app.post("/productos", (req, res) => {
    const { nombre, precio } = req.body ?? {};
    if (!nombre || typeof precio !== "number") {
      return res.status(400).json({ error: "payload inválido" });
    }
    return res.status(201).json({ nombre, precio });
  });
  return app;
}',
  'AC',
  '{"score": 85, "similarity": 0.50, "complexity": "O(1)", "structure_hash": "d4f6a8"}'::jsonb,
  47, 20, 85,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result;

-- --- SIS-420 Examen Parcial: API con auth (exercise 013) --------------------
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict,
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override,
  generated_cases, submitted_at
) VALUES
(
  'a3000000-0000-0000-0000-000000000061',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000013',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria
  '6ed99b6e-fd02-4256-9bf1-5fd2232ac932',
  'const tokens = new Set<string>();
export function emitirToken(userId: string): string {
  const t = "tok-" + userId + "-" + Math.random().toString(36).slice(2);
  tokens.add(t);
  return t;
}
export function requireAuth(req: { headers: Record<string, string> }): boolean {
  const auth = req.headers["authorization"] ?? "";
  if (!auth.startsWith("Bearer ")) return false;
  const token = auth.slice("Bearer ".length).trim();
  return token !== "" && tokens.has(token);
}',
  'AC',
  '{"score": 100, "similarity": 0.04, "complexity": "O(1)", "structure_hash": "a9b8c7"}'::jsonb,
  22, 14, 100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  '{"seed": 2026, "cases_generated": 3}'::jsonb,
  NOW() - INTERVAL '3 hours'
),
(
  'a3000000-0000-0000-0000-000000000062',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000013',
  'e30c0f6d-ff63-4392-adcf-6336d6132afc', -- Sofía
  'c2000000-0000-0000-0000-000000000001',
  'const tokens = new Set<string>();
export function emitirToken(userId: string): string {
  const t = "tok-" + userId + "-" + Math.random().toString(36).slice(2);
  tokens.add(t);
  return t;
}
export function requireAuth(req: { headers: Record<string, string> }): boolean {
  const auth = req.headers["authorization"] ?? "";
  return auth.startsWith("Bearer ");
}',
  'AC',
  '{"score": 90, "similarity": 0.10, "complexity": "O(1)", "structure_hash": "b8c7d6"}'::jsonb,
  24, 15, 90,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', false,
  '{"seed": 2026, "cases_generated": 3}'::jsonb,
  NOW() - INTERVAL '2 hours'
)
ON CONFLICT (id) DO UPDATE
SET verdict = EXCLUDED.verdict, score = EXCLUDED.score, ast_result = EXCLUDED.ast_result;
-- ============================================================================
-- B6. Keystroke events para el Replay del SpeedGrader (modo solo lectura)
-- ============================================================================
-- Lucas, Práctica 2 (016): incluye un pegado detectado (os.system)
DELETE FROM submission_keystroke_events WHERE submission_id = 'a3000000-0000-0000-0000-000000000016';
INSERT INTO submission_keystroke_events (
  id, submission_id, timestamp_ms, event_type, position, content, paste_source_detected, created_at
) VALUES
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 800,   'insert', 0,   'import os\n',                                false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 2400,  'insert', 10,  'from collections import deque\n',              false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 5100,  'insert', 36,  'class ColaConDosPilas:\n',                    false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 8300,  'insert', 59,  '    def __init__(self):\n',                   false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 11900, 'insert', 82,  '        self.entrada = deque()\n',              false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 14700, 'insert', 108, '        self.salida = deque()\n',                false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 18300, 'insert', 134, '    def push(self, x):\n',                      false, NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 22100, 'paste',  167, 'comando = input()\nos.system(comando)\n',      true,  NOW() - INTERVAL '6 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000016', 23800, 'insert', 207, '# procesar comandos del laboratorio\n',          false, NOW() - INTERVAL '6 hours');

-- Daniel, Práctica 3 (024): entrega pendiente con replay
DELETE FROM submission_keystroke_events WHERE submission_id = 'a3000000-0000-0000-0000-000000000024';
INSERT INTO submission_keystroke_events (
  id, submission_id, timestamp_ms, event_type, position, content, paste_source_detected, created_at
) VALUES
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 900,   'insert', 0,   'def merge_sort(lista):\n',                 false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 4100,  'insert', 21,  '    if len(lista) <= 1:\n',                false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 7200,  'insert', 44,  '        return lista, 0\n',                false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 10300, 'insert', 63,  '    medio = len(lista) // 2\n',            false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 13800, 'insert', 87,  '    izq, c1 = merge_sort(lista[:medio])\n', false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 17100, 'insert', 118, '    der, c2 = merge_sort(lista[medio:])\n',  false, NOW() - INTERVAL '5 hours'),
(gen_random_uuid(), 'a3000000-0000-0000-0000-000000000024', 22400, 'insert', 151, '    while i < len(izq) and j < len(der):\n', false, NOW() - INTERVAL '5 hours');

-- ============================================================================
-- B7. Comentarios del docente en SpeedGrader (submission_comments)
-- ============================================================================
INSERT INTO submission_comments (id, tenant_id, submission_id, author_id, line_number, comment, created_at)
VALUES
(
  '92000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'a3000000-0000-0000-0000-000000000034', -- Mateo, P1 REST
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  8,
  'El subtotal está bien, pero falta aplicar el IVA del 13% sobre el total. Sumás precio por precio sin impuesto.',
  NOW() - INTERVAL '6 days'
),
(
  '92000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'a2000000-0000-0000-0000-000000000003', -- Mateo, P1 Búsqueda Binaria
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  6,
  'Revisá el caso de un arreglo con un solo elemento: inicializás high = len(arr) y te saltás el último índice. Usá len(arr) - 1.',
  NOW() - INTERVAL '1 day'
),
(
  '92000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'a2000000-0000-0000-0000-000000000005', -- Daniel, P1 (caso de plagio)
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  4,
  'El flujo de búsqueda está correcto, pero la similitud con la entrega de Camila Rojas es alta (78%). Lo revisamos en clase.',
  NOW() - INTERVAL '4 hours'
),
(
  '92000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'a3000000-0000-0000-0000-000000000016', -- Lucas, P2 (AST bloqueado)
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  1,
  'No se permite os.system en este entorno (política AST). Reemplazá la ejecución de comandos por estructuras de datos puras.',
  NOW() - INTERVAL '6 hours'
),
(
  '92000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  'a2000000-0000-0000-0000-000000000001', -- Valeria, P1
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  12,
  'Solución limpia, O(log n) y casos límite cubiertos. Excelente.',
  NOW() - INTERVAL '5 days'
)
ON CONFLICT (id) DO UPDATE
SET comment = EXCLUDED.comment;

-- ============================================================================
-- B8. run_metrics (analítica del curso: casos más fallados, tiempos)
-- ============================================================================
INSERT INTO run_metrics (id, exercise_id, language, image_digest, duration_ms, verdict, case_index, created_at)
VALUES
  -- SIS-211 P1 Búsqueda Binaria (exercise 001, 4 cases)
  ('91000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'python', '', 38, 'AC',  1, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000001', 'python', '', 35, 'AC',  2, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000001', 'python', '', 41, 'AC',  3, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000004', 'e0000000-0000-0000-0000-000000000001', 'python', '', 33, 'AC',  4, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000005', 'e0000000-0000-0000-0000-000000000001', 'python', '', 36, 'AC',  1, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000006', 'e0000000-0000-0000-0000-000000000001', 'python', '', 40, 'AC',  2, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000007', 'e0000000-0000-0000-0000-000000000001', 'python', '', 42, 'AC',  3, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000008', 'e0000000-0000-0000-0000-000000000001', 'python', '', 37, 'AC',  4, NOW() - INTERVAL '5 days'),
  ('91000000-0000-0000-0000-000000000009', 'e0000000-0000-0000-0000-000000000001', 'python', '', 39, 'AC',  1, NOW() - INTERVAL '4 days'),
  ('91000000-0000-0000-0000-000000000010', 'e0000000-0000-0000-0000-000000000001', 'python', '', 44, 'WA',  2, NOW() - INTERVAL '4 days'),
  ('91000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000001', 'python', '', 45, 'WA',  3, NOW() - INTERVAL '4 days'),
  ('91000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000001', 'python', '', 40, 'WA',  4, NOW() - INTERVAL '4 days'),
  ('91000000-0000-0000-0000-000000000013', 'e0000000-0000-0000-0000-000000000001', 'python', '', 2150, 'TLE', 3, NOW() - INTERVAL '18 hours'),
  ('91000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000001', 'python', '', 2240, 'TLE', 4, NOW() - INTERVAL '18 hours'),
  ('91000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000001', 'python', '', 42, 'WA',  1, NOW() - INTERVAL '4 hours'),
  -- SIS-211 P3 Ordenamiento (exercise 022, 3 cases)
  ('91000000-0000-0000-0000-000000000016', 'e0000000-0000-0000-0000-000000000022', 'python', '', 152, 'AC', 1, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000017', 'e0000000-0000-0000-0000-000000000022', 'python', '', 148, 'AC', 2, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000018', 'e0000000-0000-0000-0000-000000000022', 'python', '', 155, 'AC', 3, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000019', 'e0000000-0000-0000-0000-000000000022', 'python', '', 160, 'AC', 1, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000022', 'python', '', 158, 'AC', 2, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000021', 'e0000000-0000-0000-0000-000000000022', 'python', '', 171, 'WA', 3, NOW() - INTERVAL '1 day'),
  ('91000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000022', 'python', '', 175, 'AC', 1, NOW() - INTERVAL '20 hours'),
  ('91000000-0000-0000-0000-000000000023', 'e0000000-0000-0000-0000-000000000022', 'python', '', 173, 'AC', 2, NOW() - INTERVAL '20 hours'),
  ('91000000-0000-0000-0000-000000000024', 'e0000000-0000-0000-0000-000000000022', 'python', '', 190, 'WA', 3, NOW() - INTERVAL '20 hours'),
  ('91000000-0000-0000-0000-000000000025', 'e0000000-0000-0000-0000-000000000022', 'python', '', 158, 'AC', 1, NOW() - INTERVAL '5 hours'),
  ('91000000-0000-0000-0000-000000000026', 'e0000000-0000-0000-0000-000000000022', 'python', '', 155, 'AC', 2, NOW() - INTERVAL '5 hours'),
  ('91000000-0000-0000-0000-000000000027', 'e0000000-0000-0000-0000-000000000022', 'python', '', 160, 'AC', 3, NOW() - INTERVAL '5 hours'),
  ('91000000-0000-0000-0000-000000000028', 'e0000000-0000-0000-0000-000000000022', 'python', '', 998, 'RE', 1, NOW() - INTERVAL '4 hours'),
  -- SIS-420 Examen Parcial (exercise 013, 3 cases)
  ('91000000-0000-0000-0000-000000000029', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 22, 'AC', 1, NOW() - INTERVAL '3 hours'),
  ('91000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 24, 'AC', 2, NOW() - INTERVAL '3 hours'),
  ('91000000-0000-0000-0000-000000000031', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 21, 'AC', 3, NOW() - INTERVAL '3 hours'),
  ('91000000-0000-0000-0000-000000000032', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 23, 'AC', 1, NOW() - INTERVAL '2 hours'),
  ('91000000-0000-0000-0000-000000000033', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 26, 'WA', 2, NOW() - INTERVAL '2 hours'),
  ('91000000-0000-0000-0000-000000000034', 'e0000000-0000-0000-0000-000000000013', 'typescript', '', 25, 'WA', 3, NOW() - INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;
-- ============================================================================
-- B9. evaluation_generated_cases (Examen Parcial con seed por alumno)
-- ============================================================================
INSERT INTO evaluation_generated_cases (exercise_id, student_id, tenant_id, seed, cases, created_at)
VALUES
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132af8', '00000000-0000-0000-0000-000000000001', 101,
   '[{"input":"Authorization: Bearer tok-u1-aa11","expected":"true"},{"input":"Authorization: Bearer tok-falso","expected":"false"},{"input":"sin header","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afc', '00000000-0000-0000-0000-000000000001', 102,
   '[{"input":"Authorization: Bearer tok-u2-bb22","expected":"true"},{"input":"Authorization: Basic dXNlcg==","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afa', '00000000-0000-0000-0000-000000000001', 103,
   '[{"input":"Authorization: Bearer tok-u3-cc33","expected":"true"},{"input":"Authorization: Bearer","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132af9', '00000000-0000-0000-0000-000000000001', 104,
   '[{"input":"Authorization: Bearer tok-u4-dd44","expected":"true"},{"input":"sin header","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afb', '00000000-0000-0000-0000-000000000001', 105,
   '[{"input":"Authorization: Bearer tok-u5-ee55","expected":"true"},{"input":"Authorization: Bearer tok-u5-ee55 extra","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afd', '00000000-0000-0000-0000-000000000001', 106,
   '[{"input":"Authorization: Bearer tok-u6-ff66","expected":"true"},{"input":"Authorization: Bearer tok-viejo","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afe', '00000000-0000-0000-0000-000000000001', 107,
   '[{"input":"Authorization: Bearer tok-u7-aabb","expected":"true"},{"input":"sin header","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132b01', '00000000-0000-0000-0000-000000000001', 108,
   '[{"input":"Authorization: Bearer tok-u8-ccdd","expected":"true"},{"input":"Authorization: Basic eA==","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days'),
  ('e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132b02', '00000000-0000-0000-0000-000000000001', 109,
   '[{"input":"Authorization: Bearer tok-u9-eeff","expected":"true"},{"input":"Authorization: Bearer","expected":"false"}]'::jsonb, NOW() - INTERVAL '10 days')
ON CONFLICT (exercise_id, student_id) DO UPDATE
SET seed = EXCLUDED.seed, cases = EXCLUDED.cases;

-- ============================================================================
-- B10. deadline_reminders_sent (alumnos en riesgo del Examen Parcial)
-- ============================================================================
INSERT INTO deadline_reminders_sent (id, tenant_id, exercise_id, student_id, sent_at)
VALUES
  ('93000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132afe', NOW() - INTERVAL '8 hours'),
  ('93000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132b01', NOW() - INTERVAL '6 hours'),
  ('93000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000013', 'e30c0f6d-ff63-4392-adcf-6336d6132b02', NOW() - INTERVAL '6 hours')
ON CONFLICT (tenant_id, exercise_id, student_id) DO NOTHING;

-- ============================================================================
-- B11. host_metrics_samples (telemetría del nodo, últimas horas)
-- ============================================================================
DELETE FROM host_metrics_samples WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
INSERT INTO host_metrics_samples (sampled_at, ram_percent, ram_used_gb, cpu_percent, active_containers, tenant_id)
SELECT
  NOW() - (i || ' minutes')::interval,
  CASE WHEN i IN (8, 9) THEN 92.0 + (i % 3) * 1.5 ELSE 58.0 + (i % 7) * 2.0 END,
  CASE WHEN i IN (8, 9) THEN 18.4 + (i % 3) * 0.3 ELSE 11.4 + (i % 7) * 0.4 END,
  22.0 + ((i * 37) % 45),
  3 + (i % 3),
  '00000000-0000-0000-0000-000000000001'
FROM generate_series(1, 24) AS i;

-- ============================================================================
-- B12. Notificaciones del docente (campana del topbar)
-- ============================================================================
INSERT INTO notifications (
  id, tenant_id, recipient_user_id, channel, severity, title, message,
  event_type, metadata, is_read, read_at, occurrence_count, link, created_at
) VALUES
(
  '94000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'critical',
  'Contenedor excedió el límite de memoria',
  'El entorno de Lucas Benítez (SIS-211) fue detenido por OOM. El workspace quedó hibernado.',
  'workspace.oom_killed',
  '{"workspace_id": "c2000000-0000-0000-0000-000000000003", "student": "Lucas Benítez", "subject": "SIS-211"}'::jsonb,
  false, null, 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000002',
  NOW() - INTERVAL '2 hours'
),
(
  '94000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'warning',
  'Entrega pendiente de revisión',
  'Daniel Quispe envió "Práctica 3: Ordenamiento Merge Sort" y quedó sin calificar en la cola.',
  'submission.pending_review',
  '{"submission_id": "a3000000-0000-0000-0000-000000000024"}'::jsonb,
  false, null, 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000002',
  NOW() - INTERVAL '5 hours'
),
(
  '94000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'warning',
  'Entrega bloqueada por política AST',
  'Lucas Benítez usó os.system en "Práctica 2: Pilas y Colas". La entrega quedó retenida para revisión.',
  'submission.ast_blocked',
  '{"submission_id": "a3000000-0000-0000-0000-000000000016", "rule": "python.import-forbidden"}'::jsonb,
  false, null, 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000002',
  NOW() - INTERVAL '6 hours'
),
(
  '94000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'info',
  'Plazo del Examen Parcial',
  'Quedan menos de 24 horas para el Examen Parcial (SIS-420). 3 estudiantes aún no entregaron.',
  'deadline.reminder',
  '{"exercise_id": "e0000000-0000-0000-0000-000000000013", "sin_entrega": 3}'::jsonb,
  false, null, 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000001',
  NOW() - INTERVAL '8 hours'
),
(
  '94000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'info',
  'Auditoría completada',
  'Auditaste la entrega de Daniel Quispe en "Práctica 1: Endpoints REST" y confirmaste el veredicto.',
  'submission.audited',
  '{"submission_id": "a3000000-0000-0000-0000-000000000036"}'::jsonb,
  true, NOW() - INTERVAL '1 day', 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000001',
  NOW() - INTERVAL '1 day'
),
(
  '94000000-0000-0000-0000-000000000006',
  '00000000-0000-0000-0000-000000000001',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'in_app', 'info',
  'Alumnos sin actividad',
  '3 alumnos no registran actividad en los últimos 7 días en SIS-420 (Desarrollo Web Fullstack).',
  'student.inactivity',
  '{"subject_id": "50000000-0000-0000-0000-000000000001", "count": 3}'::jsonb,
  true, NOW() - INTERVAL '2 days', 1,
  '/teacher/courses/50000000-0000-0000-0000-000000000001',
  NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title, message = EXCLUDED.message, severity = EXCLUDED.severity,
    is_read = EXCLUDED.is_read, read_at = EXCLUDED.read_at;

-- ============================================================================
-- B13. Audit logs (actividad del docente Carlos)
-- ============================================================================
INSERT INTO audit_logs (
  id, tenant_id, actor_id, action, resource_type, resource_id, status_code,
  metadata, ip_address, user_agent, created_at
) VALUES
  ('95000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'EXERCISE_CREATED', 'exercise', 'e0000000-0000-0000-0000-000000000011', 201, '{"title": "Práctica 2: Consultas con JOIN y Agregados"}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '20 days'),
  ('95000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'EXERCISE_PUBLISHED', 'exercise', 'e0000000-0000-0000-0000-000000000011', 200, '{"title": "Práctica 2: Consultas con JOIN y Agregados"}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '19 days'),
  ('95000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'EXERCISE_CREATED', 'exercise', 'e0000000-0000-0000-0000-000000000013', 201, '{"title": "Examen Parcial: API REST con autenticación"}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '12 days'),
  ('95000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'EXERCISE_PUBLISHED', 'exercise', 'e0000000-0000-0000-0000-000000000013', 200, '{"title": "Examen Parcial: API REST con autenticación"}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '10 days'),
  ('95000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'SUBMISSION_REVIEWED', 'submission', 'a2000000-0000-0000-0000-000000000003', 200, '{"verdict": "WA", "score": 50}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '1 day'),
  ('95000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'SUBMISSION_OVERRIDDEN', 'submission', 'a3000000-0000-0000-0000-000000000034', 200, '{"verdict": "WA", "score": 40, "reason": "omite IVA del 13%"}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '6 days'),
  ('95000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'GRADES_EXPORTED', 'subject', '50000000-0000-0000-0000-000000000001', 200, '{"format": "csv", "rows": 9}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '4 days'),
  ('95000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'COURSE_WORKSPACES_PAUSED', 'subject', '50000000-0000-0000-0000-000000000002', 200, '{"workspaces": 4}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '2 days'),
  ('95000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'SUBMISSION_COMMENTED', 'submission', 'a3000000-0000-0000-0000-000000000016', 200, '{"line": 1}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '6 hours'),
  ('95000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '0e5e6bde-1933-4084-82b0-d481b4309ca7', 'EVALUATOR_RUN', 'exercise', 'e0000000-0000-0000-0000-000000000013', 200, '{"dry_run": true, "cases": 3}'::jsonb, '10.0.0.5', 'Mozilla/5.0 (X11; Linux x86_64) Chrome/126.0', NOW() - INTERVAL '4 days')
ON CONFLICT (id) DO UPDATE
SET action = EXCLUDED.action, metadata = EXCLUDED.metadata, created_at = EXCLUDED.created_at;

COMMIT;

-- ============================================================================
-- B13b. Normalización del entorno IDE (remediación idempotente de demo)
-- Los laboratorios IDE_PERSISTENTE se corrigen con rúbrica manual en
-- SpeedGrader: no pasan por el juez I/O. Las submissions con veredictos
-- automáticos (AC/WA/RE/TLE/AST_BLOCKED) y casos de prueba de ejercicios IDE
-- eran data de demo; se normalizan a la semántica real (PENDING y sin casos).
-- Re-ejecutable sin efectos secundarios.
-- ============================================================================
UPDATE submissions su
SET verdict = 'PENDING',
    score = 0,
    execution_time_ms = 0,
    memory_used_mb = 0,
    graded_by = NULL,
    manual_override = FALSE,
    override_reason = ''
FROM exercises e
WHERE su.exercise_id = e.id
  AND e.environment_type = 'IDE_PERSISTENTE'
  AND su.verdict <> 'PENDING';

DELETE FROM exercise_test_cases tc
USING exercises e
WHERE tc.exercise_id = e.id
  AND e.environment_type = 'IDE_PERSISTENTE';

-- Type canónico según entorno (aliado de la migración 00033)
UPDATE exercises SET type = 'project'
WHERE type = 'code' AND environment_type = 'IDE_PERSISTENTE';

UPDATE exercises SET type = 'algorithm'
WHERE type = 'code'
  AND (environment_type = 'JUEZ_EFIMERO' OR environment_type IS NULL OR environment_type = '');

-- ============================================================================
-- B14. Verificación (para validación visual)
-- ============================================================================
-- 1. Resumen por curso de Carlos:
--    SELECT s.name, COUNT(DISTINCT e.student_id) AS alumnos FROM subjects s
--    JOIN enrollments e ON e.subject_id = s.id WHERE s.teacher_id = '0e5e6bde-1933-4084-82b0-d481b4309ca7'
--    GROUP BY s.name;
-- 2. Entregas pendientes (cola de revisión):
--    SELECT u.first_name, ex.title, sub.verdict FROM submissions sub
--    JOIN exercises ex ON ex.id = sub.exercise_id JOIN users u ON u.id = sub.student_id
--    WHERE sub.graded_by IS NULL ORDER BY sub.submitted_at DESC;
-- 3. Alerta crítica (OOM) de workspaces:
--    SELECT w.status, u.first_name FROM workspaces w JOIN subjects s ON s.id = w.subject_id
--    LEFT JOIN users u ON u.id = w.student_id
--    WHERE s.teacher_id = '0e5e6bde-1933-4084-82b0-d481b4309ca7' AND w.status IN ('oom_killed','failed');
-- 4. "Activos ahora" (heartbeat < 2 min):
--    SELECT u.first_name, w.subject_id FROM workspaces w JOIN users u ON u.id = w.student_id
--    WHERE w.last_heartbeat_at >= NOW() - INTERVAL '2 minutes';
-- 5. "En riesgo" (Examen Parcial con plazo < 24h):
--    SELECT ex.title, ex.due_date FROM exercises ex
--    WHERE ex.due_date > NOW() AND ex.due_date <= NOW() + INTERVAL '24 hours';
