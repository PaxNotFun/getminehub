/* ══════════════════════════════════════════════════════
   UTILITIES
══════════════════════════════════════════════════════ */
const $  = id => document.getElementById(id);
const $$ = sel => document.querySelectorAll(sel);
function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag);
  // Atributos booleanos que deben manejarse con propiedades, no con setAttribute.
  // setAttribute('disabled', false) NO deshabilita — hay que usar e.disabled = false.
  const BOOL_ATTRS = new Set(['disabled', 'checked', 'selected', 'readonly', 'multiple', 'autofocus']);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'class') e.className = v;
    else if (k === 'style') Object.assign(e.style, v);
    else if (k.startsWith('on')) e.addEventListener(k.slice(2), v);
    else if (BOOL_ATTRS.has(k)) e[k] = !!v;   // true activa, false desactiva correctamente
    else e.setAttribute(k, v);
  }
  for (const c of children) {
    if (typeof c === 'string') e.appendChild(document.createTextNode(c));
    else if (c instanceof Node) e.appendChild(c);
  }
  return e;
}
function html(str) { const d = document.createElement('div'); d.innerHTML = str; return d.firstChild; }

let toastTimer = null;
function showToast(msg) {
  const t = $('toast');
  t.textContent = msg;
  t.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove('show'), 3000);
}

function fmtSize(gb) {
  if (gb < 0.01) return '< 0.01 GB';
  return gb.toFixed(2) + ' GB';
}

