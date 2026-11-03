'use strict';

const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const runtimeSource = fs.readFileSync(path.join(__dirname, '..', 'routes', 'overlay-runtime.js'), 'utf8');

function createHarness(options) {
    options = options || {};
    const stateEl = { textContent: options.stateText || JSON.stringify({
        revision: 1,
        template_version: 1,
        events_url: '/view/token-a/events',
    }) };
    const spanList = (options.spans || []).map(span => ({
        dataset: { overlayField: span.name },
        textContent: span.value === undefined ? '' : span.value,
    }));
    const sources = [];
    const windowListeners = new Map();
    let reloadCount = 0;

    class FakeEventSource {
        constructor(url) {
            this.url = url;
            this.closed = false;
            this.listeners = new Map();
            sources.push(this);
        }
        addEventListener(type, handler) {
            if (!this.listeners.has(type)) {
                this.listeners.set(type, []);
            }
            this.listeners.get(type).push(handler);
        }
        close() {
            this.closed = true;
        }
    }

    const windowObject = {
        location: {
            reload: function() {
                reloadCount += 1;
            },
        },
        addEventListener: function(type, listener) {
            if (!windowListeners.has(type)) {
                windowListeners.set(type, []);
            }
            windowListeners.get(type).push(listener);
        },
    };

    const documentObject = {
        querySelector: function(selector) {
            if (selector === '#overlay-system-state') {
                return stateEl;
            }
            return null;
        },
        querySelectorAll: function() {
            return spanList;
        },
    };

    const context = vm.createContext({
        document: documentObject,
        window: windowObject,
        EventSource: FakeEventSource,
    });
    vm.runInContext(runtimeSource, context, { filename: 'overlay-runtime.js', timeout: 1000 });

    function emitEvent(source, type, data) {
        const handlers = source.listeners.get(type);
        if (!handlers) {
            return false;
        }
        for (const handler of handlers) {
            handler({ data: data });
        }
        return true;
    }

    function fireWindowEvent(type, event) {
        const listeners = windowListeners.get(type);
        if (!listeners) {
            return false;
        }
        for (const listener of listeners) {
            listener(event);
        }
        return true;
    }

    return {
        stateEl: stateEl,
        spanList: spanList,
        sources: sources,
        emitEvent: emitEvent,
        fireWindowEvent: fireWindowEvent,
        get reloadCount() {
            return reloadCount;
        },
    };
}

function statePayload(revision, templateVersion, fields) {
    const payload = {
        revision: revision,
        template_version: templateVersion,
        fields: fields,
    };
    return JSON.stringify(payload);
}

test('runtime opens a single event source from the initial metadata', () => {
    const harness = createHarness();
    assert.strictEqual(harness.sources.length, 1);
    assert.strictEqual(harness.sources[0].url, '/view/token-a/events');
    assert.strictEqual(harness.sources[0].closed, false);
});

test('runtime stays idle when initial metadata is malformed', () => {
    const harness = createHarness({ stateText: 'not-json{{' });
    assert.strictEqual(harness.sources.length, 0);
});

test('runtime stays idle when initial metadata has unsafe versions', () => {
    const harness = createHarness({
        stateText: JSON.stringify({ revision: 0, template_version: 0, events_url: '/view/token-a/events' }),
    });
    assert.strictEqual(harness.sources.length, 0);
});

test('runtime ignores malformed JSON state payloads', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.sources.length, 1);
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', 'not-json{{'), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
});

test('runtime ignores state payloads with unsafe or invalid versions', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(0, 1, { open: 'two' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(1.5, 1, { open: 'two' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(2, 0, { open: 'two' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
});

test('runtime ignores state payloads whose fields are not a non-null string map', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', JSON.stringify({
        revision: 2,
        template_version: 1,
        fields: null,
    })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', JSON.stringify({
        revision: 2,
        template_version: 1,
        fields: [],
    })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', JSON.stringify({
        revision: 2,
        template_version: 1,
        fields: { open: 5 },
    })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'one');
});

test('runtime writes repeated fields to every span with textContent', () => {
    const harness = createHarness({
        spans: [
            { name: 'konuk_adi', value: 'ÖZBAY' },
            { name: 'konuk_adi', value: 'ÖZBAY' },
        ],
    });
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(2, 1, { konuk_adi: '<b>ÖZBAY</b>' })), true);
    assert.strictEqual(harness.spanList[0].textContent, '<b>ÖZBAY</b>');
    assert.strictEqual(harness.spanList[1].textContent, '<b>ÖZBAY</b>');
});

