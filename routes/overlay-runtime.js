(function() {
    const stateEl = document.querySelector('#overlay-system-state');
    if (!stateEl) {
        return;
    }

    const fieldsMap = new Map();
    let currentState = {};
    let currentRevision = -1;
    let currentTemplateVersion = null;
    let hasReloaded = false;
    let eventSource = null;

    function isSafePositiveInt(value) {
        return Number.isSafeInteger(value) && value > 0;
    }

    function closeEventSource() {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
    }

    function reloadForTemplateChange() {
        if (hasReloaded) {
            return;
        }
        hasReloaded = true;
        closeEventSource();
        window.location.reload();
    }

    function initializeEventSource(url) {
        if (hasReloaded) {
            return;
        }
        closeEventSource();
        eventSource = new EventSource(url);

        eventSource.addEventListener('state', function(event) {
            handleStateEvent(event);
        });

        eventSource.addEventListener('template_changed', function(event) {
            handleTemplateChangedEvent(event);
        });
    }

    function handleStateEvent(event) {
        let payload;
        try {
            payload = JSON.parse(event.data);
        } catch (error) {
            return;
        }
        if (!payload || !isSafePositiveInt(payload.revision) || !isSafePositiveInt(payload.template_version)) {
            return;
        }
        const fields = payload.fields;
        if (typeof fields !== 'object' || fields === null || Array.isArray(fields)) {
            return;
        }
        for (const fieldName in fields) {
            if (typeof fields[fieldName] !== 'string') {
                return;
            }
        }
        if (payload.revision <= currentRevision) {
            return;
        }
        if (payload.template_version !== currentTemplateVersion) {
            reloadForTemplateChange();
            return;
        }
        const fieldNames = Object.keys(fields);
        for (let i = 0; i < fieldNames.length; i++) {
            const fieldName = fieldNames[i];
            const newValue = fields[fieldName];
            const spans = fieldsMap.get(fieldName) || [];
            spans.forEach(span => {
                if (span && span.textContent !== String(newValue)) {
                    span.textContent = String(newValue);
                }
            });
        }
        currentRevision = payload.revision;
        currentTemplateVersion = payload.template_version;
    }

    function handleTemplateChangedEvent(event) {
        let payload;
        try {
            payload = JSON.parse(event.data);
        } catch (error) {
            return;
        }
        if (!payload || !isSafePositiveInt(payload.template_version)) {
            return;
        }
        const valid = Number.isSafeInteger(payload.revision) && payload.revision > currentRevision && isSafePositiveInt(payload.template_version);
        if (valid && payload.template_version !== currentTemplateVersion) {
            reloadForTemplateChange();
        }
    }

    document.querySelectorAll('span[data-overlay-field]').forEach(span => {
        const name = span.dataset.overlayField;
        if (name) {
            if (!fieldsMap.has(name)) {
                fieldsMap.set(name, []);
            }
            fieldsMap.get(name).push(span);
        }
    });

    try {
        currentState = JSON.parse(stateEl.textContent);
        if (!currentState || !isSafePositiveInt(currentState.revision) || !isSafePositiveInt(currentState.template_version)) {
            return;
        }
        currentRevision = currentState.revision;
        currentTemplateVersion = currentState.template_version;
        initializeEventSource(currentState.events_url);
    } catch (error) {
        // Initial metadata is missing or unparsable; the runtime stays idle.
    }

    window.addEventListener('pagehide', () => {
        closeEventSource();
    });

    window.addEventListener('pageshow', (event) => {
        if (event.persisted && !hasReloaded && !eventSource && typeof currentState.events_url === 'string') {
            initializeEventSource(currentState.events_url);
        }
    });
})();
