'use strict';
window.overlayAccess = (() => {
  let cached;
  async function session() {
    if (!cached) {
      cached = fetch('/access/session', { cache: 'no-store' }).then(async (response) => {
        if (!response.ok) throw new Error('Oturum bulunamadı');
        const data = await response.json();
        if (typeof data.csrf_token !== 'string' || !data.csrf_token) throw new Error('CSRF belirteci eksik');
        return data;
      }).catch((err) => { cached = undefined; throw err; });
    }
    return cached;
  }
  async function mutate(url, options = {}) {
    const info = await session();
    const headers = new Headers(options.headers || {});
    headers.set('X-CSRF-Token', info.csrf_token);
    return fetch(url, { ...options, headers });
  }
  return { session, mutate };
})();
