/* ══════════════════════════════════════════════════════
   SERVER DASHBOARD
══════════════════════════════════════════════════════ */
function renderDashboard() {
  S.view = 'dashboard';
  const app = $('app');
  const layout = el('div', { class: 'dashboard-layout' },
    buildSidebar(),
    el('div', { class: 'content-area', id: 'content-area' })
  );
  app.innerHTML = '';
  app.appendChild(layout);
  switchDashView(S.dashView);
}

function buildSidebar() {
  const sb = el('div', { class: 'sidebar' });

  const header = el('div', { class: 'sidebar-header' },
    el('div', { class: 'sidebar-logo' }, 'GetMineHub'),
    el('div', { class: 'sidebar-powered' }, 'Powered by Claude IA'),
    el('div', { class: 'sidebar-server-name', id: 'sb-server-name' }, S.activeServer ? S.activeServer.Name : '')
  );

  // Sin jugadores — se eliminó esa sección
  const navItems = [
    { id: 'nav-console',    icon: '💻', label: 'Consola',     view: 'console'     },
    { id: 'nav-properties', icon: '📝', label: 'Propiedades', view: 'properties'  },
    { id: 'nav-options',    icon: '⚙️', label: 'Opciones',    view: 'options'     },
  ];

  const nav = el('div', { class: 'sidebar-nav' },
    ...navItems.map(n => {
      const btn = el('button', {
        class: 'nav-btn' + (S.dashView === n.view ? ' active' : ''),
        id: n.id,
        onclick: () => switchDashView(n.view)
      },
        el('span', { class: 'nav-icon' }, n.icon),
        n.label
      );
      return btn;
    })
  );

  const toastEl = el('div', { class: 'sidebar-toast', id: 'sidebar-toast' }, '');
  const menuBtn = el('button', {
    class: 'btn btn-ghost btn-full btn-sm',
    id: 'btn-main-menu',
    onclick: goToMainMenu
  }, '← Menú Principal');
  if (S.isRunning) menuBtn.disabled = true;

  const footer = el('div', { class: 'sidebar-footer' }, toastEl, menuBtn);

  sb.appendChild(header);
  sb.appendChild(nav);
  sb.appendChild(footer);
  return sb;
}

async function switchDashView(view) {
  S.dashView = view;
  $$('.nav-btn').forEach(b => b.classList.remove('active'));
  const navEl = $('nav-' + view);
  if (navEl) navEl.classList.add('active');
  const content = $('content-area');
  if (!content) return;
  if (view === 'console') {
    S.consoleText = await window.go.main.App.GetConsoleHistory().catch(() => S.consoleText);
    renderConsoleView(content);
  } else {
    switch (view) {
      case 'properties': renderPropertiesView(content); break;
      case 'options':    renderOptionsView(content);    break;
    }
  }
}

async function goToMainMenu() {
  if (S.isRunning) {
    showToast('Detén el servidor antes de volver al menú');
    return;
  }
  const ok = await confirmDialog('Volver al Menú', '¿Volver al Menú Principal?');
  if (!ok) return;
  window.go.main.App.CloseServer();
  S.activeServer = null;
  S.isRunning = false;
  S.consoleText = '';
  renderMainMenu();
}

function sidebarToast(msg) {
  const t = $('sidebar-toast');
  if (!t) return;
  t.textContent = msg;
  t.style.display = 'block';
  clearTimeout(sidebarToast._timer);
  sidebarToast._timer = setTimeout(() => { t.style.display = 'none'; }, 3000);
}
