import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REPO_ROOT = path.resolve(__dirname, '../../..');

export const DEFAULT_TENANT_ID = '00000000-0000-0000-0000-000000000001';

// Identidades sintéticas compartidas con backend/seeds/e2e_actors.sql. No se
// usan correos ni nombres de personas reales.
export const CANONICAL_ACTORS = {
  admin: {
    id: 'e2e00000-0000-4000-8000-000000000001',
    email: 'admin@e2e.solv.test',
    role: 'admin',
    firstName: 'Admin',
    lastName: 'E2E',
    tenantId: DEFAULT_TENANT_ID
  },
  teacher: {
    id: 'e2e00000-0000-4000-8000-000000000002',
    email: 'teacher@e2e.solv.test',
    role: 'teacher',
    firstName: 'Docente',
    lastName: 'E2E',
    tenantId: DEFAULT_TENANT_ID
  },
  student: {
    id: 'e2e00000-0000-4000-8000-000000000003',
    email: 'student@e2e.solv.test',
    role: 'student',
    firstName: 'Estudiante',
    lastName: 'E2E',
    tenantId: DEFAULT_TENANT_ID
  }
};

function base64UrlEncode(input) {
  const buf = Buffer.isBuffer(input) ? input : Buffer.from(String(input), 'utf8');
  return buf
    .toString('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
}

export function resolveJwtSecret() {
  if (process.env.JWT_SECRET && process.env.JWT_SECRET.trim()) {
    return process.env.JWT_SECRET.trim().replace(/^"|"$/g, '');
  }

  const candidatePaths = [
    path.join(REPO_ROOT, '.env'),
    path.join(REPO_ROOT, 'backend', '.env')
  ];

  for (const envPath of candidatePaths) {
    if (!fs.existsSync(envPath)) continue;
    const raw = fs.readFileSync(envPath, 'utf8');
    for (const line of raw.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) continue;
      const eqIdx = trimmed.indexOf('=');
      if (eqIdx === -1) continue;
      const key = trimmed.slice(0, eqIdx).trim();
      if (key === 'JWT_SECRET') {
        const val = trimmed.slice(eqIdx + 1).trim().replace(/^"|"$/g, '');
        if (val) return val;
      }
    }
  }

  throw new Error('JWT_SECRET not found in environment or .env');
}

export function signSolvJwt(actor, ttlSeconds = 86400) {
  const secret = resolveJwtSecret();
  const header = { alg: 'HS256', typ: 'JWT' };
  const payload = {
    user_id: actor.id,
    email: actor.email,
    role: actor.role,
    tenant_id: actor.tenantId || DEFAULT_TENANT_ID,
    exp: Math.floor(Date.now() / 1000) + ttlSeconds
  };

  const encodedHeader = base64UrlEncode(JSON.stringify(header));
  const encodedPayload = base64UrlEncode(JSON.stringify(payload));
  const signingInput = `${encodedHeader}.${encodedPayload}`;
  const signature = crypto
    .createHmac('sha256', Buffer.from(secret, 'utf8'))
    .update(signingInput)
    .digest();

  return `${signingInput}.${base64UrlEncode(signature)}`;
}

export async function createAuthenticatedRoleSession(browser, roleKey, baseURL = 'http://localhost:4200') {
  const actor = CANONICAL_ACTORS[roleKey];
  if (!actor) {
    throw new Error(`Unknown roleKey: ${roleKey}`);
  }

  return createAuthenticatedActorSession(browser, actor, baseURL);
}

/**
 * Crea una sesión autenticada para cualquier actor (por ejemplo, un estudiante
 * adicional creado durante el propio test para comparar semillas de examen).
 */
export async function createAuthenticatedActorSession(browser, actor, baseURL = 'http://localhost:4200') {
  const token = signSolvJwt(actor);
  const parsedUrl = new URL(baseURL);

  const context = await browser.newContext({
    baseURL,
    viewport: { width: 1440, height: 900 }
  });

  await context.addCookies([
    {
      name: 'solv_session',
      value: token,
      domain: parsedUrl.hostname,
      path: '/',
      httpOnly: true,
      sameSite: 'Lax'
    }
  ]);

  await context.addInitScript((jwtToken) => {
    window.sessionStorage.setItem('solv_token', jwtToken);
  }, token);

  const page = await context.newPage();
  // La sesión debe quedar en el origen de la app: sessionStorage no existe en
  // about:blank y varios flujos evaluan fetch con el token guardado.
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  return { context, page, actor, token };
}

/**
 * Registra un estudiante adicional desde la UI/API de administración. La
 * plataforma no permite crear estudiantes desde el rol docente, por eso el
 * registro lo hace la administración y luego se abre una sesión propia para
 * ese estudiante.
 */
export async function createStudentViaAdminAPI(adminPage, { email, firstName, lastName }) {
  if (!/^https?:\/\//.test(adminPage.url())) {
    await adminPage.goto('/');
  }

  const studentId = await adminPage.evaluate(
    async ({ email, firstName, lastName }) => {
      const token = window.sessionStorage.getItem('solv_token');
      const resp = await fetch('/api/v1/admin/students', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify({ email, first_name: firstName, last_name: lastName })
      });
      const json = await resp.json();
      const body = json?.data ?? json;
      return body?.id ?? '';
    },
    { email, firstName, lastName }
  );

  if (!studentId) {
    throw new Error(`No se pudo registrar el estudiante adicional ${email}`);
  }
  return studentId;
}
