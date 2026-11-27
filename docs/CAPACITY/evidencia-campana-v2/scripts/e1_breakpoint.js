// Saturación / punto de quiebre del flujo síncrono de evaluación.
// Rampa 40 → 60 → 100 VUs sobre el mismo flujo autenticado de E1.
// Se detiene por sí sola al agotar la rampa; no se usa como carga de aula.
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const TOKENS = JSON.parse(open(__ENV.TOKENS_FILE || '../raw/tokens-campaign.json'));
const TOKEN_COUNT = Object.keys(TOKENS).length;
const BASE_URL = __ENV.BASE_URL || 'https://127.0.0.1';
const HOST_HEADER = __ENV.HOST_HEADER || 'solv.dedyn.io';
const CODE_B64 = 'cHJpbnQoc3VtKG1hcChpbnQsIGlucHV0KCkuc3BsaXQoKSkpKQ==';

const readDuration = new Trend('read_duration', true);
const submitDuration = new Trend('submit_duration', true);
const evalDuration = new Trend('eval_duration', true);
const statusCounter = new Counter('resp_status');
const evalTotal = new Counter('eval_total');
const evalOk = new Counter('eval_ok');
const submitCompleted = new Counter('submit_completed');

export const options = {
  scenarios: {
    breakpoint: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 40 },
        { duration: '45s', target: 40 },
        { duration: '30s', target: 60 },
        { duration: '45s', target: 60 },
        { duration: '40s', target: 100 },
        { duration: '60s', target: 100 },
        { duration: '20s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  insecureSkipTLSVerify: true,
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

export default function () {
  const index = ((__VU - 1) % TOKEN_COUNT) + 1;
  const student = TOKENS[String(index)];
  if (!student) return;
  const headers = {
    'Content-Type': 'application/json',
    Host: HOST_HEADER,
    Cookie: `solv_session=${student.token}`,
  };
  const exerciseNum = ((__ITER + __VU) % 10) + 1;
  const exerciseId = `f0000000-0000-0000-0000-0000000000${String(exerciseNum).padStart(2, '0')}`;

  const read = http.get(`${BASE_URL}/api/v1/exercises/${exerciseId}`, {
    headers, tags: { name: 'read' }, timeout: '60s',
  });
  statusCounter.add(1, { phase: 'read', code: String(read.status) });
  if (read.status === 200) readDuration.add(read.timings.duration);
  check(read, { 'lectura 200': (r) => r.status === 200 });

  const evalRes = http.post(`${BASE_URL}/api/v1/evaluations`, JSON.stringify({
    exercise_id: exerciseId, language: 'python', source_code_b64: CODE_B64,
  }), { headers, tags: { name: 'evaluation' }, timeout: '60s' });
  evalTotal.add(1);
  statusCounter.add(1, { phase: 'evaluation', code: String(evalRes.status) });
  evalDuration.add(evalRes.timings.duration);
  let receipt = '';
  try {
    const body = JSON.parse(evalRes.body);
    receipt = (body.data && body.data.evaluation_receipt) || body.evaluation_receipt || '';
  } catch (e) { /* timeout */ }
  if (evalRes.status === 200 && receipt) evalOk.add(1);

  if (receipt) {
    const subRes = http.post(`${BASE_URL}/api/v1/submissions`, JSON.stringify({
      exercise_id: exerciseId,
      student_id: student.student_id,
      code: 'print(sum(map(int, input().split())))',
      evaluation_receipt: receipt,
    }), { headers, tags: { name: 'submission' }, timeout: '60s' });
    statusCounter.add(1, { phase: 'submission', code: String(subRes.status) });
    if (subRes.status === 201) {
      submitDuration.add(subRes.timings.duration);
      submitCompleted.add(1);
    }
  }
  // Sin think time largo: esta corrida busca el techo, no simular aula.
  sleep(2 + Math.random() * 2);
}

export function handleSummary(data) {
  return { [__ENV.SUMMARY_OUT || 'breakpoint.json']: JSON.stringify(data, null, 2) };
}
