'use strict';
(() => {
  // Remove QR credentials from the address bar before any asynchronous work.
  const params = new URLSearchParams(location.hash.slice(1));
  let qrToken = params.get('qr_token');
  if (location.hash) history.replaceState(null, '', location.pathname + location.search);
  const feedback = document.getElementById('access-feedback');
  const say = (text) => { feedback.textContent = text; };
  const jsonPost = (body) => ({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  const page = document.body.dataset.accessPage;
  if (page === 'login') {
    const form = document.getElementById('pair-form');
    const code = document.getElementById('pair-code');
    const name = document.getElementById('device-name');
    let busy = false;
    async function pair(token) {
      if (busy) return;
      busy = true;
      try {
        const payload = { device_name: name.value.trim() || 'Yeni cihaz' };
        if (token) payload.qr_token = token; else payload.code = code.value;
        const options = jsonPost(payload);
        options.headers['X-Overlay-Request'] = '1';
        const response = await fetch('/access/pair', options);
        if (response.status === 429) { say('Çok fazla hatalı deneme. Birkaç dakika sonra tekrar deneyin.'); return; }
        if (!response.ok) throw new Error('pair failed');
        location.assign('/');
      } catch (_) { say('Eşleştirme başarısız. Kodu kontrol edip tekrar deneyin.'); }
      finally { busy = false; }
    }
    form.addEventListener('submit', (event) => { event.preventDefault(); if (form.reportValidity()) pair(null); });
    document.getElementById('local-login').addEventListener('click', async () => {
      if (busy) return;
      busy = true;
      try {
        const options = jsonPost({}); options.headers['X-Overlay-Request'] = '1';
        const response = await fetch('/access/local', options);
        if (!response.ok) throw new Error('local failed');
        location.assign('/');
      } catch (_) { say('Bu işlem yalnızca yerel bilgisayarda kullanılabilir.'); }
      finally { busy = false; }
    });
    overlayAccess.session().then(() => location.assign('/')).catch(() => {
      if (qrToken) { const token = qrToken; qrToken = null; pair(token); }
      else if (params.get('code') && /^[0-9]{6}$/.test(params.get('code'))) { code.value = params.get('code'); pair(null); }
    });
    return;
  }
  if (page !== 'devices') return;
  const result = document.getElementById('pairing-result');
  const image = document.getElementById('pair-qr');
  const display = document.getElementById('display-code');
  const expiry = document.getElementById('pair-expiry');
  const select = document.getElementById('pair-origin');
  let timer;
  function clearPairing() { clearTimeout(timer); result.hidden = true; image.removeAttribute('src'); display.textContent = ''; expiry.textContent = ''; }
  async function loadDevices() {
    const response = await fetch('/access/devices', { cache: 'no-store' });
    if (response.status === 401) { location.assign('/access'); return; }
    if (!response.ok) throw new Error('devices failed');
    const data = await response.json();
    const devices = Array.isArray(data) ? data : data.devices;
    if (!Array.isArray(devices)) throw new Error('invalid devices');
    const list = document.getElementById('devices-list');
    list.replaceChildren();
    devices.forEach((device) => {
      const item = document.createElement('li'); item.className = 'list-group-item d-flex flex-wrap gap-2 align-items-center';
      const text = document.createElement('span');
      text.textContent = `${device.device_name} — Oluşturulma: ${new Date(device.created_at).toLocaleString('tr-TR')} — Bitiş: ${new Date(device.expires_at).toLocaleString('tr-TR')}`;
      const button = document.createElement('button'); button.type = 'button'; button.className = 'btn btn-subtle ms-auto'; button.textContent = 'Erişimi kaldır';
      button.addEventListener('click', async () => {
        if (!confirm('Bu cihazın erişimi kaldırılsın mı?')) return;
        button.disabled = true;
        try {
          const response = await overlayAccess.mutate('/access/devices/revoke', jsonPost({ id: device.id }));
          if (!response.ok) throw new Error('revoke failed');
          await loadDevices();
        } catch (_) { say('Cihazın erişimi kaldırılamadı.'); button.disabled = false; }
      });
      item.append(text, button); list.appendChild(item);
    });
  }
  document.getElementById('logout').addEventListener('click', async () => {
    try { const response = await overlayAccess.mutate('/access/logout', jsonPost({})); if (!response.ok) throw new Error('logout'); location.assign('/access'); }
    catch (_) { say('Çıkış yapılamadı.'); }
  });
  document.getElementById('revoke-all').addEventListener('click', async () => {
    if (!confirm('Tüm cihazların erişimi kaldırılsın mı? Bu cihazdan da çıkış yapılacak.')) return;
    try { const response = await overlayAccess.mutate('/access/devices/revoke', jsonPost({ all: true })); if (!response.ok) throw new Error('revoke'); location.assign('/access'); }
    catch (_) { say('Erişimler kaldırılamadı.'); }
  });
  document.getElementById('create-pairing').addEventListener('click', async (event) => {
    const button = event.currentTarget;
    button.disabled = true; clearPairing();
    try {
      const response = await overlayAccess.mutate('/access/pairing', jsonPost({ origin: select.value }));
      if (!response.ok) throw new Error('pairing');
      const data = await response.json();
      const remaining = Date.parse(data.expires_at) - Date.now();
      if (!/^[0-9]{6}$/.test(data.code) || !/^data:image\/png;base64,[A-Za-z0-9+/=]+$/.test(data.qr_image) || !(remaining > 0)) throw new Error('invalid pairing');
      image.src = data.qr_image; display.textContent = data.code;
      expiry.textContent = 'Son geçerlilik: ' + new Date(data.expires_at).toLocaleString('tr-TR'); result.hidden = false;
      timer = setTimeout(() => { clearPairing(); say('Eşleştirme kodunun süresi doldu.'); }, remaining);
      say('');
    } catch (_) { say('Eşleştirme kodu oluşturulamadı.'); }
    finally { button.disabled = false; }
  });
  overlayAccess.session().then(async (info) => {
    if (!info.local) { document.getElementById('remote-note').hidden = false; return; }
    document.getElementById('local-controls').hidden = false;
    const origins = Array.isArray(info.admin_origins) ? info.admin_origins : [];
    let lan = false;
    origins.forEach((origin) => {
      const option = document.createElement('option'); option.value = origin; option.textContent = origin; select.appendChild(option);
      try { const host = new URL(origin).hostname; if (host !== 'localhost' && host !== '[::1]' && !/^127\./.test(host)) { if (!lan) select.value = origin; lan = true; } } catch (_) { /* invalid origin */ }
    });
    document.getElementById('lan-warning').hidden = lan;
    await loadDevices();
  }).catch(() => { say('Oturum veya cihaz listesi yüklenemedi.'); });
})();
