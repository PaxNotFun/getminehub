/* ══════════════════════════════════════════════════════
   NEW SERVER MODAL
══════════════════════════════════════════════════════ */
async function showNewServerModal() {
  const hasNet = await window.go.main.App.CheckInternetConnection();
  if (!hasNet) { await infoDialog('Sin Internet', 'No se puede crear un servidor sin conexión a internet.'); return; }

  const types = ['Vanilla','PaperMC','Folia','Forge','Fabric'];
  let versions = [];

  const typeSelect    = el('select', {});
  const versionSelect = el('select', { disabled: true });
  const nameInput     = el('input',  { type: 'text', placeholder: 'Mi Servidor Epico' });
  const versionStatus = el('span', { class: 'muted', style: { fontSize: '0.75rem' } }, '⟳ Cargando...');

  types.forEach(t => typeSelect.appendChild(el('option', { value: t }, t)));
  typeSelect.value = 'Vanilla';

  async function loadVersions(type) {
    versionSelect.disabled = true;
    versionSelect.innerHTML = '<option>Cargando...</option>';
    versionStatus.textContent = '⟳ Consultando...';
    const vs = await window.go.main.App.GetVersions(type).catch(() => []);
    versionSelect.innerHTML = '';
    if (vs && vs.length) {
      vs.forEach(v => versionSelect.appendChild(el('option', { value: v }, v)));
      versionSelect.disabled = false;
      versionStatus.textContent = '✅ ' + vs.length + ' versiones';
    } else {
      versionSelect.appendChild(el('option', {}, 'Sin versiones'));
      versionStatus.textContent = '❌ Sin conexión';
    }
  }
  typeSelect.onchange = () => loadVersions(typeSelect.value);
  loadVersions('Vanilla');

  const m = el('div', { class: 'modal' },
    el('div', { class: 'modal-header' },
      el('h3', {}, '✨ Crear Nuevo Servidor'),
      el('button', { class: 'modal-close', onclick: closeModal }, '✕')
    ),
    el('div', { class: 'modal-body' },
      el('div', { class: 'form-group' },
        el('label', { class: 'form-label' }, 'TIPO DE SERVIDOR'),
        typeSelect
      ),
      el('div', { class: 'form-group' },
        el('div', { style: { display:'flex', justifyContent:'space-between', alignItems:'center' } },
          el('label', { class: 'form-label' }, 'VERSIÓN DE MINECRAFT'),
          versionStatus
        ),
        versionSelect
      ),
      el('div', { class: 'form-group' },
        el('label', { class: 'form-label' }, 'NOMBRE DEL SERVIDOR'),
        nameInput
      )
    ),
    el('div', { class: 'modal-footer' },
      el('button', { class: 'btn btn-ghost', onclick: closeModal }, 'Cancelar'),
      el('button', { class: 'btn btn-primary', onclick: async () => {
        const name    = nameInput.value.trim();
        const version = versionSelect.value;
        const type    = typeSelect.value;
        if (!name) { showToast('Ingresa un nombre para el servidor'); return; }
        if (!version || version === 'Sin versiones' || version === 'Cargando...') {
          showToast('Selecciona una versión válida'); return; }
        closeModal();
        S.installTitle = 'Creando Servidor';
        renderInstallProgress('Creando Servidor');
        window.go.main.App.InstallServer(name, type, version);
      }}, 'Crear Servidor')
    )
  );
  openModal(m);
}

