/* ══════════════════════════════════════════════════════
   JAVA VIEW — Gestión de versiones de Java
══════════════════════════════════════════════════════ */

async function renderJavaView() {
  S.view = 'java';
  const app = $('app');
  app.innerHTML = '';

  const page = el('div', { class: 'java-page' });

  // Header con botón de volver
  const header = el('div', { class: 'java-header' },
    el('button', {
      class: 'btn btn-ghost btn-sm',
      onclick: () => renderMainMenu()
    }, '← Menú Principal'),
    el('div', { class: 'java-header-title' },
      el('span', {}, '☕'),
      el('div', {},
        el('div', { class: 'java-title' }, 'Gestión de Java'),
        el('div', { class: 'java-subtitle' }, 'Administra las versiones de Java para tus servidores')
      )
    )
  );

  const content = el('div', { class: 'java-content', id: 'java-content' });
  page.appendChild(header);
  page.appendChild(content);
  app.appendChild(page);

  await refreshJavaContent();

  // Escuchar progreso de descarga
  if (window.runtime) {
    window.runtime.EventsOn('java:download-progress', handleJavaDownloadProgress);
  }
}

async function refreshJavaContent() {
  const content = $('java-content');
  if (!content) return;
  content.innerHTML = '';

  const [installed, options] = await Promise.all([
    window.go.main.App.GetInstalledJavas().catch(() => []),
    window.go.main.App.GetJavaDownloadOptions().catch(() => []),
  ]);

  // ── Sección: Javas instalados ──────────────────────────────────────────────
  const installedSection = el('div', { class: 'java-section' },
    el('div', { class: 'java-section-title' }, '✅ Versiones Instaladas')
  );

  if (!installed || installed.length === 0) {
    installedSection.appendChild(
      el('div', { class: 'java-empty' },
        el('div', { class: 'java-empty-icon' }, '☕'),
        el('div', {}, 'No hay versiones de Java instaladas localmente.'),
        el('div', { class: 'muted', style: { fontSize: '0.82rem', marginTop: '4px' } },
          'Descarga una versión a continuación para usarla en tus servidores.')
      )
    );
  } else {
    const grid = el('div', { class: 'java-installed-grid' });
    installed.forEach(j => {
      grid.appendChild(
        el('div', { class: 'java-installed-card' },
          el('div', { class: 'java-installed-icon' }, '☕'),
          el('div', { class: 'java-installed-info' },
            el('div', { class: 'java-installed-name' }, j.name),
            el('div', { class: 'java-installed-path muted' }, j.path)
          ),
          el('div', { class: 'java-version-badge' }, 'Java ' + j.version)
        )
      );
    });
    installedSection.appendChild(grid);
  }

  // ── Sección: Disponibles para descargar ────────────────────────────────────
  const downloadSection = el('div', { class: 'java-section' },
    el('div', { class: 'java-section-title' }, '⬇️ Versiones Disponibles para Descargar')
  );

  const downloadGrid = el('div', { class: 'java-download-grid' });

  if (!options || options.length === 0) {
    downloadGrid.appendChild(el('div', { class: 'muted' }, 'No hay versiones configuradas.'));
  } else {
    options.forEach(opt => {
      const isDownloading = S.java.downloading && S.java.downloadingVersion === opt.javaVersion;

      const downloadBtn = el('button', {
        class: 'btn btn-primary btn-sm',
        id: 'java-btn-' + opt.javaVersion,
        disabled: opt.installed || isDownloading,
        onclick: () => startJavaDownload(opt.javaVersion, opt.name)
      }, opt.installed ? '✅ Instalado' : '⬇️ Descargar');

      const card = el('div', {
        class: 'java-download-card' + (opt.installed ? ' java-card-installed' : ''),
        id: 'java-card-' + opt.javaVersion
      },
        el('div', { class: 'java-download-card-header' },
          el('div', { class: 'java-download-icon' }, '☕'),
          el('div', { class: 'java-download-info' },
            el('div', { class: 'java-download-name' }, opt.name),
            el('div', { class: 'java-download-meta muted' },
              'Requerido para Minecraft ' + opt.mcVersion + (opt.mcVersion === '0.0.0' ? '+' : '+'))
          ),
          downloadBtn
        )
      );

      downloadGrid.appendChild(card);
    });
  }

  downloadSection.appendChild(downloadGrid);

  // Área de progreso global
  const globalProgress = el('div', {
    class: 'java-global-progress',
    id: 'java-global-progress',
    style: { display: 'none' }
  });

  content.appendChild(installedSection);
  content.appendChild(downloadSection);
  content.appendChild(globalProgress);
}

async function startJavaDownload(javaVersion, name) {
  if (S.java.downloading) {
    showToast('Ya hay una descarga en progreso.');
    return;
  }

  const ok = await confirmDialog(
    'Descargar Java',
    `¿Descargar e instalar ${name}?\n\nLa descarga puede tardar varios minutos dependiendo de tu conexión.`
  );
  if (!ok) return;

  setJavaDownloading(javaVersion);

  // Mostrar la pantalla de progreso global (igual que al crear/actualizar un servidor)
  renderInstallProgress('Descargando ' + name + '...');

  window.go.main.App.DownloadJava(javaVersion);
}

function handleJavaDownloadProgress(evt) {
  if (!evt) return;

  const bar  = $('prog-bar');
  const text = $('prog-text');

  if (evt.error) {
    clearJavaDownloading();
    infoDialog('Error al descargar Java', evt.error).then(() => renderJavaView());
    return;
  }

  if (bar)  bar.style.width  = Math.round((evt.progress || 0) * 100) + '%';
  if (text) text.textContent = evt.text || 'Descargando...';

  if (evt.success) {
    clearJavaDownloading();
    showToast('☕ ' + (evt.text || 'Java instalado correctamente.'));
    // Volver a la vista de Java con el nuevo runtime ya visible
    setTimeout(() => renderJavaView(), 800);
  }
}
