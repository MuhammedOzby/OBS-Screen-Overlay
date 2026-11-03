'use strict';
const panelEditorMutation = {
  pending: false,
  setBusy: function (busy) {
    this.pending = busy;
    const applyButton = document.getElementById('apply-button');
    const publishButton = document.getElementById('publish-button');
    if (applyButton) applyButton.disabled = busy;
    if (publishButton) publishButton.disabled = busy;
  }
};

(function () {
  const input = document.getElementById('obs-url');
  const button = document.getElementById('copy-obs-url');
  const feedback = document.getElementById('copy-feedback');
  if (input === null || button === null || feedback === null) { return; }
  button.addEventListener('click', async () => {
    try {
      if (!navigator.clipboard || !navigator.clipboard.writeText) { throw new Error('clipboard unavailable'); }
      await navigator.clipboard.writeText(input.value);
      feedback.textContent = 'OBS bağlantısı kopyalandı.';
    } catch (err) {
      input.focus();
      input.select();
      feedback.textContent = 'Bağlantı seçildi; kopyalamak için Ctrl+C kullanın.';
    }
  });
})();

// START TEMPLATE FORM PUBLISHING LOGIC
(function () {
  const form = document.getElementById('template-form');
  const fileInput = document.getElementById('html-file');
  const publishButton = document.getElementById('publish-button');
  const editForm = document.getElementById('edit-form');
  if (form === null || fileInput === null || publishButton === null || editForm === null) { return; }

  fileInput.disabled = false;

  publishButton.disabled = false;

  // --- Validation function ---
  const validateTemplateForm = () => {
    if (fileInput.files.length !== 1) { 
      alert('Lütfen tam olarak bir .html dosyası seçin.'); 
      return false; 
    }
    const file = fileInput.files[0];
    if (!/\.html$/i.test(file.name)) { 
      alert('Yalnızca .html uzantılı dosya seçin.'); 
      return false; 
    }
    if (file.size < 1 || file.size > 2097152) { 
      alert('Dosya boyutu 1 - 2MB arasında olmalıdır.'); 
      return false; 
    }
    return true;
  };
  
  // --- Submit handler ---
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (panelEditorMutation.pending) { 
      document.getElementById('demo-feedback').textContent = 'Lütfen bekleyin, tasarım şu anda yayınlanıyor.';
      return;
    }
    if (!validateTemplateForm()) { 
      return; 
    }

    if (!window.confirm('Bu tasarımı yayınlamak istediğinizden emin misiniz?')) {
      return;
    }

    panelEditorMutation.setBusy(true);


    
    // Prepare FormData
    const formData = new FormData();
    formData.append('html', fileInput.files[0]);
    formData.append('expected_revision', editForm ? editForm.dataset.revision : '0');

    try {
      const response = await window.overlayAccess.mutate(`/api/pages/${encodeURIComponent(editForm ? editForm.dataset.id : '')}/template`, {
        method: 'POST',
        body: formData,
      });

      let data = null;
      try {
        data = await response.json();
      } catch (err) {
        // Failed to parse JSON
      }

      if (response.ok) {
        // Success
        if (data && typeof data.revision === 'number') {
          editForm.dataset.revision = data.revision;
          form.reset();
          document.getElementById('demo-feedback').textContent = 'Tasarım yayınlandı. Açık OBS sayfaları yenilenir.';
        } else {
          document.getElementById('demo-feedback').textContent = 'Tasarım başarıyla gönderildi, ancak revizyon güncellenemedi.';
        }
      } else if (response.status === 409) {
        // Conflict
        document.getElementById('reload-record').hidden = false;
        // Preserve selected file and edit draft
        document.getElementById('demo-feedback').textContent = '409 Conflict: Tasarım başka bir yerden güncellendi. Güncel kaydı yükleyin ve taslağınızı koruyun.';
      } else {
        // Other errors (Turkish text required)
        let errorMsg = 'Tasarım yayınlanırken bir hata oluştu. Lütfen tekrar deneyin.';
        if (data && typeof data.error === 'string' && data.error !== '') {
          errorMsg = data.error;
        } else {
           errorMsg = 'Sunucudan bilinmeyen bir hata alındı.';
        }
        document.getElementById('demo-feedback').textContent = errorMsg;
      }

    } catch (err) {
      // Network or fetch error
      document.getElementById('demo-feedback').textContent = 'Ağ hatası: Sunucuya ulaşılamadı veya yayınlama başarısız oldu.';
    } finally {
      panelEditorMutation.setBusy(false)
    }
  });
})();
// END TEMPLATE FORM PUBLISHING LOGIC

