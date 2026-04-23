/* ══════════════════════════════════════════════════════
   MODAL SYSTEM
══════════════════════════════════════════════════════ */
function openModal(modalEl) {
  const overlay = $('modal-overlay');
  overlay.innerHTML = '';
  overlay.appendChild(modalEl);
  overlay.classList.add('open');
  overlay.onclick = e => { if (e.target === overlay) closeModal(); };
}
function closeModal() {
  $('modal-overlay').classList.remove('open');
}

function confirmDialog(title, message, isDanger = false) {
  return new Promise(resolve => {
    const m = el('div', { class: 'modal modal-sm' },
      el('div', { class: 'modal-header' },
        el('h3', {}, title),
        el('button', { class: 'modal-close', onclick: () => { closeModal(); resolve(false); } }, '✕')
      ),
      el('div', { class: 'modal-body' },
        el('div', { class: 'confirm-message' }, message)
      ),
      el('div', { class: 'modal-footer' },
        el('button', { class: 'btn btn-ghost', onclick: () => { closeModal(); resolve(false); } }, 'Cancelar'),
        el('button', { class: 'btn ' + (isDanger ? 'btn-danger' : 'btn-primary'), onclick: () => { closeModal(); resolve(true); } }, 'Confirmar')
      )
    );
    openModal(m);
  });
}

function infoDialog(title, message) {
  return new Promise(resolve => {
    const m = el('div', { class: 'modal modal-sm' },
      el('div', { class: 'modal-header' },
        el('h3', {}, title),
        el('button', { class: 'modal-close', onclick: () => { closeModal(); resolve(); } }, '✕')
      ),
      el('div', { class: 'modal-body' },
        el('div', { class: 'confirm-message' }, message)
      ),
      el('div', { class: 'modal-footer' },
        el('button', { class: 'btn btn-primary', onclick: () => { closeModal(); resolve(); } }, 'Aceptar')
      )
    );
    openModal(m);
  });
}

