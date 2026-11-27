// Soak de 30 minutos: 15 VUs de aula + consultas de estado y envíos espaciados.
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
const readTotal = new Counter('read_total');
const evalTotal = new Counter('eval_total');
const submitCompleted = new Counter('submit_completed');

export const options = {
  scenarios: {
    soak: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 15 },
        { duration: '29m', target: 15 },
        { duration: '30s', target: 0 },
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
  readTotal.add(1);
  statusCounter.add(1, { phase: 'read', code: String(read.status) });
  if (read.status === 200) readDuration.add(read.timings.duration);
  check(read, { 'lectura 200': (r) => r.status === 200 });
  sleep(3 + Math.random() * 2);

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
      let subId = '';
      try {
        const body = JSON.parse(subRes.body);
        subId = body.id || (body.data && body.data.id) || '';
      } catch (e) { /* sin id */ }
      if (subId) {
        const statusRes = http.get(`${BASE_URL}/api/v1/submissions/${subId}`, {
          headers, tags: { name: 'submission_status' }, timeout: '60s',
        });
        statusCounter.add(1, { phase: 'submission_status', code: String(statusRes.status) });
      }
    }
  }
  sleep(12 + Math.random() * 6);
}

export function handleSummary(data) {
  return { [__ENV.SUMMARY_OUT || 'soak_summary.json']: JSON.stringify(data, null, 2) };
}
