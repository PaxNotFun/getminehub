/* ══════════════════════════════════════════════════════
   INIT
══════════════════════════════════════════════════════ */
async function init() {
  setupWailsEvents();

  // Check for updates in background
  window.go.main.App.CheckForUpdates().then(async res => {
    if (res && res.available) {
      const ok = await confirmDialog('Actualización Disponible',
        'Versión instalada: ' + res.current + '\nNueva versión: ' + res.latest + '\n\n¿Ir a la página de descarga?');
      if (ok) {
        const url = await window.go.main.App.GetDownloadURL().catch(() => '');
        if (url) window.runtime && window.runtime.BrowserOpenURL(url);
      }
    }
  }).catch(() => {});

  // Try to reopen last server
  try {
    const lastPath = await window.go.main.App.GetLastServerPath();
    if (lastPath) {
      const srv = await window.go.main.App.OpenServer(lastPath).catch(() => null);
      if (srv) {
        S.activeServer = srv;
        S.isRunning = await window.go.main.App.IsServerRunning();
        S.consoleText = await window.go.main.App.GetConsoleHistory().catch(() => '');
        S.dashView = 'console';
        renderDashboard();
        return;
      }
    }
  } catch (_) {}

  renderMainMenu();
}

// Wait for DOM and Wails runtime
window.addEventListener('DOMContentLoaded', () => {
  // Wails injects window.go and window.runtime before DOMContentLoaded in most cases
  // but we add a small safety wait
  if (typeof window.go !== 'undefined') {
    init();
  } else {
    const check = setInterval(() => {
      if (typeof window.go !== 'undefined') {
        clearInterval(check);
        init();
      }
    }, 50);
  }
});
