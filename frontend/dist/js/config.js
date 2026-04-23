'use strict';

/* ══════════════════════════════════════════════════════
   PROPERTY METADATA (migrado desde views/properties_view.go)
══════════════════════════════════════════════════════ */
const PROP_DEFS = {
  'level-name':  { label: 'Nombre del Mundo',        type: 'text',     default: 'world',     category: 'Mundo' },
  'level-seed':  { label: 'Semilla del Mundo',        type: 'text',     default: '',          category: 'Mundo' },
  'level-type':  { label: 'Tipo de Mundo',            type: 'dropdown', options: ['default','flat','largeBiomes','amplified'], default: 'default', category: 'Mundo' },
  'generate-structures': { label: 'Generar Estructuras', type: 'bool', default: 'true', category: 'Mundo' },
  'gamemode':    { label: 'Modo de Juego',            type: 'dropdown', options: ['survival','creative','adventure','spectator'], default: 'survival', category: 'Juego' },
  'difficulty':  { label: 'Dificultad',               type: 'dropdown', options: ['peaceful','easy','normal','hard'], default: 'easy', category: 'Juego' },
  'hardcore':    { label: 'Modo Hardcore',            type: 'bool',     default: 'false',     category: 'Juego' },
  'pvp':         { label: 'PvP Activado',             type: 'bool',     default: 'true',      category: 'Juego' },
  'force-gamemode': { label: 'Forzar Modo de Juego', type: 'bool',     default: 'false',     category: 'Juego' },
  'spawn-monsters': { label: 'Generar Monstruos',    type: 'bool',     default: 'true',      category: 'Mobs' },
  'spawn-animals':  { label: 'Generar Animales',     type: 'bool',     default: 'true',      category: 'Mobs' },
  'spawn-npcs':     { label: 'Generar NPCs',         type: 'bool',     default: 'true',      category: 'Mobs' },
  'server-port': { label: 'Puerto del Servidor',     type: 'number',   default: '25565',     category: 'Servidor', min: 1, max: 65535 },
  'max-players': { label: 'Jugadores Máximos',       type: 'number',   default: '20',        category: 'Servidor', min: 1, max: 2147483647 },
  'motd':        { label: 'MOTD',                    type: 'text',     default: 'A Minecraft Server', category: 'Servidor' },
  'online-mode': { label: 'Modo Online',             type: 'bool',     default: 'true',      category: 'Servidor' },
  'white-list':  { label: 'Whitelist',               type: 'bool',     default: 'false',     category: 'Servidor' },
  'view-distance': { label: 'Distancia de Vista',    type: 'number',   default: '10',        category: 'Rendimiento', min: 2, max: 32 },
  'simulation-distance': { label: 'Distancia de Simulación', type: 'number', default: '10', category: 'Rendimiento', min: 3, max: 32 },
  'allow-flight':  { label: 'Permitir Vuelo',        type: 'bool',     default: 'false',     category: 'Otros' },
  'allow-nether':  { label: 'Permitir Nether',       type: 'bool',     default: 'true',      category: 'Otros' },
  'enable-rcon':   { label: 'Habilitar RCON',        type: 'bool',     default: 'false',     category: 'Otros' },
  'rcon.password': { label: 'Contraseña RCON',       type: 'text',     default: '',          category: 'Otros' },
  'rcon.port':     { label: 'Puerto RCON',           type: 'number',   default: '25575',     category: 'Otros', min: 1, max: 65535 },
};
const CATEGORY_ORDER = ['Mundo','Juego','Mobs','Servidor','Rendimiento','Otros','Desconocidas'];
const PRESETS = {
  Survival:  { gamemode:'survival',  difficulty:'normal',  pvp:'true',  'spawn-monsters':'true',  'spawn-animals':'true', hardcore:'false' },
  Creative:  { gamemode:'creative',  difficulty:'peaceful', pvp:'false', 'spawn-monsters':'false', 'spawn-animals':'true', hardcore:'false' },
  Hardcore:  { gamemode:'survival',  difficulty:'hard',    pvp:'true',  'spawn-monsters':'true',  'spawn-animals':'true', hardcore:'true' },
  Peaceful:  { gamemode:'survival',  difficulty:'peaceful', pvp:'false', 'spawn-monsters':'false', 'spawn-animals':'true', hardcore:'false' },
};
const TYPE_ICONS = { Vanilla:'⚡', PaperMC:'📄', Folia:'🌿', Forge:'⚒️', Fabric:'🧵' };
function typeIcon(t) { return TYPE_ICONS[t] || '🎮'; }

