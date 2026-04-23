/* ══════════════════════════════════════════════════════
   WAILS EVENT HANDLERS
══════════════════════════════════════════════════════ */
function setupWailsEvents() {
  const rt = window.runtime;
  if (!rt) { setTimeout(setupWailsEvents, 200); return; }

  rt.EventsOn('console:output', text => {
    appendToConsole(text);
  });

  rt.EventsOn('server:status', running => {
    updateServerStatusUI(running);
    if (running) {
      sidebarToast('✅ Servidor en línea');
    } else {
      sidebarToast('🔴 Servidor detenido');
    }
  });

  rt.EventsOn('server:graceful-fail', () => {
    const stopBtn = $('btn-stop');
    if (stopBtn) {
      stopBtn.textContent = '⚡ Forzar';
      stopBtn.className = 'btn btn-danger btn-sm';
      stopBtn.disabled = false;
    }
  });

  rt.EventsOn('install:progress', data => {
    if (!data) return;
    const bar  = $('prog-bar');
    const text = $('prog-text');

    if (data.error) {
      infoDialog('Error de Instalación', data.error).then(() => renderMainMenu());
      return;
    }
    if (bar)  bar.style.width = Math.round(data.progress * 100) + '%';
    if (text) text.textContent = data.text || 'Procesando...';

    if (data.success) {
      sidebarToast = () => {};  // prevent stale calls
      resetServerState();
      // Re-abrir el servidor recién instalado
      window.go.main.App.GetLastServerPath().then(path => {
        if (path) return window.go.main.App.OpenServer(path);
        throw new Error('no path');
      }).then(async srv => {
        S.activeServer = srv;
        // Restore toast function
        sidebarToast = function(msg) {
          const t = $('sidebar-toast');
          if (!t) return;
          t.textContent = msg;
          t.style.display = 'block';
          clearTimeout(sidebarToast._timer);
          sidebarToast._timer = setTimeout(() => { t.style.display = 'none'; }, 3000);
        };
        renderDashboard();
        const action = data.wasUpdate ? 'actualizado' : data.wasReinstall ? 'reinstalado' : 'creado';
        showToast('🎉 Servidor "' + (data.serverName || srv.Name) + '" ' + action + ' correctamente');
      }).catch(() => {
        renderMainMenu();
      });
    }
  });
}

