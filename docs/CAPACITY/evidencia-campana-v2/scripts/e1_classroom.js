// E1 — escalonado realista de aula. Un VU = un alumno distinto del pool.
// Lectura con think time 3–5 s; envío de evaluación cada 12–18 s por alumno.
// La rampa de 10 s evita el pico sintético de arranque simultáneo.
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const TOKENS = JSON.parse(open(__ENV.TOKENS_FILE || '../raw/tokens-campaign.json'));
const TOKEN_COUNT = Object.keys(TOKENS).length;
const BASE_URL = __ENV.BASE_URL || 'https://127.0.0.1';
const HOST_HEADER = __ENV.HOST_HEADER || 'solv.dedyn.io';
const VUS = Number(__ENV.VUS || 10);
const DURATION = __ENV.DURATION || '60s';
const RAMP = __ENV.RAMP || '10s';
const TIMEOUT = __ENV.TIMEOUT || '60s';
const CODE_B64 = 'cHJpbnQoc3VtKG1hcChpbnQsIGlucHV0KCkuc3BsaXQoKSkpKQ==';
const CODE_RAW = 'print(sum(map(int, input().split())))';

const readDuration = new Trend('read_duration', true);
const submitDuration = new Trend('submit_duration', true);
const readTotal = new Counter('read_total');
const readOk = new Counter('read_ok');
const evalTotal = new Counter('eval_total');
const evalOk = new Counter('eval_ok');
const submitTotal = new Counter('submit_total');
const submitCompleted = new Counter('submit_completed');
const statusCounter = new Counter('resp_status');

export const options = {
  scenarios: {
    classroom: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: RAMP, target: VUS },
        { duration: DURATION, target: VUS },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    read_duration: ['p(95)<2000'],
    submit_duration: ['p(95)<60000'],
  },
  insecureSkipTLSVerify: true,
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

function headersFor(token) {
  return {
    'Content-Type': 'application/json',
    Host: HOST_HEADER,
    Cookie: `solv_session=${token}`,
  };
}

function tag(res, phase) {
  statusCounter.add(1, { phase, code: String(res.status) });
}

export default function () {
  const index = ((__VU - 1) % TOKEN_COUNT) + 1;
  const student = TOKENS[String(index)];
  if (!student) return;
  const headers = headersFor(student.token);

  const exerciseNum = ((__ITER + __VU) % 10) + 1;
  const exerciseId = `f0000000-0000-0000-0000-0000000000${String(exerciseNum).padStart(2, '0')}`;

  // 1. Lectura de ejercicio (contenido y enunciado).
  const read = http.get(`${BASE_URL}/api/v1/exercises/${exerciseId}`, {
    headers, tags: { name: 'read' }, timeout: TIMEOUT,
  });
  readTotal.add(1);
  tag(read, 'read');
  if (read.status === 200) {
    readDuration.add(read.timings.duration);
    readOk.add(1);
  }
  check(read, { 'lectura 200': (r) => r.status === 200 });
  sleep(3 + Math.random() * 2);

  // 2. Envío de evaluación: recibo autoritativo (ADR-066) + entrega + estado.
  const evalRes = http.post(`${BASE_URL}/api/v1/evaluations`, JSON.stringify({
    exercise_id: exerciseId,
    language: 'python',
    source_code_b64: CODE_B64,
  }), { headers, tags: { name: 'evaluation' }, timeout: TIMEOUT });
  evalTotal.add(1);
  tag(evalRes, 'evaluation');
  let receipt = '';
  try {
    const body = JSON.parse(evalRes.body);
    receipt = (body.data && body.data.evaluation_receipt) || body.evaluation_receipt || '';
  } catch (e) { /* cuerpo no JSON (timeout o 429) */ }
  if (evalRes.status === 200 && receipt) evalOk.add(1);

  if (receipt) {
    submitTotal.add(1);
    const subRes = http.post(`${BASE_URL}/api/v1/submissions`, JSON.stringify({
      exercise_id: exerciseId,
      student_id: student.student_id,
      code: CODE_RAW,
      evaluation_receipt: receipt,
    }), { headers, tags: { name: 'submission' }, timeout: TIMEOUT });
    tag(subRes, 'submission');
    if (subRes.status === 201) {
      submitDuration.add(subRes.timings.duration);
      submitCompleted.add(1);
      let subId = '';
      try {
        const body = JSON.parse(subRes.body);
        subId = body.id || (body.data && body.data.id) || '';
      } catch (e) { /* sin id */ }
      if (subId) {
        const statusRes = http.get(`${BASE_URL}/api/v1/submissions/${subId}`, {
          headers, tags: { name: 'submission_status' }, timeout: TIMEOUT,
        });
        tag(statusRes, 'submission_status');
      }
    }
  }

  // 3. Think time de entrega: 12–18 s entre envíos del mismo alumno.
  sleep(12 + Math.random() * 6);
}

export function handleSummary(data) {
  const out = __ENV.SUMMARY_OUT || 'summary.json';
  return { [out]: JSON.stringify(data, null, 2) };
}
