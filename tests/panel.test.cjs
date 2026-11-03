'use strict';

// Panel JavaScript davranış testleri — minimal DOM harness + Node built-in node:test.
// template/assets/panel.js kaynak kodu bu sınırlı fake DOM içinde çalıştırılır ve
// gerçek event handler'ları (submit/click) üzerinden test edilir.

const fs = require('fs');
const path = require('path');
const vm = require('vm');
const test = require('node:test');
const assert = require('node:assert/strict');

const panelSource = fs.readFileSync(path.join(__dirname, '..', 'template', 'assets', 'panel.js'), 'utf8');
const accessSource = fs.readFileSync(path.join(__dirname, '..', 'template', 'assets', 'access-client.js'), 'utf8');

class MockFormData {
  constructor(form) { this.form = form; this.data = Object.create(null); }
  append(key, value) { this.data[key] = (value && typeof value === 'object' && typeof value.name === 'string') ? value : String(value); }
}

class FakeElement {
  constructor(tag, opts) {
    opts = opts || {};
    this.tagName = String(tag).toUpperCase();
    this.nodeType = 1;
    this.children = [];
    this.parentNode = null;
    this.listeners = Object.create(null);
    this.dataset = new Proxy(Object.assign({}, opts.dataset || {}), {
      set(target, key, value) { target[key] = String(value); return true; }
    });
    this.attributes = Object.assign({}, opts.attributes || {});
    this.style = {};
    this.disabled = !!opts.disabled;
    this.hidden = !!opts.hidden;
    this.checked = !!opts.checked;
    this.required = !!opts.required;
    this.pattern = opts.pattern || '';
    this.name = opts.name || '';
    this.value = opts.value === undefined ? '' : opts.value;
    this.textContent = opts.textContent || '';
    this.innerHTML = '';
    this.className = opts.className || '';
    this.id = opts.id || '';
    this.type = opts.type || (this.tagName === 'INPUT' ? 'text' : this.tagName === 'BUTTON' ? 'button' : this.tagName === 'TEXTAREA' ? 'textarea' : this.tagName === 'SELECT' ? 'select' : 'text');
    this.files = opts.files || [];
    this.rows = opts.rows || 2;
    this.accept = opts.accept || '';
  }
  get classList() { const self = this; return { contains: (c) => String(self.className || '').split(/\s+/).indexOf(c) !== -1 }; }
  appendChild(child) { child.parentNode = this; this.children.push(child); return child; }
  append(...kids) { for (const k of kids) { if (k !== null && k !== undefined && typeof k === 'object') { this.appendChild(k); } } }
  remove() { const p = this.parentNode; if (p) { const i = p.children.indexOf(this); if (i !== -1) { p.children.splice(i, 1); } this.parentNode = null; } }
  addEventListener(type, fn) { (this.listeners[type] = this.listeners[type] || []).push(fn); }
  dispatchEvent(event) {
    const handlers = (this.listeners[event.type] || []).slice();
    const results = handlers.map((fn) => fn(event));
    return Promise.all(results.map((r) => (r && typeof r.then === 'function') ? r : Promise.resolve()));
  }
  preventDefault() { this.defaultPrevented = true; }
  reset() {
    const visit = (el) => {
      if (el.tagName === 'INPUT' && el.type === 'file') { el.files = []; }
      else if (el.tagName === 'INPUT' && el.type !== 'button' && el.type !== 'checkbox' && el.type !== 'radio') { el.value = ''; }
      else if (el.tagName === 'TEXTAREA') { el.value = ''; }
      for (const c of el.children) { visit(c); }
    };
    visit(this);
  }
  focus() { this.focused = true; }
  select() { this.selected = true; }
  reportValidity() { return true; }
  querySelector(selector) { const all = collect(this, selector); return all.length > 0 ? all[0] : null; }
  querySelectorAll(selector) { return collect(this, selector); }
  getAttribute(name) { if (Object.prototype.hasOwnProperty.call(this.attributes, name)) { return this.attributes[name]; } if (name === 'id') { return this.id; } return undefined; }
  setAttribute(name, value) { this.attributes[name] = String(value); if (name === 'id') { this.id = String(value); } }
  closest(selector) { let node = this; while (node) { if (matches(node, selector)) { return node; } node = node.parentNode; } return null; }
}

