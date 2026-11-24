import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

// Tokens sembrados y pre-generados con firma JWT real (ADR-089, Trampa #4 y #5)
const tokensData = JSON.parse(open('./tokens.json'));

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Fase 1: 50 usuarios concurrentes
    { duration: '45s', target: 100 },  // Fase 2: 100 usuarios (carga media)
    { duration: '45s', target: 200 },  // Fase 3: 200 usuarios (pico de deadline)
    { duration: '20s', target: 0 },    // Cool-down
  ],
  thresholds: {
    'http_req_duration{name:get_exercise}': ['p(95)<2000'],
    'http_req_duration{name:post_evaluations}': ['p(95)<15000'],
    'http_req_duration{name:post_submissions}': ['p(95)<5000'],
    'http_req_duration{name:get_submission}': ['p(95)<2000'],
    'http_req_failed': ['rate<0.10'],
    errors: ['rate<0.10'],
  },
  insecureSkipTLSVerify: true,
};

const BASE_URL = 'https://127.0.0.1';
const HOST_HEADER = 'solv.dedyn.io';
const CODE_B64 = 'cHJpbnQoc3VtKG1hcChpbnQsIGlucHV0KCkuc3BsaXQoKSkpKQ==';
const CODE_RAW = 'print(sum(map(int, input().split())))';

export default function () {
  const vuIndex = String(((__VU - 1) % 200) + 1);
  const studentEntry = tokensData[vuIndex];
  const token = studentEntry ? studentEntry.token : '';
  const studentId = studentEntry ? studentEntry.student_id : `00000000-0000-0000-0001-${String(vuIndex).padStart(12, '0')}`;

  const exerciseNum = ((__ITER + __VU) % 10) + 1;
  const exerciseId = `e0000000-0000-0000-0000-0000000000${String(exerciseNum).padStart(2, '0')}`;

  const headers = {
    'Content-Type': 'application/json',
    'Host': HOST_HEADER,
    'Cookie': `solv_session=${token}`,
  };

  // 1. Obtener detalle del ejercicio
  const getExRes = http.get(`${BASE_URL}/api/v1/exercises/${exerciseId}`, {
    headers,
    tags: { name: 'get_exercise' },
  });
  const getExOk = check(getExRes, {
    'get exercise status 200': (r) => r.status === 200,
  });

  // 2. Solicitar evaluación autoritativa (ADR-066, Trampa #3)
  const evalPayload = JSON.stringify({
    exercise_id: exerciseId,
    language: 'python',
    source_code_b64: CODE_B64,
  });

  const evalRes = http.post(`${BASE_URL}/api/v1/evaluations`, evalPayload, {
    headers,
    tags: { name: 'post_evaluations' },
  });
  const evalOk = check(evalRes, {
    'evaluation status 200': (r) => r.status === 200,
  });

  let receipt = '';
  try {
    const evalBody = JSON.parse(evalRes.body);
    receipt = (evalBody.data && evalBody.data.evaluation_receipt) || evalBody.evaluation_receipt || '';
  } catch (e) {}

  // 3. Enviar entrega con recibo de evaluación firmado
  let submitOk = false;
  let subId = '';
  if (receipt) {
    const subPayload = JSON.stringify({
      exercise_id: exerciseId,
      student_id: studentId,
      code: CODE_RAW,
      evaluation_receipt: receipt,
    });

    const submitRes = http.post(`${BASE_URL}/api/v1/submissions`, subPayload, {
      headers,
      tags: { name: 'post_submissions' },
    });
    submitOk = check(submitRes, {
      'submission created 201': (r) => r.status === 201,
    });

    try {
      const subBody = JSON.parse(submitRes.body);
      subId = subBody.id || (subBody.data && subBody.data.id) || '';
    } catch (e) {}
  }

  // 4. Consultar resultado de submission (GET /api/v1/submissions/:id)
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

  const hasError = !getExOk || !evalOk || !submitOk || !getSubOk;
  errorRate.add(hasError);

  // Think-time de 6 a 8 segundos para respetar el límite de 10 envíos/min (Trampa #2)
  sleep(Math.random() * 2 + 6);
}