test('runtime keeps spans without a payload entry untouched', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(2, 1, { open: 'two' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'two');
    // SSE-disabled field: absent from payload must not reset or clear DOM.
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(3, 1, {})), true);
    assert.strictEqual(harness.spanList[0].textContent, 'two');
});

test('runtime ignores stale revisions', () => {
    const harness = createHarness({
        stateText: JSON.stringify({ revision: 5, template_version: 1, events_url: '/view/token-a/events' }),
        spans: [{ name: 'open', value: 'five' }],
    });
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(5, 1, { open: 'stale' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'five');
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(4, 1, { open: 'stale' })), true);
    assert.strictEqual(harness.spanList[0].textContent, 'five');
});

test('runtime reloads once for a template version change on the state event', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.reloadCount, 0);
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(2, 2, { open: 'two' })), true);
    assert.strictEqual(harness.reloadCount, 1);
    assert.strictEqual(harness.sources[0].closed, true);
    // The reload guard must suppress further reloads.
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'state', statePayload(3, 3, { open: 'three' })), true);
    assert.strictEqual(harness.reloadCount, 1);
});

test('runtime reloads once for a template_changed event', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    assert.strictEqual(harness.reloadCount, 0);
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'template_changed', JSON.stringify({ revision: 2, template_version: 2 })), true);
    assert.strictEqual(harness.reloadCount, 1);
    // The reload guard must suppress further reloads.
    assert.strictEqual(harness.emitEvent(harness.sources[0], 'template_changed', JSON.stringify({ revision: 3, template_version: 3 })), true);
    assert.strictEqual(harness.reloadCount, 1);
});

test('invalid template_changed events do not throw, reload or poison the next event', () => {
    const harness = createHarness();
    const source = harness.sources[0];
    const invalid = [
        'not-json', 'null', '{}', '[]',
        JSON.stringify({ revision: 2, template_version: 0 }),
        JSON.stringify({ revision: 2, template_version: 1.5 }),
        JSON.stringify({ revision: 2, template_version: Number.MAX_SAFE_INTEGER + 1 }),
        JSON.stringify({ revision: Number.MAX_SAFE_INTEGER + 1, template_version: 2 }),
        JSON.stringify({ revision: 1, template_version: 2 }),
    ];
    for (const data of invalid) {
        assert.doesNotThrow(() => harness.emitEvent(source, 'template_changed', data));
        assert.strictEqual(harness.reloadCount, 0);
        assert.strictEqual(source.closed, false);
    }
    harness.emitEvent(source, 'template_changed', JSON.stringify({ revision: 2, template_version: 2 }));
    assert.strictEqual(harness.reloadCount, 1);
    assert.strictEqual(source.closed, true);
});

test('an error event does not reload or replace the source; later state still works', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    const source = harness.sources[0];
    harness.emitEvent(source, 'error', '');
    assert.strictEqual(harness.sources.length, 1);
    assert.strictEqual(harness.reloadCount, 0);
    assert.strictEqual(source.closed, false);
    harness.emitEvent(source, 'state', statePayload(2, 1, { open: 'after error' }));
    assert.strictEqual(harness.spanList[0].textContent, 'after error');
});

test('runtime closes the connection on pagehide and reopens once for a persisted pageshow', () => {
    const harness = createHarness({ spans: [{ name: 'open', value: 'one' }] });
    const firstSource = harness.sources[0];
    assert.strictEqual(harness.fireWindowEvent('pagehide', {}), true);
    assert.strictEqual(firstSource.closed, true);
    assert.strictEqual(harness.fireWindowEvent('pageshow', { persisted: true }), true);
    assert.strictEqual(harness.sources.length, 2);
    assert.strictEqual(harness.sources[0].closed, true);
    assert.strictEqual(harness.sources[1].closed, false);
    // A second persisted pageshow must not open another connection.
    assert.strictEqual(harness.fireWindowEvent('pageshow', { persisted: true }), true);
    assert.strictEqual(harness.sources.length, 2);
    // A non-persisted pageshow must not reopen.
    assert.strictEqual(harness.fireWindowEvent('pageshow', { persisted: false }), true);
    assert.strictEqual(harness.sources.length, 2);
});