function parseCompound(str) {
  const result = { tag: null, id: null, classes: [], attrs: [] };
  const re = /([a-zA-Z][a-zA-Z0-9-]*)|(#[A-Za-z0-9_-]+)|(\.[A-Za-z0-9_-]+)|\[([^\]]+)\]/g;
  let m;
  re.lastIndex = 0;
  while ((m = re.exec(str)) !== null) {
    if (m[1] !== undefined) { result.tag = m[1].toUpperCase(); }
    else if (m[2] !== undefined) { result.id = m[2].slice(1); }
    else if (m[3] !== undefined) { result.classes.push(m[3].slice(1)); }
    else if (m[4] !== undefined) {
      const am = /^([a-zA-Z0-9_-]+)(?:\s*=\s*["']?([^"']*)["']?)?/.exec(m[4]);
      if (am) { result.attrs.push({ name: am[1], value: am[2] }); }
    }
  }
  return result;
}

function matchesCompound(el, c) {
  if (c.tag !== null && el.tagName !== c.tag) { return false; }
  if (c.id !== null && el.id !== c.id) { return false; }
  const classes = String(el.className || '').split(/\s+/);
  for (const cls of c.classes) { if (classes.indexOf(cls) === -1) { return false; } }
  for (const attr of c.attrs) {
    const actual = el[attr.name];
    if (attr.value === undefined) { if (actual === undefined || actual === null) { return false; } }
    else if (String(actual) !== String(attr.value)) { return false; }
  }
  return true;
}

function matches(el, selector) {
  if (String(selector).includes(',')) {
    return String(selector).split(',').some((part) => matches(el, part));
  }
  const parts = String(selector).trim().split(/\s+/);
  if (parts.length === 0 || parts[0] === '') { return false; }
  if (!matchesCompound(el, parseCompound(parts[parts.length - 1]))) { return false; }
  let node = el.parentNode;
  for (let i = parts.length - 2; i >= 0; i -= 1) {
    const part = parseCompound(parts[i]);
    let found = null;
    let cur = node;
    while (cur) { if (matchesCompound(cur, part)) { found = cur; break; } cur = cur.parentNode; }
    if (found === null) { return false; }
    node = found.parentNode;
  }
  return true;
}

function collect(root, selector) {
  const out = [];
  const walk = (el) => {
    for (const child of el.children) {
      if (child && typeof child === 'object' && matches(child, selector)) { out.push(child); }
      walk(child);
    }
  };
  walk(root);
  return out;
}

function buildPanelDocument(page) {
  const registry = Object.create(null);
  const root = new FakeElement('html');
  const body = new FakeElement('body');
  root.appendChild(body);
  const reg = (el) => { if (el.id !== '') { registry[el.id] = el; } return el; };

  const demoFeedback = reg(new FakeElement('div', { id: 'demo-feedback' }));
  const reloadRecord = reg(new FakeElement('a', { id: 'reload-record', attributes: { href: '/duzelt/rec1' } }));
  reloadRecord.hidden = true;
  const copyFeedback = reg(new FakeElement('div', { id: 'copy-feedback' }));
  const obsUrl = reg(new FakeElement('input', { id: 'obs-url', type: 'text', value: 'http://127.0.0.1:8081/view/tok' }));
  const copyButton = reg(new FakeElement('button', { id: 'copy-obs-url', type: 'button', textContent: 'OBS bağlantısını kopyala' }));

  const newForm = reg(new FakeElement('form', { id: 'new-form' }));
  newForm.appendChild(new FakeElement('button', { id: 'create-button', type: 'submit', disabled: true, textContent: 'Kaydı oluştur' }));
  newForm.appendChild(new FakeElement('input', { id: 'record-name', type: 'text', name: 'name' }));
  newForm.appendChild(new FakeElement('select', { id: 'record-type', name: 'type', type: 'select', value: 'screen_overlay' }));
  newForm.appendChild(new FakeElement('input', { id: 'html-file', name: 'html', type: 'file', accept: '.html,text/html' }));

  const editForm = reg(new FakeElement('form', { id: 'edit-form', dataset: { id: 'rec1', revision: '7' } }));
  editForm.appendChild(new FakeElement('button', { id: 'apply-button', type: 'submit', disabled: true, textContent: 'Yayına uygula' }));
  editForm.appendChild(new FakeElement('input', { id: 'record-name', type: 'text', name: 'name', value: 'Konuk' }));
  editForm.appendChild(new FakeElement('select', { id: 'record-type', type: 'select', value: 'screen_overlay' }));
  const fieldList = reg(new FakeElement('div', { id: 'field-list', className: 'field-list' }));
  const addField = reg(new FakeElement('button', { id: 'add-field', type: 'button', textContent: 'Değişken ekle' }));
  const fieldRow = new FakeElement('div', { className: 'field-row mb-3' });
  const fieldNameInput = new FakeElement('input', { type: 'text', name: 'field_name', value: 'konuk_adi' });
  const fieldValueArea = new FakeElement('textarea', { name: 'field_value', rows: 2, value: 'ÖZBAY' });
  const fieldSse = new FakeElement('input', { type: 'checkbox', name: 'field_sse', checked: true });
  const removeFieldBtn = new FakeElement('button', { type: 'button', className: 'remove-field', textContent: 'Kaldır' });
  fieldRow.appendChild(fieldNameInput);
  fieldRow.appendChild(fieldValueArea);
  fieldRow.appendChild(fieldSse);
  fieldRow.appendChild(removeFieldBtn);
  fieldList.appendChild(fieldRow);
  editForm.appendChild(fieldList);
  editForm.appendChild(addField);

  const templateForm = reg(new FakeElement('form', { id: 'template-form' }));
  const fileInput = reg(new FakeElement('input', { id: 'html-file', type: 'file', disabled: true, accept: '.html,text/html' }));
  const publishButton = reg(new FakeElement('button', { id: 'publish-button', type: 'submit', disabled: true, textContent: 'Tasarımı yayınla' }));
  templateForm.appendChild(fileInput);
  templateForm.appendChild(publishButton);

  const deleteFeedback = reg(new FakeElement('div', { id: 'delete-feedback' }));
  const emptyNote = reg(new FakeElement('p', { id: 'empty-records', className: 'alert' }));
  emptyNote.hidden = true;
  const recordsTable = new FakeElement('table', { className: 'records-table' });
  const recordsBody = new FakeElement('tbody', { className: 'records-body' });
  const delRow = new FakeElement('tr', { className: 'record-row' });
  const delButton = new FakeElement('button', { type: 'button', className: 'delete-record', disabled: true, textContent: 'Sil', attributes: { 'data-id': 'rec1', 'data-name': 'Test' } });
  delRow.appendChild(delButton);
  recordsBody.appendChild(delRow);
  recordsTable.appendChild(recordsBody);

  if (page === 'new') {
    body.append(newForm, demoFeedback);
  } else if (page === 'list') {
    body.append(deleteFeedback, emptyNote, recordsTable);
  } else {
    body.append(editForm, templateForm, demoFeedback, reloadRecord, copyFeedback, obsUrl, copyButton);
  }
  // Only attached elements exist, with the same unique IDs as the real pages.
  for (const key of Object.keys(registry)) { delete registry[key]; }
  const registerTree = (el) => {
    reg(el);
    for (const child of el.children) { registerTree(child); }
  };
  registerTree(root);

  const documentApi = {
    title: '',
    getElementById: (id) => registry[id] || null,
    querySelector: (selector) => { const all = collect(root, selector); return all.length > 0 ? all[0] : null; },
    querySelectorAll: (selector) => collect(root, selector),
    createElement: (tag) => new FakeElement(tag),
  };

  return { registry, root, document: documentApi };
}

function boot(opts) {
  opts = opts || {};
  const doc = buildPanelDocument(opts.page || 'edit');
  const calls = [];
  const alerts = [];
  const clipboardWrites = [];
  const redirects = [];
  let revision = 7;
  const respond = opts.respond || (async (url, request) => {
    if (request.method === 'DELETE') { return { ok: true, status: 204 }; }
    if (url === '/api/pages') { return { ok: true, status: 201, json: async () => ({ redirect: '/duzelt/new-record' }) }; }
    return { ok: true, status: 200, json: async () => ({ revision: ++revision }) };
  });
  const fetchImpl = async (url, requestOpts) => {
    if (url === '/access/session') return opts.sessionFail ? { ok: false } : { ok: true, json: async () => ({ csrf_token: 'test-csrf', local: true }) };
    assert.equal(requestOpts.headers.get('X-CSRF-Token'), 'test-csrf');
    calls.push({ url, opts: requestOpts }); return respond(url, requestOpts);
  };
  const context = {
    document: doc.document,
    window: { confirm: () => (opts.confirm === undefined ? true : opts.confirm) },
    navigator: { clipboard: { writeText: async (value) => {
      if (opts.clipboardError) { throw new Error('clipboard denied'); }
      clipboardWrites.push(value);
    } } },
    fetch: fetchImpl,
    alert: (msg) => { alerts.push(msg); },
    FormData: MockFormData,
    Headers,
    location: { assign: (url) => { redirects.push(url); } },
    console,
  };
  context.confirm = context.window.confirm;
  context.window = context;
  vm.createContext(context);
  vm.runInContext(accessSource, context, { filename: 'access-client.js', timeout: 1000 });
  vm.runInContext(panelSource, context, { filename: 'panel.js', timeout: 1000 });
  return { doc, document: doc.document, calls, alerts, redirects, clipboardWrites };
}

const flush = () => new Promise(setImmediate);

const submitEvent = (target) => ({ type: 'submit', preventDefault() { this.defaultPrevented = true; }, target });

test('failed session lookup never sends a panel mutation', async () => {
  const { document, calls } = boot({ sessionFail: true });
  const form = document.getElementById('edit-form');
  await form.dispatchEvent(submitEvent(form));
  assert.equal(calls.length, 0);
});

test('script enables initially disabled publish and edit controls', async () => {
  const { document } = boot();
  const publishButton = document.getElementById('publish-button');
  const fileInput = document.getElementById('html-file');
  const editForm = document.getElementById('edit-form');
  const applyButton = editForm ? editForm.querySelector('button[type="submit"]') : null;
  assert.ok(publishButton, 'publish butonu bulunamadi');
  assert.ok(fileInput, 'html dosya girdisi bulunamadi');
  assert.equal(publishButton.disabled, false, 'publish butonu baslangicta disabled olmamalidir');
  assert.equal(fileInput.disabled, false, 'html dosya girdisi baslangicinda disabled olmamalidir');
  assert.equal(applyButton.disabled, false, 'apply butonu baslangicta disabled olmamalidir');
});

test('pending template publish blocks PATCH and duplicate publish', async () => {
  let resolveFetch;
  const pendingResponse = new Promise((resolve) => { resolveFetch = resolve; });
  const { calls, document } = boot({ respond: () => pendingResponse });
  const fileInput = document.getElementById('html-file');
  fileInput.files = [{ name: 'tasarim.html', size: 1024 }];
  const editForm = document.getElementById('edit-form');
  editForm.dataset.revision = '7';
  const templateForm = document.getElementById('template-form');
  const publishPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await flush();
  assert.equal(calls.length, 1);
  assert.ok(calls[0].url.indexOf('/api/pages/rec1/template') !== -1, 'onceki bekleyen istek yayın musteri');
  assert.equal(calls[0].opts.method, 'POST');
  assert.equal(document.getElementById('apply-button').disabled, true);
  assert.equal(document.getElementById('publish-button').disabled, true);
  const duplicate = templateForm.dispatchEvent(submitEvent(templateForm));

  const applyPromise = editForm.dispatchEvent(submitEvent(editForm));
  await flush();
  assert.equal(calls.length, 1, 'yayın sürerken apply/PATCH yollanmamalı');

  resolveFetch({ ok: true, status: 200, json: async () => ({ revision: 43 }) });
  await publishPromise;
  await applyPromise;
  await duplicate;
  assert.equal(document.getElementById('apply-button').disabled, false);
  assert.equal(document.getElementById('publish-button').disabled, false);
});

test('pending PATCH blocks template publish and duplicate PATCH', async () => {
  let resolveFetch;
  const pendingResponse = new Promise((resolve) => { resolveFetch = resolve; });
  const { calls, document } = boot({ respond: () => pendingResponse });
  const editForm = document.getElementById('edit-form');
  const templateForm = document.getElementById('template-form');
  document.getElementById('html-file').files = [{ name: 'valid.html', size: 1024 }];
  const applyPromise = editForm.dispatchEvent(submitEvent(editForm));
  await flush();
  assert.equal(calls.length, 1);
  assert.ok(calls[0].url === '/api/pages/rec1');
  assert.equal(document.getElementById('apply-button').disabled, true);
  assert.equal(document.getElementById('publish-button').disabled, true);
  const duplicate = editForm.dispatchEvent(submitEvent(editForm));

  const publishPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await flush();
  assert.equal(calls.length, 1, 'apply sürerken tasarım yayınlaması bloklanmalı');

  resolveFetch({ ok: true, status: 200, json: async () => ({ revision: 44 }) });
  await applyPromise;
  await publishPromise;
  await duplicate;
});

test('successful publish updates revision, preserves edit draft and permits retry', async () => {
  const { calls, document } = boot();
  const fileInput = document.getElementById('html-file');
  const selectedFile = { name: 'tasarim.html', size: 1024 };
  fileInput.files = [selectedFile];
  const editForm = document.getElementById('edit-form');
  editForm.dataset.revision = '7';
  const templateForm = document.getElementById('template-form');
  const feedback = document.getElementById('demo-feedback');
  const publishButton = document.getElementById('publish-button');
  const publishPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await publishPromise;
  assert.equal(calls.length, 1);
  assert.equal(calls[0].opts.method, 'POST');
  assert.ok(calls[0].url.indexOf('/api/pages/rec1/template') !== -1);
  const body = calls[0].opts.body;
  assert.ok(body, 'FormData gövdesi beklenmeli');
  assert.equal(body.data['expected_revision'], '7');
  assert.equal(body.data['html'], selectedFile);
  assert.notEqual(body.form, editForm, 'template publish must not serialize the edit draft');
  assert.equal(editForm.dataset.revision, '8');
  assert.equal(feedback.textContent, 'Tasarım yayınlandı. Açık OBS sayfaları yenilenir.');
  assert.equal(publishButton.disabled, false);

  fileInput.files = [{ name: 'tasarim.html', size: 1024 }];
  const retryPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await retryPromise;
  assert.equal(calls.length, 2, 'basarisiz retry yok; basarili retry serbest');
  assert.equal(calls[1].opts.body.data.expected_revision, '8');
  assert.deepEqual(Object.keys(body.data).sort(), ['expected_revision', 'html']);
});

test('cancelling template publication sends no request and keeps the selected file', async () => {
  const { calls, document } = boot({ confirm: false });
  const file = { name: 'selected.html', size: 1024 };
  document.getElementById('html-file').files = [file];
  const form = document.getElementById('template-form');
  await form.dispatchEvent(submitEvent(form));
  assert.equal(calls.length, 0);
  assert.equal(document.getElementById('html-file').files[0], file);
  assert.equal(document.getElementById('edit-form').dataset.revision, '7');
  assert.equal(document.getElementById('apply-button').disabled, false);
  assert.equal(document.getElementById('publish-button').disabled, false);
});

test('conflict preserves the edit draft, selected file and old revision', async () => {
  const conflictResponse = { ok: false, status: 409, json: async () => ({ error: 'Revizyon çatışması: Kayıt bu sırada değişti' }) };
  const { calls, document } = boot({ respond: async () => conflictResponse });
  const fileInput = document.getElementById('html-file');
  fileInput.files = [{ name: 'tasarim.html', size: 1024 }];
  const editForm = document.getElementById('edit-form');
  editForm.dataset.revision = '7';
  const editName = editForm.querySelector('#record-name');
  editName.value = 'Yeni İsim';
  const fieldRows = editForm.querySelectorAll('.field-row');
  assert.ok(fieldRows.length > 0);
  fieldRows[0].querySelector('input[type="text"]').value = 'konuk_adi';
  const templateForm = document.getElementById('template-form');
  const feedback = document.getElementById('demo-feedback');
  const publishButton = document.getElementById('publish-button');
  const submitPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await submitPromise;
  assert.equal(calls.length, 1);
  assert.equal(document.getElementById('reload-record').hidden, false);
  assert.equal(editForm.dataset.revision, '7');
  assert.equal(fileInput.files.length, 1);
  assert.equal(editName.value, 'Yeni İsim');
  assert.equal(publishButton.disabled, false);
  assert.ok(String(feedback.textContent).length > 0, '409 kullaniciya geri bildirim vermeli');
});

test('network failure preserves the selected file and restores controls', async () => {
  const { calls, document } = boot({ respond: async () => { throw new Error('network down'); } });
  const fileInput = document.getElementById('html-file');
  fileInput.files = [{ name: 'tasarim.html', size: 1024 }];
  const editForm = document.getElementById('edit-form');
  editForm.dataset.revision = '7';
  const templateForm = document.getElementById('template-form');
  const feedback = document.getElementById('demo-feedback');
  const publishButton = document.getElementById('publish-button');
  const submitPromise = templateForm.dispatchEvent(submitEvent(templateForm));
  await submitPromise;
  assert.equal(calls.length, 1);
  assert.equal(publishButton.disabled, false, 'hata sonrasi buton tekrar kullanilabilir olmalı');
  assert.ok(String(feedback.textContent).length > 0, 'ağ hatasında geri bildirim verilmeli');
  assert.equal(editForm.dataset.revision, '7');
  assert.equal(fileInput.files.length, 1);
});

test('new page requires an HTML file and redirects after successful creation', async () => {
  const { calls, document, redirects } = boot({ page: 'new' });
  const newForm = document.getElementById('new-form');
  const createButton = newForm.querySelector('button[type="submit"]');
  const feedback = document.getElementById('demo-feedback');
  newForm.querySelector('#record-name').value = 'Yeni Kayıt';
  const submitPromise = newForm.dispatchEvent(submitEvent(newForm));
  await submitPromise;
  assert.equal(calls.length, 0, 'dosya olmadan istek gonderilmemeli');
  assert.equal(createButton.disabled, false);
  assert.ok(String(feedback.textContent).length > 0, 'dogrulama mesaji geri bildirimde olmalı');

  const fileInput = newForm.querySelector('#html-file');
  fileInput.files = [{ name: 'not-html.zip', size: 1024 }];
  await newForm.dispatchEvent(submitEvent(newForm));
  assert.equal(calls.length, 0, 'invalid extension must not reach the server');
  fileInput.files = [{ name: 'tasarim.html', size: 1024 }];
  newForm.querySelector('#record-name').value = 'Yeni Kayıt';
  const createPromise = newForm.dispatchEvent(submitEvent(newForm));
  await createPromise;
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, '/api/pages');
  assert.equal(calls[0].opts.method, 'POST');
  assert.equal(calls[0].opts.body.form, newForm);
  assert.deepEqual(redirects, ['/duzelt/new-record']);
  await newForm.dispatchEvent(submitEvent(newForm));
  assert.equal(calls.length, 1, 'successful creation stays locked while navigating');
});

