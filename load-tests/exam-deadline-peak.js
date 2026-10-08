import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '2m', target: 50 },   // Fase 1: 50 usuarios (calentamiento)
    { duration: '3m', target: 100 },  // Fase 2: 100 usuarios (carga media)
    { duration: '5m', target: 200 },  // Fase 3: 200 usuarios (pico de deadline)
    { duration: '2m', target: 0 },    // Cool-down
  ],
  thresholds: {
    'http_req_duration{name:post_submissions}': ['p(95)<3000'], // P95 < 3 segundos
    'http_req_duration{name:get_exercise}': ['p(95)<1000'],
    'http_req_duration{name:get_submission}': ['p(95)<1000'],
    'http_req_failed': ['rate<0.01'],                          // Errores HTTP < 1%
    errors: ['rate<0.01'],                                     // Tasa de fallos lógicos < 1%
  },
};

const BASE_URL = 'http://127.0.0.1:3000';
const TENANT_ID = '00000000-0000-0000-0000-000000000001';

export default function () {
  const vuIndex = (__VU % 200) + 1;
  const studentId = `00000000-0000-0000-0001-${String(vuIndex).padStart(12, '0')}`;
  const exerciseNum = ((__ITER + __VU) % 10) + 1;
  const exerciseId = `e0000000-0000-0000-0000-0000000000${String(exerciseNum).padStart(2, '0')}`;

  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-Id': TENANT_ID,
    'X-User-Id': studentId,
    'X-User-Role': 'student',
  };

  // 1. Obtener detalle del ejercicio
  const getExRes = http.get(`${BASE_URL}/api/v1/exercises/${exerciseId}`, {
    headers,
    tags: { name: 'get_exercise' },
  });
  const getExOk = check(getExRes, {
    'get exercise status 200': (r) => r.status === 200,
  });

  // 2. Enviar código Python de solución (POST /api/v1/submissions)
  const subPayload = JSON.stringify({
    exercise_id: exerciseId,
    student_id: studentId,
    code: 'print(sum(map(int, input().split())))',
    verdict: 'AC',
    execution_time_ms: 12,
    memory_used_mb: 14,
    ast_result: { nodes_count: 15 },
    generated_cases: {},
    complexity_analysis: { time: 'O(N)', space: 'O(1)' },
  });

  const submitRes = http.post(`${BASE_URL}/api/v1/submissions`, subPayload, {
    headers,
    tags: { name: 'post_submissions' },
  });
  const submitOk = check(submitRes, {
    'submission created 201': (r) => r.status === 201,
  });

  let subId = '';
  try {
    const subBody = JSON.parse(submitRes.body);
    subId = subBody.id || (subBody.data && subBody.data.id) || '';
  } catch (e) {}

  // 3. Consultar resultado de submission (GET /api/v1/submissions/:id)
  let getSubOk = true;
  if (subId) {
    const getSubRes = http.get(`${BASE_URL}/api/v1/submissions/${subId}`, {
      headers,
      tags: { name: 'get_submission' },
    });
    getSubOk = check(getSubRes, {
      'get submission status 200': (r) => r.status === 200,
    });
  }

  const hasError = !getExOk || !submitOk || !getSubOk;
  errorRate.add(hasError);

  // Think-time de 2 a 5 segundos según especificación
  sleep(Math.random() * 3 + 2);
}
