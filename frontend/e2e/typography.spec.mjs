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