test('delete cancellation sends no request; server error keeps the row usable', async () => {
  const { calls, document } = boot({ page: 'list', confirm: false });
  const deleteButtons = document.querySelectorAll('.delete-record');
  assert.equal(deleteButtons.length, 1);
  const delButton = deleteButtons[0];

  // Iptal: confirm false -> istek yok, buton kilinmez.
  const cancelPromise = delButton.dispatchEvent({ type: 'click', target: delButton, preventDefault() { this.defaultPrevented = true; } });
  await cancelPromise;
  assert.equal(calls.length, 0);
  assert.equal(delButton.disabled, false);

  // Hata: confirm true -> 500/JSON hatasi -> geri bildirim ve buton yenilenir.
  const errorResponse = { ok: false, status: 500, json: async () => ({ error: 'Kayıt silinemedi. Lütfen tekrar deneyin.' }) };
  const { calls: errorCalls, document: errorDoc } = boot({ page: 'list', respond: async () => errorResponse });
  const errButton = errorDoc.querySelectorAll('.delete-record')[0];
  const errPromise = errButton.dispatchEvent({ type: 'click', target: errButton, preventDefault() { this.defaultPrevented = true; } });
  await errPromise;
  assert.equal(errorCalls.length, 1);
  assert.equal(errButton.disabled, false);
  assert.equal(errorDoc.querySelectorAll('.records-table tbody tr').length, 1);
});

