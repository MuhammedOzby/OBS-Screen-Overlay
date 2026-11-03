'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '../template/assets/access.js'), 'utf8');

for (const [status, expected] of [
  [429, 'Çok fazla hatalı deneme. Birkaç dakika sonra tekrar deneyin.'],
  [403, 'Eşleştirme başarısız. Kodu kontrol edip tekrar deneyin.'],
]) {
  test(`pairing status ${status} displays the appropriate feedback`, async () => {
    const elements = {};
    for (const id of ['access-feedback', 'pair-form', 'pair-code', 'device-name', 'local-login']) {
      elements[id] = { value: '', textContent: '', addEventListener(type, fn) { this[type] = fn; } };
    }
    elements['pair-form'].reportValidity = () => true;
    elements['pair-code'].value = '123456';
    const context = {
      URLSearchParams,
      location: { hash: '', pathname: '/access', search: '', assign() { throw new Error('unexpected redirect'); } },
      history: {},
      document: { body: { dataset: { accessPage: 'login' } }, getElementById: (id) => elements[id] },
      overlayAccess: { session: () => Promise.reject(new Error('no session')) },
      fetch: async () => ({ status, ok: false }),
    };
    vm.runInNewContext(source, context);
    elements['pair-form'].submit({ preventDefault() {} });
    await new Promise(setImmediate);
    assert.equal(elements['access-feedback'].textContent, expected);
  });
}
