/* ══════════════════════════════════════════════════════
   SETTINGS MODAL
══════════════════════════════════════════════════════ */
async function showSettingsModal() {
  const s = await window.go.main.App.GetSettings();

  // NOTA: Wails serializa el struct de Go con sus json tags (snake_case):
  //   ServersBaseDir       → s.servers_base_dir
  //   MaxRAMLimit          → s.max_ram_limit
  //   CheckForUpdates      → s.check_for_updates
  //   NotificationsEnabled → s.notifications_enabled
  //   Timeout              → s.timeout
  const dirInput    = el('input', { type:'text',   value: s.servers_base_dir || '' });
  const ramInput    = el('input', { type:'number', value: String(s.max_ram_limit != null ? s.max_ram_limit : 0), placeholder:'0 = Sin límite' });
  const notifsCheck = el('input', { type:'checkbox' }); notifsCheck.checked = !!s.notifications_enabled;
  const updatesCheck= el('input', { type:'checkbox' }); updatesCheck.checked = !!s.check_for_updates;

  const m = el('div', { class: 'modal' },
    el('div', { class: 'modal-header' },
      el('h3', {}, '⚙️ Configuración'),
      el('button', { class: 'modal-close', onclick: closeModal }, '✕')
    ),
    el('div', { class: 'modal-body' },
      el('div', { class: 'form-group' },
        el('label', { class: 'form-label' }, '📁 DIRECTORIO DE SERVIDORES'),
        el('div', { style:{ display:'flex', gap:'8px' } },
          dirInput,
          el('button', { class: 'btn btn-secondary btn-sm', onclick: async () => {
            const dir = await window.go.main.App.SelectFolder().catch(() => null);
            if (dir) dirInput.value = dir;
          }}, '📁')
        )
      ),
      el('div', { class: 'form-group' },
        el('label', { class: 'form-label' }, '💾 LÍMITE DE RAM (MB, 0 = sin límite)'),
        ramInput
      ),
      el('div', { class: 'sep' }),
      el('label', { class: 'check-wrap' }, notifsCheck, el('span', {}, 'Notificaciones de Escritorio')),
      el('label', { class: 'check-wrap' }, updatesCheck, el('span', {}, 'Verificar actualizaciones al iniciar')),
    ),
    el('div', { class: 'modal-footer' },
      el('button', { class: 'btn btn-ghost', onclick: closeModal }, 'Cancelar'),
      el('button', { class: 'btn btn-primary', onclick: async () => {
        const ramVal = parseInt(ramInput.value, 10);
        if (isNaN(ramVal) || ramVal < 0) { showToast('El límite de RAM debe ser un número no negativo'); return; }
        if (!dirInput.value.trim()) { showToast('El directorio no puede estar vacío'); return; }
        // Enviamos con snake_case (json tags del struct de Go) para que Wails deserialice correctamente
        await window.go.main.App.SaveSettings({
          servers_base_dir:      dirInput.value.trim(),
          max_ram_limit:         ramVal,
          check_for_updates:     updatesCheck.checked,
          notifications_enabled: notifsCheck.checked,
          timeout:               s.timeout || 15,
        });
        closeModal();
        showToast('✅ Configuración guardada');
      }}, '💾 Guardar')
    )
  );
  openModal(m);
}

/* ══════════════════════════════════════════════════════
   OPEN SERVER → DASHBOARD
══════════════════════════════════════════════════════ */
async function openServerByRecord(srv) {
  try {
    const server = await window.go.main.App.OpenServer(srv.Path);
    S.activeServer = server;
    S.isRunning = await window.go.main.App.IsServerRunning();
    S.consoleText = await window.go.main.App.GetConsoleHistory();
    S.dashView = 'console';
    renderDashboard();
  } catch (e) {
    await infoDialog('Error', 'No se pudo abrir el servidor:\n' + String(e));
  }
}