test('successful delete removes the row and shows the empty-list note', async () => {
  const { calls, document } = boot({ page: 'list' });
  const button = document.querySelector('.delete-record');
  await button.dispatchEvent({ type: 'click', target: button });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].opts.method, 'DELETE');
  assert.equal(document.querySelectorAll('.records-table tbody tr').length, 0);
  assert.equal(document.getElementById('empty-records').hidden, false);
});

test('copy uses the saved public URL and selects it when clipboard access fails', async () => {
  const success = boot();
  await success.document.getElementById('copy-obs-url').dispatchEvent({ type: 'click' });
  assert.deepEqual(success.clipboardWrites, [success.document.getElementById('obs-url').value]);
  const denied = boot({ clipboardError: true });
  await denied.document.getElementById('copy-obs-url').dispatchEvent({ type: 'click' });
  const input = denied.document.getElementById('obs-url');
  assert.equal(input.focused, true);
  assert.equal(input.selected, true);
  assert.notEqual(denied.document.getElementById('copy-feedback').textContent, '');
});

test('fields can be added with SSE enabled and removed without a request', async () => {
  const { calls, document } = boot();
  const list = document.getElementById('field-list');
  await document.getElementById('add-field').dispatchEvent({ type: 'click' });
  const rows = list.querySelectorAll('.field-row');
  assert.equal(rows.length, 2);
  assert.equal(rows[1].querySelector('input[type="checkbox"]').checked, true);
  const remove = rows[1].querySelector('.remove-field');
  await list.dispatchEvent({ type: 'click', target: remove });
  assert.equal(list.querySelectorAll('.field-row').length, 1);
  assert.equal(calls.length, 0);
});
