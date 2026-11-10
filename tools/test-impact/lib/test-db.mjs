const LOOPBACK_HOSTS = new Set(['127.0.0.1', 'localhost', '::1']);
const TEST_PORTS = new Set(['5433', '55432']);
const TEST_DB_NAME = /^solv_(test|migration_test|e2e)(_[a-z0-9_]+)?$/;
const AMBIGUOUS_QUERY_KEYS = new Set([
  'dbname',
  'host',
  'port',
  'user',
  'password',
  'service',
  'passfile'
]);

function decoded(value, label) {
  try {
    return decodeURIComponent(value);
  } catch {
    throw new Error(`${label} del DSN de tests no se puede decodificar`);
  }
}

export function validateTestDatabaseURL(dsn) {
  if (!String(dsn || '').trim()) {
    throw new Error(
      'la suite requiere TEST_DB_DSN o TEST_DATABASE_URL; no hay DSN de tests predeterminado ni se usa DATABASE_URL'
    );
  }

  let parsed;
  try {
    parsed = new URL(dsn);
  } catch {
    throw new Error('el DSN de tests no es una URL válida');
  }
  if (parsed.protocol !== 'postgres:' && parsed.protocol !== 'postgresql:') {
    throw new Error(`el DSN de tests debe usar postgres://, no ${parsed.protocol}`);
  }

  const host = parsed.hostname.replace(/^\[|\]$/g, '');
  if (!LOOPBACK_HOSTS.has(host)) {
    throw new Error(`host de base de tests no permitido: ${parsed.hostname}`);
  }
  const port = parsed.port || '5432';
  if (!TEST_PORTS.has(port)) {
    throw new Error(`puerto de base de tests no permitido: ${port}`);
  }

  const name = decoded(parsed.pathname.replace(/^\//, ''), 'nombre');
  if (!name || name.includes('/') || !TEST_DB_NAME.test(name)) {
    throw new Error(`nombre de base de tests no permitido: "${name}"`);
  }
  if (name === 'solv_db' || name === 'solv_dev_db') {
    throw new Error(`nombre de base de tests no permitido: "${name}"`);
  }

  for (const key of parsed.searchParams.keys()) {
    if (AMBIGUOUS_QUERY_KEYS.has(key.toLowerCase())) {
      throw new Error(`el DSN de tests no puede sobrescribir "${key}" en la query`);
    }
  }

  if (!decoded(parsed.username, 'usuario').trim() || !decoded(parsed.password, 'contraseña')) {
    throw new Error('el DSN de tests debe declarar usuario y contraseña explícitos');
  }

  return dsn;
}

export function requireTestDatabase(env = {}) {
  const dsn = String(env.TEST_DB_DSN || env.TEST_DATABASE_URL || '').trim();
  return validateTestDatabaseURL(dsn);
}