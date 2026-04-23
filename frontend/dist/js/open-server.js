/* ══════════════════════════════════════════════════════
   OPEN SERVER MODAL
══════════════════════════════════════════════════════ */
async function showOpenServerModal() {
  const servers = await window.go.main.App.GetAllServers().catch(() => []);
  if (!servers || servers.length === 0) {
    await infoDialog('Sin servidores', '¡No hay servidores guardados!\nCrea uno nuevo primero.');
    return;
  }
  const list = el('div', { class: 'server-list' });
  servers.forEach(s => {
    const item = el('div', { class: 'server-list-item', onclick: () => { closeModal(); openServerByRecord(s); } },
      el('span', { class: 'sli-icon' }, typeIcon(s.Type)),
      el('div', {},
        el('div', { class: 'sli-name' }, s.Name),
        el('div', { class: 'sli-meta' }, s.Type + ' · MC ' + s.Version)
      )
    );
    list.appendChild(item);
  });

  const m = el('div', { class: 'modal' },
    el('div', { class: 'modal-header' },
      el('h3', {}, '📂 Tus Servidores'),
      el('button', { class: 'modal-close', onclick: closeModal }, '✕')
    ),
    el('div', { class: 'modal-body' },
      el('p', { class: 'muted', style:{ fontSize:'0.8rem' } }, servers.length + ' servidor(es) disponible(s)'),
      list
    ),
    el('div', { class: 'modal-footer' },
      el('button', { class: 'btn btn-ghost', onclick: closeModal }, 'Cancelar')
    )
  );
  openModal(m);
}

