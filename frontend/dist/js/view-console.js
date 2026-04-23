/* ══════════════════════════════════════════════════════
   CONSOLE VIEW
══════════════════════════════════════════════════════ */
function renderConsoleView(container) {
  const srv = S.activeServer;
  const isRunning = S.isRunning;

  resetAnsiState(); // siempre limpiar estado de color al (re)construir la vista
  const output = el('pre', { class: 'console-output', id: 'console-output' });
  if (S.consoleText) output.appendChild(ansiToFragment(S.consoleText));
  const cmdInput = el('input', {
    type: 'text', placeholder: 'Escribe un comando...', id: 'cmd-input',
    disabled: !isRunning
  });
  cmdInput.onkeydown = e => {
    if (e.key === 'Enter') sendConsoleCommand();
    if (e.key === 'ArrowUp') {
      if (S.cmdHistIdx < S.cmdHistory.length - 1) {
        S.cmdHistIdx++;
        cmdInput.value = S.cmdHistory[S.cmdHistory.length - 1 - S.cmdHistIdx] || '';
      }
    }
    if (e.key === 'ArrowDown') {
      if (S.cmdHistIdx > 0) {
        S.cmdHistIdx--;
        cmdInput.value = S.cmdHistory[S.cmdHistory.length - 1 - S.cmdHistIdx] || '';
      } else { S.cmdHistIdx = -1; cmdInput.value = ''; }
    }
  };

  const btnStart   = el('button', { class: 'btn btn-success btn-sm', id: 'btn-start',   onclick: () => startServer() },   '▶ Encender');
  const btnStop    = el('button', { class: 'btn btn-danger  btn-sm', id: 'btn-stop',    onclick: () => stopServer()  },   '■ Apagar');
  const btnRestart = el('button', { class: 'btn btn-warning btn-sm', id: 'btn-restart', onclick: () => restartServer() }, '↺ Reiniciar');
  if (isRunning) { btnStart.disabled = true; }
  else { btnStop.disabled = true; btnRestart.disabled = true; }

  const statusDot = el('span', { class: 'status-dot' + (isRunning ? ' running' : '') , id: 'status-dot' });
  const statusTxt = el('span', { id: 'status-txt' }, isRunning ? 'En línea' : 'Detenido');
  const status = el('div', { class: 'server-status' }, statusDot, statusTxt);

  container.innerHTML = '';
  container.appendChild(
    el('div', { class: 'view-container' },
      el('div', { class: 'view-header' },
        el('div', {},
          el('div', { class: 'view-header-title' }, srv ? srv.Name : 'Consola'),
          el('div', { class: 'view-header-meta' }, srv ? (srv.Type + ' · MC ' + srv.Version) : '')
        ),
        el('button', { class: 'btn btn-ghost btn-sm', onclick: () => { S.consoleText = ''; resetAnsiState(); const o = $('console-output'); if (o) o.textContent = ''; } }, '🗑 Limpiar')
      ),
      el('div', { class: 'console-controls' }, btnStart, btnStop, btnRestart, status),
      output,
      el('div', { class: 'console-input-bar' },
        cmdInput,
        el('button', { class: 'btn btn-secondary btn-sm', onclick: sendConsoleCommand }, 'Enviar')
      )
    )
  );
  scrollConsoleToBottom();
  applyConsoleStyles();
}

function scrollConsoleToBottom() {
  const o = $('console-output');
  if (!o) return;
  o.scrollTop = o.scrollHeight;
}

function applyConsoleStyles() {
  const o = $('console-output');
  if (!o) return;
  o.style.background    = '#0f0f18';
  o.style.lineHeight    = '1.45';
  o.style.fontSize      = '13.5px';
  o.style.padding       = '16px';
  o.scrollTop           = o.scrollHeight;
}

async function startServer() {
  try { await window.go.main.App.StartServer(); }
  catch (e) { sidebarToast('⚠️ ' + String(e)); }
}
async function stopServer() {
  try { await window.go.main.App.StopServer(); }
  catch (e) { sidebarToast('⚠️ ' + String(e)); }
}
async function restartServer() {
  try { await window.go.main.App.RestartServer(); }
  catch (e) { sidebarToast('⚠️ ' + String(e)); }
}

async function sendConsoleCommand() {
  const inp = $('cmd-input');
  if (!inp) return;
  const cmd = inp.value.trim();
  if (!cmd) return;
  S.cmdHistory.push(cmd);
  S.cmdHistIdx = -1;
  inp.value = '';
  try { await window.go.main.App.SendCommand(cmd); }
  catch (e) { appendToConsole('\n❌ Error: ' + String(e) + '\n'); }
}

/* ── Parser ANSI → spans coloreados ──────────────────────────────────────── */
const ANSI_COLORS = {
  '30':'#586e75','31':'#ff6b6b','32':'#1dd1a1','33':'#e2e8f0',
  '34':'#54a0ff','35':'#ff9ff3','36':'#48dbfb','37':'#f1f2f6',
  '90':'#888888','91':'#ff9999','92':'#55efc4','93':'#ffeaa7',
  '94':'#74b9ff','95':'#fd79a8','96':'#81ecec','97':'#dfe6e9',
};

// Estado de color persistente entre llamadas — el servidor manda línea por línea
// y un color abierto en una línea debe seguir en la siguiente hasta recibir reset.
let _ansiColor = null;

// Convierte texto con códigos ANSI en un DocumentFragment con spans coloreados.
// Mantiene el color activo entre llamadas para manejar códigos que cruzan líneas.
function ansiToFragment(text) {
  const frag = document.createDocumentFragment();
  const re = /\x1b\[([\d;]*)m/g;
  let last = 0;
  const push = str => {
    if (!str) return;
    if (_ansiColor) {
      const span = document.createElement('span');
      span.style.color = _ansiColor;
      span.appendChild(document.createTextNode(str));
      frag.appendChild(span);
    } else {
      frag.appendChild(document.createTextNode(str));
    }
  };
  let m;
  while ((m = re.exec(text)) !== null) {
    push(text.slice(last, m.index));
    last = m.index + m[0].length;
    const codes = m[1].split(';').filter(Boolean);
    if (codes.length === 0 || codes[0] === '0') {
      _ansiColor = null;
    } else {
      for (const code of codes) {
        if (ANSI_COLORS[code]) { _ansiColor = ANSI_COLORS[code]; break; }
      }
    }
  }
  push(text.slice(last));
  return frag;
}

// Llamar esto al limpiar la consola para resetear el estado de color
function resetAnsiState() { _ansiColor = null; }

function appendToConsole(text) {
  S.consoleText += text;
  const o = $('console-output');
  if (!o) return;
  o.appendChild(ansiToFragment(text));
  while (o.childNodes.length > 2000) o.removeChild(o.firstChild);
  scrollConsoleToBottom();
}

function updateServerStatusUI(running) {
  S.isRunning = running;
  const dot = $('status-dot');
  const txt = $('status-txt');
  const btnStart   = $('btn-start');
  const btnStop    = $('btn-stop');
  const btnRestart = $('btn-restart');
  const cmdInput   = $('cmd-input');
  const menuBtn    = $('btn-main-menu');

  if (dot) { dot.className = 'status-dot' + (running ? ' running' : ''); }
  if (txt) { txt.textContent = running ? 'En línea' : 'Detenido'; }
  if (btnStart)   { btnStart.disabled   = running; }
  if (btnStop)    { btnStop.disabled    = !running; }
  if (btnRestart) { btnRestart.disabled = !running; }
  if (cmdInput)   { cmdInput.disabled   = !running; }
  if (menuBtn)    { menuBtn.disabled    = running; }
}

