/* ══════════════════════════════════════════════════════
   STATE — Fuente única de verdad para el estado de la UI
   Todas las vistas deben leer/escribir aquí en lugar de
   mantener variables globales dispersas.
══════════════════════════════════════════════════════ */
const S = {
  // Servidor activo y su estado de proceso
  activeServer: null,
  isRunning: false,

  // Navegación
  view: 'main-menu',   // 'main-menu' | 'dashboard' | 'installing' | 'java'
  dashView: 'console', // 'console' | 'properties' | 'options'

  // Consola
  consoleText: '',
  cmdHistory: [],
  cmdHistIdx: -1,

  // server.properties
  properties: [],      // [{key, value}]

  // Instalación
  installTitle: 'Instalando...',
  menuBlocked: false,

  // Java (estado de descarga en curso)
  java: {
    downloading: false,
    downloadingVersion: 0,
  },
};

/* ── Helpers de mutación ─────────────────────────────────
   Usar estas funciones en lugar de mutar S directamente
   hace que sea fácil agregar logging/reactividad después.
─────────────────────────────────────────────────────── */

/** Resetea todo el estado relacionado con un servidor abierto. */
function resetServerState() {
  S.activeServer = null;
  S.isRunning    = false;
  S.consoleText  = '';
  S.cmdHistory   = [];
  S.cmdHistIdx   = -1;
  S.properties   = [];
  S.dashView     = 'console';
}

/** Marca el inicio de una descarga de Java. */
function setJavaDownloading(javaVersion) {
  S.java.downloading       = true;
  S.java.downloadingVersion = javaVersion;
}

/** Limpia el estado de descarga de Java (éxito o error). */
function clearJavaDownloading() {
  S.java.downloading        = false;
  S.java.downloadingVersion = 0;
}
