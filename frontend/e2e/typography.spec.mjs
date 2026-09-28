import { expect, test } from '@playwright/test';

test('computed typography follows the global font tokens', async ({ page }) => {
  await page.route(/^https:\/\/fonts\.(googleapis|gstatic)\.com\//, (route) => route.abort());
  await page.goto('/', { waitUntil: 'commit' });
  await page.locator('app-root').waitFor({ state: 'attached' });
  await page.waitForFunction(() =>
    [...document.styleSheets].some((sheet) => sheet.href?.includes('styles'))
  );

  const families = await page.evaluate(() => {
    const root = document.documentElement;
    root.style.setProperty('--font-sans', '"SOLV Browser UI", sans-serif');
    root.style.setProperty('--font-mono', '"SOLV Browser Machine", monospace');

    const fixtures = [
      Object.assign(document.createElement('span'), { className: 'font-mono' }),
      Object.assign(document.createElement('time'), { dateTime: '2026-09-28', textContent: '2026-09-28' }),
      document.createElement('input'),
      document.createElement('select'),
      document.createElement('textarea'),
      document.createElement('button'),
    ];

    for (const fixture of fixtures) {
      document.body.append(fixture);
    }

    return {
      body: getComputedStyle(document.body).fontFamily,
      span: getComputedStyle(fixtures[0]).fontFamily,
      time: getComputedStyle(fixtures[1]).fontFamily,
      input: getComputedStyle(fixtures[2]).fontFamily,
      select: getComputedStyle(fixtures[3]).fontFamily,
      textarea: getComputedStyle(fixtures[4]).fontFamily,
      button: getComputedStyle(fixtures[5]).fontFamily,
    };
  });

  expect(families.span).toContain('SOLV Browser Machine');
  expect(families.time).toContain('SOLV Browser Machine');

  for (const control of ['input', 'select', 'textarea', 'button']) {
    expect(families[control]).toContain('SOLV Browser UI');
    expect(families[control]).toBe(families.body);
  }
});

test('shared roles follow the size matrix and tenant UI font', async ({ page }) => {
  await page.route(/^https:\/\/fonts\.(googleapis|gstatic)\.com\//, (route) => route.abort());
  await page.goto('/', { waitUntil: 'commit' });
  await page.locator('app-root').waitFor({ state: 'attached' });
  await page.waitForFunction(() =>
    [...document.styleSheets].some((sheet) => sheet.href?.includes('styles'))
  );

  const roles = await page.evaluate(() => {
    const root = document.documentElement;
    root.style.setProperty('--font-sans', '"SOLV Tenant UI", sans-serif');
    root.style.setProperty('--font-mono', '"SOLV Tenant Machine", monospace');

    const mk = (tag, className, text) => {
      const el = document.createElement(tag);
      if (className) el.className = className;
      el.textContent = text ?? className;
      document.body.append(el);
      return el;
    };

    const pageTitle = mk('h1', 'page-title', 'Título página');
    const sectionTitle = mk('h2', 'section-title', 'Título sección');
    const cardTitle = mk('h3', 'card-title', 'Título tarjeta');
    const label = mk('label', 'form-label', 'Etiqueta');
    const input = mk('input', 'form-input', '');
    const kpi = mk('div', 'kpi-value font-mono', '62%');
    const time = document.createElement('time');
    time.dateTime = '2026-09-28';
    time.textContent = '28/09/2026';
    document.body.append(time);
    const tableHead = mk('th', '', 'Cabecera');

    const css = (el) => getComputedStyle(el);
    return {
      pageTitle: { family: css(pageTitle).fontFamily, size: css(pageTitle).fontSize },
      sectionTitle: { family: css(sectionTitle).fontFamily, size: css(sectionTitle).fontSize },
      cardTitle: { family: css(cardTitle).fontFamily, size: css(cardTitle).fontSize },
      label: { family: css(label).fontFamily, size: css(label).fontSize },
      input: { family: css(input).fontFamily, size: css(input).fontSize },
      kpi: { family: css(kpi).fontFamily, size: css(kpi).fontSize },
      time: { family: css(time).fontFamily },
      tenantBody: css(document.body).fontFamily,
    };
  });

  // Familias: UI vs máquina con fuente tenant configurada.
  expect(roles.pageTitle.family).toContain('SOLV Tenant UI');
  expect(roles.sectionTitle.family).toContain('SOLV Tenant UI');
  expect(roles.label.family).toContain('SOLV Tenant UI');
  expect(roles.input.family).toContain('SOLV Tenant UI');
  expect(roles.kpi.family).toContain('SOLV Tenant Machine');
  expect(roles.time.family).toContain('SOLV Tenant Machine');
  expect(roles.tenantBody).toContain('SOLV Tenant UI');

  // Tamaños de la matriz: xl 20px, lg 18px, base 16px, xs 12px, sm 14px.
  expect(roles.pageTitle.size).toBe('20px');
  expect(roles.sectionTitle.size).toBe('18px');
  expect(roles.cardTitle.size).toBe('16px');
  expect(roles.label.size).toBe('12px');
  expect(roles.input.size).toBe('14px');
  expect(roles.kpi.size).toBe('20px');
});