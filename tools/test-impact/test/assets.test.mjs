import assert from 'node:assert/strict';
import test from 'node:test';
import { validateAssetContent } from '../lib/assets.mjs';

const svg = (body) => Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg">${body}</svg>`);

test('un SVG bien formado pasa la validación', () => {
  assert.equal(validateAssetContent('icon.svg', svg('<path d="M0 0h1v1z"/>')).ok, true);
});

test('un SVG truncado o con plantilla sin resolver falla', () => {
  assert.equal(validateAssetContent('icon.svg', Buffer.from('<svg><path')).ok, false);
  assert.equal(validateAssetContent('icon.svg', svg('{{ color }}')).ok, false);
  assert.equal(validateAssetContent('icon.svg', Buffer.from('no es un svg')).ok, false);
});

test('un JSON inválido falla y uno válido pasa', () => {
  assert.equal(validateAssetContent('data.json', Buffer.from('{"a": 1}')).ok, true);
  assert.equal(validateAssetContent('data.json', Buffer.from('{a: 1}')).ok, false);
});

test('las imágenes se validan por firma de formato', () => {
  const png = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0]);
  const jpeg = Buffer.from([0xff, 0xd8, 0xff, 0xe0, 0, 0]);
  const webp = Buffer.concat([
    Buffer.from('RIFF'),
    Buffer.from([0, 0, 0, 0]),
    Buffer.from('WEBP')
  ]);
  assert.equal(validateAssetContent('image.png', png).ok, true);
  assert.equal(validateAssetContent('image.jpg', jpeg).ok, true);
  assert.equal(validateAssetContent('image.webp', webp).ok, true);
  assert.equal(validateAssetContent('image.png', Buffer.from('falso')).ok, false);
});

test('SCSS y HTML tienen comprobaciones sintácticas proporcionales', () => {
  assert.equal(validateAssetContent('style.scss', Buffer.from('.a { color: red; }')).ok, true);
  assert.equal(validateAssetContent('style.scss', Buffer.from('.a { color: red;')).ok, false);
  assert.equal(validateAssetContent('page.html', Buffer.from('<!doctype html><html></html>')).ok, true);
  assert.equal(validateAssetContent('page.html', Buffer.from('<div></div>')).ok, false);
});

test('un formato sin validador semántico se acepta y declara el límite', () => {
  const result = validateAssetContent('font.ttf', Buffer.from([0x00, 0x01, 0x00, 0x00, 1]));
  assert.equal(result.ok, true);
  const other = validateAssetContent('file.bin', Buffer.from('contenido'));
  assert.equal(other.ok, true);
  assert.match(other.limit, /sin validador semántico/);
});
