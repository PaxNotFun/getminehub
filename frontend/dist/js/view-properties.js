/* ══════════════════════════════════════════════════════
   PROPERTIES VIEW
══════════════════════════════════════════════════════ */
async function renderPropertiesView(container) {
  container.innerHTML = '';
  const presetSel = el('select', { style:{ width:'160px' } });
  presetSel.appendChild(el('option', { value:'' }, 'Seleccionar preset'));
  Object.keys(PRESETS).forEach(p => presetSel.appendChild(el('option', { value:p }, p)));
  presetSel.onchange = () => { if (presetSel.value) applyPreset(presetSel.value); };

  const propsArea = el('div', { class: 'props-scroll', id: 'props-area' });
  const saveBtn   = el('button', { class: 'btn btn-primary btn-sm', onclick: saveProperties }, '💾 Guardar Cambios');
  const reloadBtn = el('button', { class: 'btn btn-ghost btn-sm', onclick: () => loadProperties() }, 'Recargar');

  container.appendChild(
    el('div', { class: 'view-container' },
      el('div', { class: 'view-header' },
        el('div', { class: 'view-header-title' }, '📝 Editor de server.properties'),
        el('div', { style:{ display:'flex', gap:'8px', alignItems:'center' } },
          el('span', { class: 'muted', style:{ fontSize:'0.78rem' } }, 'Preset:'),
          presetSel
        )
      ),
      propsArea,
      el('div', { class: 'props-footer' }, reloadBtn, saveBtn)
    )
  );
  await loadProperties();
}

async function loadProperties() {
  const area = document.getElementById('props-area');
  if (!area) return;
  area.innerHTML = '<div style="padding:24px;text-align:center;color:var(--text-muted)">Cargando propiedades...</div>';
  try {
    S.properties = await window.go.main.App.GetServerProperties();
    renderPropertiesForm(area);
  } catch (e) {
    area.innerHTML = '<div style="padding:24px;text-align:center;color:var(--danger)">⚠️ ' + String(e) + '</div>';
  }
}

function renderPropertiesForm(area) {
  area.innerHTML = '';
  const byCategory = {};
  S.properties.forEach(p => {
    const def = PROP_DEFS[p.key];
    const cat = def ? def.category : 'Desconocidas';
    if (!byCategory[cat]) byCategory[cat] = [];
    byCategory[cat].push(p);
  });

  CATEGORY_ORDER.forEach(cat => {
    const items = byCategory[cat];
    if (!items || !items.length) return;

    // Cuerpo de la categoría (visible por defecto)
    const body = el('div', { class: 'props-category-body' });
    items.forEach(p => {
      const def   = PROP_DEFS[p.key];
      const label = def ? def.label : p.key;
      const type  = def ? def.type : 'text';
      let widget;
      if (type === 'bool') {
        widget = el('input', { type:'checkbox', 'data-prop': p.key });
        widget.checked = p.value.toLowerCase() === 'true';
        widget.style.accentColor = 'var(--accent)';
      } else if (type === 'dropdown' && def && def.options) {
        widget = el('select', { 'data-prop': p.key });
        def.options.forEach(opt => widget.appendChild(el('option', { value:opt }, opt)));
        widget.value = p.value;
      } else if (type === 'number') {
        widget = el('input', { type:'number', 'data-prop': p.key, value: p.value,
          min: def ? String(def.min || '') : '',
          max: def ? String(def.max || '') : '' });
      } else {
        widget = el('input', { type:'text', 'data-prop': p.key, value: p.value });
      }
      body.appendChild(
        el('div', { class: 'props-row' },
          el('div', { class: 'props-key' }, label),
          el('div', { class: 'props-val' }, widget)
        )
      );
    });

    // Cabecera con toggle collapse
    const header = el('div', { class: 'props-category-header' },
      el('span', {}, cat),
      el('span', { class: 'props-chevron' }, '▾')
    );
    header.onclick = () => {
      const collapsed = body.style.display === 'none';
      body.style.display = collapsed ? '' : 'none';
      header.querySelector('.props-chevron').textContent = collapsed ? '▾' : '▸';
      header.style.borderRadius = collapsed
        ? 'var(--radius-lg) var(--radius-lg) 0 0'
        : 'var(--radius-lg)';
    };

    area.appendChild(
      el('div', { class: 'props-category' }, header, body)
    );
  });
}

function applyPreset(name) {
  const vals = PRESETS[name];
  if (!vals) return;
  for (const [key, val] of Object.entries(vals)) {
    const w = document.querySelector('[data-prop="' + key + '"]');
    if (!w) continue;
    if (w.type === 'checkbox') w.checked = val === 'true';
    else w.value = val;
  }
  showToast('Preset "' + name + '" aplicado');
}

async function saveProperties() {
  const updated = [];
  S.properties.forEach(p => {
    const w = document.querySelector('[data-prop="' + p.key + '"]');
    if (!w) { updated.push({ key: p.key, value: p.value }); return; }
    let value;
    if (w.type === 'checkbox') value = w.checked ? 'true' : 'false';
    else value = w.value;
    // BUG FIX: Go json tags son minúsculas → "key"/"value" (no "Key"/"Value")
    updated.push({ key: p.key, value: value });
  });
  try {
    await window.go.main.App.SaveServerProperties(updated);
    showToast('✅ server.properties guardado — reinicia el servidor para aplicar cambios');
  } catch (e) {
    await infoDialog('Error al guardar', String(e));
  }
}
