/* ══════════════════════════════════════════════════════
   INSTALL PROGRESS VIEW
══════════════════════════════════════════════════════ */
function renderInstallProgress(title = 'Instalando...') {
  S.view = 'installing';
  const app = $('app');
  const bar  = el('div', { class: 'progress-bar-fill', id: 'prog-bar' });
  const text = el('div', { class: 'install-text', id: 'prog-text' }, 'Iniciando...');
  app.innerHTML = '';
  app.appendChild(
    el('div', { class: 'install-view' },
      el('div', { class: 'install-card' },
        el('div', { class: 'install-spinner' }, '⚙️'),
        el('div', { class: 'install-title' }, title),
        text,
        el('div', { class: 'progress-bar-wrap' }, bar),
      )
    )
  );
}