(function () {
  const MAX_SIZE = 2097152;
  const NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
  const feedback = document.getElementById('demo-feedback');
  const say = (msg) => {
    if (feedback !== null) { feedback.textContent = msg; }
  };
  const forms = ['new-form', 'edit-form']
    .map((id) => document.getElementById(id))
    .filter((f) => f !== null);
  forms.forEach((form) => {
    form.newCreate = false;
    const btn = form.querySelector('button[type="submit"]');
    if (btn !== null) { btn.disabled = false; }
    const validate = () => {
      if (!form.reportValidity()) {
        say('Lütfen zorunlu alanları doğru doldurun.');
        return false;
      }
      const name = form.querySelector('#record-name');
      if (name !== null && name.value.trim() === '') {
        say('Kayıt ismi boş olamaz.');
        return false;
      }
      const file = form.querySelector('#html-file');
      if (file !== null) {
        if (file.files.length !== 1) { say('Tam olarak bir HTML dosyası seçin.'); return false; }
        const f = file.files[0];
        if (!/\.html$/i.test(f.name)) { say('Yalnızca .html uzantılı dosya seçin.'); return false; }
        if (f.size <= 0) { say('Seçilen dosya boş olamaz.'); return false; }
        if (f.size > MAX_SIZE) { say('Dosya boyutu en fazla 2 MB olabilir.'); return false; }
      }
      if (form.id === 'edit-form') {
        const seen = Object.create(null);
        for (const input of form.querySelectorAll('.field-row input[type="text"]')) {
          const v = input.value;
          if (!NAME_RE.test(v)) { say('Değişken adları [a-zA-Z_][a-zA-Z0-9_]* biçiminde olmalıdır.'); return false; }
          if (seen[v] === true) { say('Aynı değişken adı tekrar kullanılamaz: ' + v); return false; }
          seen[v] = true;
        }
      }
      return true;
    };
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (form.newCreate || panelEditorMutation.pending) { return; }
      if (!validate()) { return; }
      if (form.id === 'new-form') {
        form.newCreate = true
        if (btn !== null) { btn.disabled = true; }
        let redirect = null;
        try {
          const response = await window.overlayAccess.mutate('/api/pages', { method: 'POST', body: new FormData(form) });
          let data = null;
          try {
            data = await response.json();
          } catch (err) {
            /* yanıt JSON değil (örn. hata sayfası) */
          }
          if (response.ok && data !== null && typeof data.redirect === 'string' && data.redirect !== '') {
            redirect = data.redirect;
          } else {
            let message = 'Kayıt oluşturulamadı. Lütfen tekrar deneyin.';
            if (data !== null && typeof data.error === 'string' && data.error !== '') {
              message = data.error;
            }
            say(message);
          }
        } catch (err) {
          say('Ağ hatası: sunucuya ulaşılamadı. Lütfen tekrar deneyin.');
        } finally {
          if (redirect === null) {
            if (btn !== null) { btn.disabled = false; }
            form.newCreate = false;
          } else {
            form.newCreate = true;
          }
        }
        if (redirect !== null) { location.assign(redirect); }
        return;
      }
      if (form.id === 'edit-form') {
        panelEditorMutation.setBusy(true)
        if (btn !== null) { btn.disabled = true; }
        try {
          const payload = {
            expected_revision: Number(form.dataset.revision),
            name: form.querySelector('#record-name').value,
            type: form.querySelector('#record-type').value,
            fields: Array.from(form.querySelectorAll('.field-row'), (row) => ({
              name: row.querySelector('input[type="text"]').value,
              value: row.querySelector('textarea').value,
              sse_enabled: row.querySelector('input[type="checkbox"]').checked
            }))
          };
          const response = await window.overlayAccess.mutate('/api/pages/' + encodeURIComponent(form.dataset.id), {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
          });
          let data = null;
          try {
            data = await response.json();
          } catch (err) {
            /* yanıt JSON değil (örn. hata sayfası) */
          }
          if (response.ok && data !== null && Number.isInteger(data.revision)) {
            form.dataset.revision = String(data.revision);
            document.title = 'OBS Screen Overlay — ' + payload.name + ' düzenle';
            say('Değişiklikler kaydedildi. SSE açık alanlar yayına uygulandı.');
          } else if (response.status === 409) {
            const reload = document.getElementById('reload-record');
            if (reload !== null) { reload.hidden = false; }
            let message = '409: Kayıt başka bir yerden değişti. Taslağınız korundu; güncel kaydı yükleyip tekrar deneyebilirsiniz.';
            if (data !== null && typeof data.error === 'string' && data.error !== '') {
              message += ' Sunucu: ' + data.error;
            }
            say(message);
          } else {
            let message = 'Kayıt güncellenemedi. Lütfen tekrar deneyin. (Taslağınız korundu.)';
            if (data !== null && typeof data.error === 'string' && data.error !== '') {
              message = data.error + ' (Taslağınız korundu.)';
            }
            say(message);
          }
        } catch (err) {
          say('Ağ hatası: sunucuya ulaşılamadı. Taslağınız korundu, lütfen tekrar deneyin.');
        } finally {
          if (btn !== null) { btn.disabled = false; }
          panelEditorMutation.setBusy(false)
        }
        return;
      }
    });
  });
})();
(function () {
  const list = document.getElementById('field-list');
  const add = document.getElementById('add-field');
  if (list === null || add === null) { return; }
  add.addEventListener('click', () => {
    const row = document.createElement('div');
    row.className = 'field-row mb-3 d-flex flex-wrap gap-2 align-items-start';
    const nameLabel = document.createElement('label');
    nameLabel.className = 'field-name flex-grow-1 min-vw-25';
    nameLabel.textContent = 'Değişken adı';
    const nameInput = document.createElement('input');
    nameInput.type = 'text';
    nameInput.className = 'form-control';
    nameInput.name = 'field_name';
    nameInput.required = true;
    nameInput.pattern = '[a-zA-Z_][a-zA-Z0-9_]*';
    nameLabel.appendChild(nameInput);
    const valueLabel = document.createElement('label');
    valueLabel.className = 'field-value flex-grow-1 min-vw-25';
    valueLabel.textContent = 'Değer';
    const valueArea = document.createElement('textarea');
    valueArea.className = 'form-control';
    valueArea.name = 'field_value';
    valueArea.rows = 2;
    valueLabel.appendChild(valueArea);
    const sseLabel = document.createElement('label');
    sseLabel.className = 'field-sse d-flex align-items-center gap-2';
    const sseBox = document.createElement('input');
    sseBox.type = 'checkbox';
    sseBox.className = 'form-check-input';
    sseBox.name = 'field_sse';
    sseBox.checked = true;
    sseLabel.appendChild(sseBox);
    sseLabel.append('SSE ile canlı güncelle');
    const removeBtn = document.createElement('button');
    removeBtn.type = 'button';
    removeBtn.className = 'btn btn-sm remove-field';
    removeBtn.textContent = 'Kaldır';
    row.append(nameLabel, valueLabel, sseLabel, removeBtn);
    list.appendChild(row);
    nameInput.focus();
  });
  list.addEventListener('click', (e) => {
    const btn = e.target.closest('.remove-field');
    if (btn !== null) { const r = btn.closest('.field-row'); if (r !== null) { r.remove(); } }
  });
})();
(function () {
  const buttons = Array.from(document.querySelectorAll('.delete-record'));
  if (buttons.length === 0) { return; }
  const feedback = document.getElementById('delete-feedback');
  const say = (msg) => {
    if (feedback !== null) { feedback.textContent = msg; }
  };
  const emptyNote = document.getElementById('empty-records');
  const tbody = document.querySelector('.records-table tbody');
  const syncEmptyNote = () => {
    if (emptyNote === null || tbody === null) { return; }
    emptyNote.hidden = tbody.querySelectorAll('tr').length > 0;
  };
  const CONFIRM_SUFFIX = 'Bu kayıt ve tüm değişkenleri kalıcı olarak silinecek. Bu işlem geri alınamaz. Devam edilsin mi?';
  buttons.forEach((btn) => {
    btn.disabled = false;
    let pending = false;
    btn.addEventListener('click', async () => {
      if (pending) { return; }
      const id = (btn.getAttribute('data-id') || '').trim();
      const name = (btn.getAttribute('data-name') || '').trim();
      if (id === '') { say('Kayıt kimliği eksik. Lütfen sayfa yenilendikten sonra silmeyi deneyin.'); return; }
      const ok = window.confirm(name + ' ' + CONFIRM_SUFFIX);
      if (!ok) { return; }
      btn.disabled = true;
      let removed = false;
      try {
        const response = await window.overlayAccess.mutate('/api/pages/' + encodeURIComponent(id), { method: 'DELETE' });
        if (response.status === 204 || response.status === 404) {
          const tr = btn.closest('tr');
          if (tr !== null) { tr.remove(); removed = true; syncEmptyNote(); }
          if (response.status === 404) {
            say(name + ' kaydı zaten silinmişti; listeden kaldırıldı.');
          } else {
            say(name + ' kaydı silindi.');
          }
        } else {
          let data = null;
          try { data = await response.json(); } catch (err) { /* gövde JSON değil (örn. hata sayfası) */ }
          let message = 'Kayıt silinemedi. Lütfen tekrar deneyin.';
          if (data !== null && typeof data.error === 'string' && data.error !== '') {
            message = data.error;
          }
          say(message);
        }
      } catch (err) {
        say('Ağ hatası: sunucuya ulaşılamadı. Lütfen tekrar deneyin.');
      } finally {
        if (!removed) { btn.disabled = false; }
      }
    });
  });
})();
