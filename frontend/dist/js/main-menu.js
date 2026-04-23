/* ══════════════════════════════════════════════════════
   MAIN MENU
══════════════════════════════════════════════════════ */
async function renderMainMenu() {
  S.view = 'main-menu';
  const app = $('app');

  const [dashData, recent] = await Promise.all([
    window.go.main.App.GetDashboardData(),
    window.go.main.App.GetRecentServers(),
  ]);

  const recentSection = recent && recent.length ? el('div', { class: 'recent-section' },
    el('div', { class: 'recent-header' }, '🕒 Servidores Recientes'),
    el('div', { class: 'recent-list' },
      ...recent.map(s => {
        const item = el('div', { class: 'recent-item' },
          el('div', { class: 'recent-info' },
            el('span', { class: 'recent-icon' }, typeIcon(s.Type)),
            el('div', {},
              el('div', { class: 'recent-name' }, s.Name),
              el('div', { class: 'recent-meta' }, s.Type + ' · MC ' + s.Version)
            )
          ),
          el('button', { class: 'btn btn-secondary btn-sm', onclick: () => openServerByRecord(s) }, 'Abrir →')
        );
        return item;
      })
    )
  ) : null;

  app.innerHTML = '';
  const menu = el('div', { class: 'main-menu' },
    el('div', { class: 'menu-hero' },
      el('div', {},
        el('div', { class: 'menu-title' }, 'GetMineHub'),
        el('div', { class: 'menu-subtitle' }, 'Gestiona servidores Minecraft con estilo')
      ),
    ),
    el('div', { class: 'stats-grid' },
      statCard('🎮', String(dashData.totalServers), 'Servidores Totales'),
      statCard('💾', fmtSize(dashData.usedGB), 'Espacio Usado'),
      statCard('⚡', dashData.mostType || 'N/A', 'Tipo Más Usado'),
      statCard('📦', dashData.mostVersion || 'N/A', 'Versión Popular'),
    ),
    ...(recentSection ? [recentSection] : []),
    el('div', { class: 'action-grid action-grid-4' },
      actionCard('✨', 'Crear Servidor', 'Instala un nuevo servidor\ndesde cero', showNewServerModal),
      actionCard('📂', 'Abrir Servidor', 'Gestiona un servidor\nexistente', showOpenServerModal),
      actionCard('☕', 'Gestión de Java', 'Administra versiones\nde Java instaladas', renderJavaView),
      actionCard('⚙️', 'Configuración', 'Personaliza tu\nexperiencia', showSettingsModal),
    )
  );
  app.appendChild(menu);
}

function statCard(icon, value, label) {
  return el('div', { class: 'stat-card' },
    el('span', { class: 'stat-icon' }, icon),
    el('div', {},
      el('div', { class: 'stat-value' }, value),
      el('div', { class: 'stat-label' }, label),
    )
  );
}
function actionCard(icon, title, desc, onClick) {
  return el('div', { class: 'action-card', onclick: onClick },
    el('div', { class: 'action-card-icon' }, icon),
    el('div', { class: 'action-card-title' }, title),
    el('div', { class: 'action-card-desc' }, desc),
  );
}
