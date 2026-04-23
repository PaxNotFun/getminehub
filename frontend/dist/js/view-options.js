/* ══════════════════════════════════════════════════════
   OPTIONS VIEW
══════════════════════════════════════════════════════ */
async function renderOptionsView(container) {
  container.innerHTML = '';
  const [jvmCfg, installedJavas] = await Promise.all([
    window.go.main.App.GetJVMConfig(),
    window.go.main.App.GetInstalledJavas(),
  ]);
  const srv = S.activeServer;

  const minRAM  = el('input', { type:'text', value: jvmCfg.minRAM || '1G', placeholder:'Ej: 2G' });
  const maxRAM  = el('input', { type:'text', value: jvmCfg.maxRAM || '2G', placeholder:'Ej: 4G' });
  const jvmArgs = el('input', { type:'text', value: jvmCfg.jvmArgs || '', placeholder:'-XX:+UseG1GC' });
  const aikar   = el('input', { type:'checkbox' }); aikar.checked = jvmCfg.useAikar;

  // ── Java selector ─────────────────────────────────────────────────────────
  const javaSelect = el('select', { class: 'form-select' });
  javaSelect.appendChild(el('option', { value: '' }, '⚡ Java del sistema (por defecto)'));
  if (installedJavas && installedJavas.length) {
    installedJavas.forEach(j => {
      const opt = el('option', { value: j.path }, `☕ ${j.name} — ${j.path}`);
      if (j.path === jvmCfg.javaExe) opt.selected = true;
      javaSelect.appendChild(opt);
    });
  }

  const saveBtn = el('button', { class: 'btn btn-primary btn-sm btn-full', onclick: async () => {
    try {
      await window.go.main.App.SaveJVMConfig(minRAM.value, maxRAM.value, jvmArgs.value, aikar.checked);
      if (javaSelect.value !== (jvmCfg.javaExe || '')) {
        await window.go.main.App.SaveServerJava(javaSelect.value);
      }
      saveBtn.textContent = '✅ ¡Guardado!';
      setTimeout(() => saveBtn.textContent = '💾 Guardar Configuración', 2000);
    } catch (e) { showToast('Error: ' + String(e)); }
  }}, '💾 Guardar Configuración');

  // ── Update section ─────────────────────────────────────────────────────────
  const updateDropdown = el('select', {});
  const updateBtn      = el('button', { class: 'btn btn-primary btn-sm options-update-btn', disabled: true, onclick: () => confirmUpdateVersion() }, 'Actualizar');
  updateDropdown.appendChild(el('option', {}, 'Cargando versiones...'));
  updateDropdown.disabled = true;

  if (srv) {
    window.go.main.App.GetVersions(srv.Type).then(async all => {
      const newer = await window.go.main.App.FilterNewerVersions(all || [], srv.Version);
      updateDropdown.innerHTML = '';
      if (newer && newer.length) {
        newer.forEach(v => updateDropdown.appendChild(el('option', { value:v }, v)));
        updateDropdown.disabled = false;
        updateBtn.disabled = false;
      } else {
        updateDropdown.appendChild(el('option', {}, 'No hay versiones superiores'));
      }
    }).catch(() => {
      updateDropdown.innerHTML = '';
      updateDropdown.appendChild(el('option', {}, 'Error al cargar'));
    });
  }

  async function confirmUpdateVersion() {
    const target = updateDropdown.value;
    if (!target) return;
    if (S.isRunning) { await infoDialog('Servidor activo', 'Detén el servidor antes de actualizarlo.'); return; }
    const ok = await confirmDialog('Confirmar Actualización',
      '¿Actualizar a MC ' + target + '?\n\nSe conservarán mundos, plugins y configuraciones.');
    if (!ok) return;
    S.installTitle = 'Actualizando Servidor';
    renderInstallProgress('Actualizando Servidor');
    window.go.main.App.UpdateServer(target);
  }

  // ── Reinstall section ──────────────────────────────────────────────────────
  const reinstallMode = el('select', {});
  ['Parcial — Conservar mundos, plugins y configuraciones',
   'Completa — Eliminar TODO excepto configuración de RAM/JVM'].forEach((v,i) =>
    reinstallMode.appendChild(el('option', { value: i === 0 ? 'partial' : 'total' }, v)));

  const reinstallBtn = el('button', { class: 'btn btn-warning btn-full', onclick: async () => {
    if (S.isRunning) { await infoDialog('Servidor activo', 'Detén el servidor antes de reinstalarlo.'); return; }
    const mode = reinstallMode.value;
    const msg  = mode === 'partial'
      ? '¿Reinstalar conservando mundos, plugins y configuraciones?'
      : '¿REINSTALACIÓN COMPLETA?\n\nSE ELIMINARÁ TODO LO DEMÁS.\n\nESTA ACCIÓN NO SE PUEDE DESHACER.';
    const ok   = await confirmDialog('Confirmar Reinstalación', msg, mode === 'total');
    if (!ok) return;
    S.installTitle = 'Reinstalando Servidor';
    renderInstallProgress('Reinstalando Servidor');
    window.go.main.App.ReinstallServer(mode);
  }}, '🔄 Reinstalar Servidor');

  // ── Delete section ─────────────────────────────────────────────────────────
  const deleteMode = el('select', {});
  ['Solo quitar de la lista — Mantener archivos en disco',
   'Eliminar completamente — Borrar TODOS los archivos'].forEach((v,i) =>
    deleteMode.appendChild(el('option', { value: i === 0 ? 'list-only' : 'full' }, v)));

  const deleteBtn = el('button', { class: 'btn btn-danger btn-full', onclick: async () => {
    if (S.isRunning) { await infoDialog('Servidor activo', 'Detén el servidor antes de eliminarlo.'); return; }
    const deleteFiles = deleteMode.value === 'full';
    const sName = srv ? srv.Name : 'este servidor';
    const msg   = deleteFiles
      ? '¿ELIMINAR COMPLETAMENTE "' + sName + '"?\n\nBorrará PERMANENTEMENTE todos los archivos.\n\nESTA ACCIÓN NO SE PUEDE DESHACER.'
      : '¿Quitar "' + sName + '" de GetMineHub?\n\nLos archivos en disco se mantendrán intactos.';
    const ok = await confirmDialog('Confirmar Eliminación', msg, deleteFiles);
    if (!ok) return;
    const result = await window.go.main.App.DeleteServer(deleteFiles);
    if (result.success) {
      await infoDialog('Eliminado', result.message);
      window.go.main.App.CloseServer();
      S.activeServer = null; S.isRunning = false;
      renderMainMenu();
    } else {
      await infoDialog('Error', result.message);
    }
  }}, '🗑️ Eliminar Servidor');

  container.appendChild(
    el('div', { class: 'view-container' },
      el('div', { class: 'view-header' },
        el('div', { class: 'view-header-title' }, '⚙️ Opciones: ' + (srv ? srv.Name : ''))
      ),
      el('div', { class: 'options-scroll' },

        /* Files */
        el('div', { class: 'options-section' },
          el('div', { class: 'options-section-header' }, '📂 Gestión de Archivos'),
          el('div', { class: 'options-section-body' },
            el('button', {
              class: 'btn btn-secondary btn-full',
              onclick: () => window.go.main.App.OpenServerFolder()
            }, '📁 Abrir Carpeta del Servidor')
          )
        ),

        /* Performance */
        el('div', { class: 'options-section' },
          el('div', { class: 'options-section-header' }, '⚡ Rendimiento y Argumentos Java'),
          el('div', { class: 'options-section-body' },
            el('div', { class: 'options-ram-row' },
              el('div', { class: 'form-group' },
                el('label', { class: 'form-label' }, 'RAM MÍNIMA'),
                minRAM
              ),
              el('div', { class: 'form-group' },
                el('label', { class: 'form-label' }, 'RAM MÁXIMA'),
                maxRAM
              )
            ),
            el('div', { class: 'form-group' },
              el('label', { class: 'form-label' }, 'ARGUMENTOS JVM ADICIONALES'),
              jvmArgs
            ),
            el('label', { class: 'check-wrap' },
              aikar,
              el('span', {}, "Aikar's Flags (Optimización G1GC recomendada)")
            ),
            el('div', { class: 'form-group' },
              el('label', { class: 'form-label' }, '☕ VERSIÓN DE JAVA PARA ESTE SERVIDOR'),
              el('div', { class: 'java-select-row' },
                javaSelect,
                el('button', {
                  class: 'btn btn-ghost btn-sm',
                  onclick: () => renderJavaView(),
                  title: 'Gestionar Java'
                }, '⚙️')
              ),
              el('p', { class: 'muted', style: { fontSize: '0.78rem', marginTop: '4px' } },
                'Puedes instalar más versiones en la sección ☕ Gestión de Java del menú principal.')
            ),
            saveBtn
          )
        ),

        /* Update */
        el('div', { class: 'options-section' },
          el('div', { class: 'options-section-header' }, '⬆️ Actualizar Servidor'),
          el('div', { class: 'options-section-body' },
            el('p', { class: 'muted', style:{ fontSize:'0.82rem' } }, 'Solo se muestran versiones superiores a la actual.'),
            el('div', { class: 'options-update-row' },
              el('div', { class: 'options-update-select' }, updateDropdown),
              updateBtn
            )
          )
        ),

        /* Reinstall */
        el('div', { class: 'options-section' },
          el('div', { class: 'options-section-header' }, '🔄 Reinstalación del Servidor'),
          el('div', { class: 'options-section-body' },
            el('div', { class: 'form-group' },
              el('label', { class: 'form-label' }, 'MODO DE REINSTALACIÓN'),
              reinstallMode
            ),
            reinstallBtn
          )
        ),

        /* Danger Zone */
        el('div', { class: 'options-section danger-section' },
          el('div', { class: 'options-section-header' }, '⚠️ Zona Peligrosa'),
          el('div', { class: 'options-section-body' },
            el('div', { class: 'form-group' },
              el('label', { class: 'form-label' }, 'MODO DE ELIMINACIÓN'),
              deleteMode
            ),
            deleteBtn
          )
        )

      ) // end scroll
    ) // end view-container
  );
}
