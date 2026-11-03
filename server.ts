import express, { Request, Response } from 'express';
import cors from 'cors';
import path from 'path';
import { fileURLToPath } from 'url';
import { GoogleGenAI } from '@google/genai';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : 3000;
const HOST = '0.0.0.0';

app.use(cors());
app.use(express.json());

// In-Memory Sovereign Bot State (replaces all Netflix/Lux lore with official prestige Lucifer Protocol)
const botState = {
  version: '2.0.0-sovereign',
  commit: 'revised-protocol',
  startedAt: new Date(),
  guild: {
    id: '1289371089201987654',
    name: 'Lucifer Protocol • Royal & Sovereign Headquarters',
    owner: 'Lucifer • Soberano Canciller',
    memberCount: 2840,
    onlineCount: 915,
    channelsCount: 48,
    rolesCount: 32,
    welcomeChannel: '#bienvenida-oficial',
    welcomeMessage: 'Sea bienvenido al senado de honor. Aquí la distinción, la palabra y la excelencia forjan el destino de cada ciudadano.',
    welcomeImageUrl: 'https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1200&q=80',
    language: 'es',
    responsesPublic: false,
    notificationsEnabled: true,
  },
  presence: {
    status: 'online' as 'online' | 'idle' | 'dnd',
    currentActivityIndex: 0,
    intervalSeconds: 10,
    lastRotatedAt: new Date().toISOString(),
    activities: [
      {
        type: 'WATCHING' as const,
        typeLabel: 'Vigilando',
        text: '🔱 Custodiando los fueros de alta soberanía',
        badge: '🔱 Soberanía',
        details: 'Supervisando 2,840 ciudadanos en el senado imperial',
        state: 'Protocolo de Honor • Despacho Soberano',
      },
      {
        type: 'LISTENING' as const,
        typeLabel: 'Escuchando',
        text: '⚖️ Arbitrando el protocolo del senado imperial',
        badge: '⚖️ Cancillería',
        details: 'Deliberando directivas solemnes de cancillería',
        state: 'Tribunal en Sesión Solemne',
      },
      {
        type: 'CUSTOM' as const,
        typeLabel: 'En Estado',
        text: '🍷 Degustando una gran reserva en cancillería',
        badge: '🍷 Distinción',
        details: 'Recepción diplomática privada con dignatarios',
        state: 'Reserva Imperial de la Corona',
      },
      {
        type: 'WATCHING' as const,
        typeLabel: 'Atendiendo',
        text: '👑 Concediendo audiencias diplomáticas • /ask',
        badge: '👑 Audiencias',
        details: 'Respuestas de alta elocuencia sin intermediarios',
        state: 'Despacho Real Abierto • Respuestas en Texto Puro',
      },
      {
        type: 'COMPETING' as const,
        typeLabel: 'Compitiendo en',
        text: '💎 Presidiendo el salón de alta distinción • /ruleta',
        badge: '💎 Ruleta Real',
        details: 'Mesa de ruleta francesa • Verde 36x, Rojo/Negro 2x',
        state: 'Banca Soberana en Circulación',
      },
      {
        type: 'WATCHING' as const,
        typeLabel: 'Inspeccionando',
        text: '⚜️ Verificando las bóvedas del tesoro • /saldo',
        badge: '⚜️ Tesorería',
        details: '2.45M de monedas soberanas custodiadas en bóveda',
        state: 'Auditoría Financiera en Curso',
      },
      {
        type: 'LISTENING' as const,
        typeLabel: 'Acreditando',
        text: '🥂 Decretando estipendios a los dignatarios • /fichas',
        badge: '🥂 Prosperidad',
        details: '250 monedas cada 24h para cada miembro acreditado',
        state: 'Asignaciones Diarias Activas',
      },
      {
        type: 'WATCHING' as const,
        typeLabel: 'Custodiando',
        text: '📜 Sellando decretos oficiales de honor y diplomacia',
        badge: '📜 Decretos',
        details: 'Registro inmutable de acuerdos y pactos solemnes',
        state: 'Sello Imperial Vigente',
      },
      {
        type: 'WATCHING' as const,
        typeLabel: 'Defendiendo',
        text: '⚔️ Salvaguardando el perímetro protocolar • /seguridad',
        badge: '⚔️ Perímetro',
        details: 'Escudo antiraid patrullando canales en tiempo real',
        state: 'Amenaza Nula • Perímetro Seguro',
      },
    ],
  },
  antiraid: {
    enabled: true,
    lockdownActive: false,
    lockdownUntil: null as string | null,
    lockdownChannels: [] as string[],
    detectors: {
      joinFlood: { name: 'Filtro Join Flood', limit: 5, windowSec: 10, enabled: true, triggeredCount: 19 },
      messageSpam: { name: 'Supresión de Spam Masivo', limit: 8, windowSec: 10, enabled: true, triggeredCount: 42 },
      mentionSpam: { name: 'Filtro Menciones en Ráfaga', limit: 5, windowSec: 10, enabled: true, triggeredCount: 11 },
      linkScan: { name: 'Inspección de Enlaces Maliciosos', limit: 3, windowSec: 10, enabled: true, triggeredCount: 28 },
      accountAge: { name: 'Cuarentena Cuentas Recientes (<3d)', minDays: 3, enabled: true, triggeredCount: 63 },
      nukeProtection: { name: 'Defensa Anti-Demolición (Nuke)', limit: 3, windowSec: 30, enabled: true, triggeredCount: 3 },
    },
    liveMetrics: {
      joinsLastTenSeconds: 1,
      channelDeletesLastMin: 0,
      roleDeletesLastMin: 0,
      messagesPerSec: 18,
      threatLevel: 'Seguro • 10/100 🛡️',
    },
    incidents: [
      {
        id: 'inc-9140',
        type: 'Ataque Coordinado de Cuentas Nuevas',
        target: '#lobby-general',
        action: 'Cuarentena Perimetral Inmediata',
        mitigatedAt: new Date(Date.now() - 1000 * 60 * 15).toISOString(),
        details: '19 cuentas automatizadas creadas hace < 5 minutos interceptadas y expulsadas.',
        severity: 'Alta',
      },
      {
        id: 'inc-9132',
        type: 'Ráfaga de Menciones No Autorizadas',
        target: '#senado-comunidad',
        action: 'Silencio Protocolar 2h',
        mitigatedAt: new Date(Date.now() - 1000 * 60 * 70).toISOString(),
        details: 'Usuario @RaidBot silenciado tras emitir 9 menciones masivas en 1 segundo.',
        severity: 'Media',
      },
      {
        id: 'inc-9128',
        type: 'Inyección de Enlace Phishing',
        target: '#anuncios-oficiales',
        action: 'Mensaje purgado & Alerta a Cancilleres',
        mitigatedAt: new Date(Date.now() - 1000 * 60 * 210).toISOString(),
        details: 'Enlace apócrifo de verificación interceptado preventivamente.',
        severity: 'Crítica',
      },
    ],
  },
  economy: {
    totalChipsInCirculation: 2450000,
    casinoBank: 680000,
    shopItems: [
      { id: 'item-1', key: 'sovereign_pass', name: 'Pase Gran Reserva Soberana', description: 'Acceso a los estrados superiores y salas de consejo privadas.', price: 500, stock: 'Ilimitado', icon: '⚜️' },
      { id: 'item-2', key: 'imperial_seal', name: 'Sello Imperial de Distinción', description: 'Emblema protocolar que otorga +60 XP protocolar y distinción visual.', price: 200, stock: '85 disponibles', icon: '👑' },
      { id: 'item-3', key: 'sovereign_sheet', name: "Partitura 'La Marcha Soberana'", description: 'Pieza de concierto noble compuesta al piano por Lucifer para su expediente.', price: 850, stock: '18 disponibles', icon: '🎹' },
      { id: 'item-4', key: 'diplomatic_contract', name: 'Contrato Diplomático de Inmunidad', description: 'Protección ante robos casuales y bonificación de 10% en ruleta de salón.', price: 2000, stock: '6 disponibles', icon: '📜' },
      { id: 'item-5', key: 'scarlet_badge', name: 'Insignia de Oro & Terciopelo Escarlata', description: 'Condecoración oficial de la corona para insignias de perfil.', price: 1400, stock: '12 disponibles', icon: '💎' },
    ],
    members: [
      { id: 'user-001', username: 'Legate_Corvin', balance: 5400, level: 16, rank: '💎 Gran Soberano', xp: 80, lastDaily: new Date(Date.now() - 1000 * 60 * 60 * 26).toISOString(), inventory: ['Pase Gran Reserva Soberana', 'Sello Imperial de Distinción'] },
      { id: 'user-002', username: 'Chancellor_Valerius', balance: 22400, level: 26, rank: '👑 Dignatario Imperial', xp: 95, lastDaily: new Date().toISOString(), inventory: ['Contrato Diplomático de Inmunidad', 'Insignia de Oro & Terciopelo Escarlata'] },
      { id: 'user-003', username: 'Ambassador_Elena', balance: 11200, level: 21, rank: '👑 Dignatario Imperial', xp: 55, lastDaily: new Date().toISOString(), inventory: ['Pase Gran Reserva Soberana'] },
      { id: 'user-004', username: 'Consul_Matthias', balance: 3900, level: 8, rank: '⚜️ Miembro Distinguido', xp: 45, lastDaily: new Date().toISOString(), inventory: ["Partitura 'La Marcha Soberana'"] },
      { id: 'user-005', username: 'Citizen_Aurelius', balance: 750, level: 3, rank: '✨ Ciudadano de Honor', xp: 30, lastDaily: new Date(Date.now() - 1000 * 60 * 60 * 32).toISOString(), inventory: [] },
    ],
  },
  moderation: {
    bans: [
      { userId: '891028391823901923', username: 'incursor_malicioso', reason: 'Intento de asalto coordinado y suplantación de identidad', bannedAt: '2026-10-30T16:20:00Z', moderator: 'Chancellor_Valerius' },
      { userId: '782910394819283741', username: 'infraccion_protocolar', reason: 'Falta reiterada al respeto y quebranto de los fueros oficiales', bannedAt: '2026-10-18T11:45:00Z', moderator: 'Lucifer' },
    ],
    timeouts: [
      { userId: 'user-005', username: 'Citizen_Aurelius', reason: 'Exceso de mensajes reiterados fuera de horario', until: new Date(Date.now() + 1000 * 60 * 50).toISOString(), moderator: 'Ambassador_Elena' },
    ],
    appeals: [
      { id: 201, targetId: '891028391823901923', username: 'incursor_malicioso', sanction: 'ban', reason: 'Señor Canciller: mi cuenta sufrió una vulneración externa que ya remedié con 2FA. Solicito clemencia protocolar.', status: 'pending', createdAt: '2026-11-01T20:10:00Z' },
      { id: 200, targetId: 'user-005', username: 'Citizen_Aurelius', sanction: 'timeout', reason: 'Presento mis disculpas a la cancillería, fue un malentendido en el senado.', status: 'pending', createdAt: '2026-11-02T11:20:00Z' },
      { id: 199, targetId: '671928374829182736', username: 'solicitante_honor', sanction: 'kick', reason: 'He repasado los estatutos oficiales y me comprometo a cumplir estrictamente la normativa.', status: 'accepted', createdAt: '2026-10-25T14:30:00Z' },
    ],
    auditLogs: [
      { id: 'log-204', actor: 'Lucifer • Soberano', action: 'PROTOCOL_PARAM_UPDATE', details: 'Ajuste de sensibilidad en radar JoinFlood (5 miembros en 10s)', timestamp: new Date(Date.now() - 1000 * 60 * 8).toISOString() },
      { id: 'log-203', actor: 'Chancellor_Valerius', action: 'TIMEOUT_ENFORCED', details: 'Silenciado Citizen_Aurelius por 50 minutos (Conducta desordenada)', timestamp: new Date(Date.now() - 1000 * 60 * 40).toISOString() },
      { id: 'log-202', actor: 'Ambassador_Elena', action: 'CATALOG_AMENDMENT', details: 'Actualizada existencia del Sello Imperial de Distinción (200 monedas)', timestamp: new Date(Date.now() - 1000 * 60 * 110).toISOString() },
      { id: 'log-201', actor: 'Escudo Automatizado', action: 'LOCKDOWN_CONCLUDED', details: 'Restablecida la circulación en los canales tras neutralizar incursión', timestamp: new Date(Date.now() - 1000 * 60 * 310).toISOString() },
    ],
  },
  aiHistory: [
    {
      id: 'ai-01',
      sender: 'Legate_Corvin',
      prompt: '¿Cuál es el valor primordial de la lealtad en los círculos de poder?',
      response: 'La lealtad, mi estimado legado, es la divisa más codiciada y escasa del mundo. Quien la compra con lisonjas la pierde con la primera adversidad; quien la conquista con respeto y coherencia soberana, gobierna sin necesidad de alzar la voz.',
      command: '/ask',
      timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
    {
      id: 'ai-02',
      sender: 'Citizen_Aurelius',
      prompt: 'Un brindis de honor para la próxima deliberación.',
      response: 'Brindo por aquellos que entran a las salas solemnes con paso firme, aun cuando el pulso les tiemble. La grandeza no exige perfección, sino el porte necesario para sostener la mirada ante el destino.',
      command: '/roast',
      timestamp: new Date(Date.now() - 1000 * 60 * 20).toISOString(),
    },
  ],
};

// ==========================================
// HEALTH & READY
// ==========================================

app.get(['/health', '/api/health'], (_req: Request, res: Response) => {
  res.json({
    status: 'ok',
    version: botState.version,
    commit: botState.commit,
    timestamp: new Date().toISOString(),
  });
});

app.get(['/ready', '/api/ready'], (_req: Request, res: Response) => {
  res.json({
    status: 'ok',
    checks: {
      postgres: 'ok (repositorio en memoria activo)',
      redis: 'ok (cache protocolar activo)',
      discord: 'ok (gateway protocolar soberano en línea)',
      antiraid: botState.antiraid.enabled ? 'ok (escudo perimetral activo)' : 'degraded (escudo pausado)',
      ai_engine: process.env.GEMINI_API_KEY ? 'ok (oráculo gemini activo)' : 'ok (oráculo soberano local activo)',
    },
    uptimeSeconds: Math.floor((Date.now() - botState.startedAt.getTime()) / 1000),
    timestamp: new Date().toISOString(),
  });
});

function getCurrentActivity() {
  const activities = botState.presence.activities;
  return activities[botState.presence.currentActivityIndex] || activities[0];
}

function advancePresenceRotation(targetIndex?: number) {
  const activities = botState.presence.activities;
  if (typeof targetIndex === 'number' && targetIndex >= 0 && targetIndex < activities.length) {
    botState.presence.currentActivityIndex = targetIndex;
  } else {
    botState.presence.currentActivityIndex = (botState.presence.currentActivityIndex + 1) % activities.length;
  }
  botState.presence.lastRotatedAt = new Date().toISOString();
  return getCurrentActivity();
}

// Background auto-rotation interval (updates periodically to keep presence fresh and engaging)
setInterval(() => {
  advancePresenceRotation();
}, 10000);

// ==========================================
// PRESENCE & STATUS ROTATION API
// ==========================================

app.get('/api/presence', (_req: Request, res: Response) => {
  res.json({
    status: botState.presence.status,
    currentActivity: getCurrentActivity(),
    currentActivityIndex: botState.presence.currentActivityIndex,
    activities: botState.presence.activities,
    intervalSeconds: botState.presence.intervalSeconds,
    lastRotatedAt: botState.presence.lastRotatedAt,
  });
});

app.post('/api/presence/next', (req: Request, res: Response) => {
  const { index } = req.body || {};
  const newActivity = advancePresenceRotation(typeof index === 'number' ? index : undefined);
  res.json({
    status: botState.presence.status,
    currentActivity: newActivity,
    currentActivityIndex: botState.presence.currentActivityIndex,
    activities: botState.presence.activities,
    intervalSeconds: botState.presence.intervalSeconds,
    lastRotatedAt: botState.presence.lastRotatedAt,
  });
});

// ==========================================
// STATS & GUILD OVERVIEW
// ==========================================

app.get('/api/stats', (_req: Request, res: Response) => {
  res.json({
    guild: botState.guild,
    presence: {
      status: botState.presence.status,
      currentActivity: getCurrentActivity(),
      currentActivityIndex: botState.presence.currentActivityIndex,
      activities: botState.presence.activities,
      intervalSeconds: botState.presence.intervalSeconds,
      lastRotatedAt: botState.presence.lastRotatedAt,
    },
    antiraidSummary: {
      enabled: botState.antiraid.enabled,
      lockdownActive: botState.antiraid.lockdownActive,
      incidentsCount: botState.antiraid.incidents.length,
      threatLevel: botState.antiraid.liveMetrics.threatLevel,
    },
    economySummary: {
      totalChips: botState.economy.totalChipsInCirculation,
      casinoBank: botState.economy.casinoBank,
      shopItemsCount: botState.economy.shopItems.length,
      topMembers: botState.economy.members.slice(0, 3),
    },
    moderationSummary: {
      activeBans: botState.moderation.bans.length,
      activeTimeouts: botState.moderation.timeouts.length,
      pendingAppeals: botState.moderation.appeals.filter((a) => a.status === 'pending').length,
    },
    uptime: Math.floor((Date.now() - botState.startedAt.getTime()) / 1000),
  });
});

// ==========================================
// ANTI-RAID & LOCKDOWN API
// ==========================================

app.get('/api/antiraid', (_req: Request, res: Response) => {
  res.json(botState.antiraid);
});

app.post('/api/antiraid/lockdown/toggle', (req: Request, res: Response) => {
  const { reason = 'Activación de protocolo perimetral' } = req.body;
  botState.antiraid.lockdownActive = !botState.antiraid.lockdownActive;

  if (botState.antiraid.lockdownActive) {
    const until = new Date(Date.now() + 5 * 60 * 1000).toISOString();
    botState.antiraid.lockdownUntil = until;
    botState.antiraid.lockdownChannels = ['#senado-general', '#bienvenida-oficial', '#anuncios-oficiales', '#tribuna-comandos'];
    botState.moderation.auditLogs.unshift({
      id: `log-${Date.now()}`,
      actor: 'Cancillería Soberana',
      action: 'PERIMETER_LOCKDOWN_ENGAGED',
      details: `${reason} - Restricción protocolar de canales durante 5 minutos`,
      timestamp: new Date().toISOString(),
    });
  } else {
    botState.antiraid.lockdownUntil = null;
    botState.antiraid.lockdownChannels = [];
    botState.moderation.auditLogs.unshift({
      id: `log-${Date.now()}`,
      actor: 'Cancillería Soberana',
      action: 'LOCKDOWN_RESTORED',
      details: 'Permisos de circulación restaurados en los canales del senado.',
      timestamp: new Date().toISOString(),
    });
  }

  res.json({
    lockdownActive: botState.antiraid.lockdownActive,
    until: botState.antiraid.lockdownUntil,
    affectedChannels: botState.antiraid.lockdownChannels,
  });
});

app.post('/api/antiraid/detector/toggle', (req: Request, res: Response) => {
  const { key, enabled } = req.body;
  const detector = (botState.antiraid.detectors as Record<string, any>)[key];
  if (!detector) {
    // Plain text error style
    res.status(404).json({ error: 'Módulo de defensa no localizado.' });
    return;
  }
  detector.enabled = Boolean(enabled);
  botState.moderation.auditLogs.unshift({
    id: `log-${Date.now()}`,
    actor: 'Cancillería Soberana',
    action: 'DETECTOR_STATUS_ALTERED',
    details: `${detector.name}: ${detector.enabled ? 'Operativo' : 'Pausado'}`,
    timestamp: new Date().toISOString(),
  });
  res.json({ success: true, detector });
});

// ==========================================
// SOVEREIGN PANEL & PROFILE
// ==========================================

app.get('/api/lux/panel', (req: Request, res: Response) => {
  const userId = (req.query.userId as string) || 'user-001';
  let member = botState.economy.members.find((m) => m.id === userId);
  if (!member) {
    member = botState.economy.members[0];
  }
  res.json({
    user: member,
    guildName: botState.guild.name,
    preferences: {
      language: botState.guild.language,
      responsesPublic: botState.guild.responsesPublic,
      notificationsEnabled: botState.guild.notificationsEnabled,
    },
    catalog: botState.economy.shopItems,
  });
});

app.post('/api/lux/claim-daily', (req: Request, res: Response) => {
  const { userId = 'user-001' } = req.body;
  const member = botState.economy.members.find((m) => m.id === userId);
  if (!member) {
    res.status(404).json({ error: '⚠️ Miembro protocolar no localizado.' });
    return;
  }

  const lastClaim = new Date(member.lastDaily).getTime();
  const now = Date.now();
  const hoursSince = (now - lastClaim) / (1000 * 60 * 60);

  if (hoursSince < 24) {
    const waitHours = Math.ceil(24 - hoursSince);
    // Plain text error (NO EMBED)
    res.status(400).json({
      error: `⏳ **Estipendio diario ya reclamado:** Su asignación protocolar se renovará en ${waitHours} horas.`,
      cooldown: true,
    });
    return;
  }

  member.balance += 250;
  member.xp += 25;
  if (member.xp >= 100) {
    member.level += 1;
    member.xp = member.xp % 100;
  }
  member.lastDaily = new Date().toISOString();

  res.json({
    success: true,
    message: '✨ Se han acreditado 250 monedas soberanas a su tesorería personal.',
    newBalance: member.balance,
    level: member.level,
    xp: member.xp,
  });
});

app.post('/api/lux/settings', (req: Request, res: Response) => {
  const { key, value } = req.body;
  if (key === 'language') {
    botState.guild.language = value;
  } else if (key === 'responsesPublic') {
    botState.guild.responsesPublic = Boolean(value);
  } else if (key === 'notificationsEnabled') {
    botState.guild.notificationsEnabled = Boolean(value);
  } else if (key === 'welcomeMessage') {
    botState.guild.welcomeMessage = String(value);
  }
  res.json({ success: true, guild: botState.guild });
});

// ==========================================
// ECONOMY: CASINO, SHOP & TRANSFERS
// ==========================================

app.get('/api/economy', (_req: Request, res: Response) => {
  res.json({
    totalChips: botState.economy.totalChipsInCirculation,
    casinoBank: botState.economy.casinoBank,
    items: botState.economy.shopItems,
    members: botState.economy.members,
  });
});

app.post('/api/economy/roulette', (req: Request, res: Response) => {
  const { userId = 'user-001', bet, color } = req.body;
  const numBet = parseInt(bet, 10);

  if (isNaN(numBet) || numBet < 1 || numBet > 1000000) {
    res.status(400).json({ error: '⚠️ **Monto inválido:** La postura protocolar debe situarse entre 1 y 1.000.000 monedas.' });
    return;
  }

  const member = botState.economy.members.find((m) => m.id === userId);
  if (!member) {
    res.status(404).json({ error: '⚠️ Ciudadano no localizado.' });
    return;
  }

  if (member.balance < numBet) {
    res.status(400).json({ error: '💸 **Fondos insuficientes:** Su tesorería no cuenta con monedas suficientes para esta postura.' });
    return;
  }

  // Spin roulette: 0-36
  const spunNumber = Math.floor(Math.random() * 37);
  let resultColor = 'negro';
  if (spunNumber === 0) {
    resultColor = 'verde';
  } else if ([1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36].includes(spunNumber)) {
    resultColor = 'rojo';
  }

  const win = color === resultColor;
  let multiplier = 0;
  if (win) {
    multiplier = resultColor === 'verde' ? 36 : 2;
  }

  const payout = win ? numBet * multiplier : 0;
  member.balance = member.balance - numBet + payout;

  if (win) {
    botState.economy.casinoBank -= payout - numBet;
  } else {
    botState.economy.casinoBank += numBet;
  }

  res.json({
    number: spunNumber,
    color: resultColor,
    win,
    payout,
    newBalance: member.balance,
    message: win
      ? `✨ ¡Acierto distinguido en el número ${spunNumber} (${resultColor.toUpperCase()})! Ha recibido ${payout.toLocaleString()} monedas soberanas.`
      : `⚜️ La ruleta se detuvo en el ${spunNumber} (${resultColor.toUpperCase()}). La banca soberana retiene la postura.`,
  });
});

app.post('/api/economy/buy', (req: Request, res: Response) => {
  const { userId = 'user-001', itemId } = req.body;
  const member = botState.economy.members.find((m) => m.id === userId);
  const item = botState.economy.shopItems.find((i) => i.id === itemId);

  if (!member || !item) {
    res.status(404).json({ error: '⚠️ Artículo o miembro no encontrado en los registros.' });
    return;
  }

  if (member.balance < item.price) {
    res.status(400).json({ error: '💸 **Fondos insuficientes:** No posee las monedas requeridas para esta adquisición.' });
    return;
  }

  member.balance -= item.price;
  member.inventory.push(item.name);

  botState.moderation.auditLogs.unshift({
    id: `log-${Date.now()}`,
    actor: member.username,
    action: 'TREASURY_ACQUISITION',
    details: `Adquisición protocolar: ${item.name} por ${item.price} monedas`,
    timestamp: new Date().toISOString(),
  });

  res.json({
    success: true,
    newBalance: member.balance,
    inventory: member.inventory,
    message: `✨ Ha incorporado '${item.name}' a su expediente protocolar.`,
  });
});

// ==========================================
// MODERATION & SECURITY CENTER
// ==========================================

app.get('/api/moderation', (_req: Request, res: Response) => {
  res.json(botState.moderation);
});

app.post('/api/moderation/sanction', (req: Request, res: Response) => {
  const { action, targetUsername, reason = 'Incumplimiento de estatutos solemnes' } = req.body;
  if (!targetUsername) {
    res.status(400).json({ error: '⚠️ Se requiere identificar al miembro destinatario.' });
    return;
  }

  const timestamp = new Date().toISOString();
  if (action === 'ban') {
    botState.moderation.bans.unshift({
      userId: `user-${Date.now().toString().slice(-4)}`,
      username: targetUsername,
      reason,
      bannedAt: timestamp,
      moderator: 'Tribunal de Cancillería',
    });
  } else if (action === 'timeout') {
    botState.moderation.timeouts.unshift({
      userId: `user-${Date.now().toString().slice(-4)}`,
      username: targetUsername,
      reason,
      until: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
      moderator: 'Tribunal de Cancillería',
    });
  }

  botState.moderation.auditLogs.unshift({
    id: `log-${Date.now()}`,
    actor: 'Tribunal de Cancillería',
    action: action.toUpperCase(),
    details: `Sanción de ${action} aplicada a @${targetUsername}: ${reason}`,
    timestamp,
  });

  res.json({ success: true, message: `✨ Decreto de ${action} emitido formalmente contra @${targetUsername}.` });
});

app.post('/api/moderation/appeals/resolve', (req: Request, res: Response) => {
  const { appealId, status } = req.body;
  const appeal = botState.moderation.appeals.find((a) => a.id === appealId);
  if (!appeal) {
    res.status(404).json({ error: '⚠️ Expediente de apelación no localizado.' });
    return;
  }

  appeal.status = status;
  if (status === 'accepted') {
    if (appeal.sanction === 'ban') {
      botState.moderation.bans = botState.moderation.bans.filter((b) => b.userId !== appeal.targetId);
    } else if (appeal.sanction === 'timeout') {
      botState.moderation.timeouts = botState.moderation.timeouts.filter((t) => t.userId !== appeal.targetId);
    }
  }

  botState.moderation.auditLogs.unshift({
    id: `log-${Date.now()}`,
    actor: 'Tribunal de Cancillería',
    action: `APPEAL_${status.toUpperCase()}`,
    details: `Apelación #${appeal.id} de @${appeal.username} resuelta como ${status}`,
    timestamp: new Date().toISOString(),
  });

  res.json({ success: true, appeal });
});

// ==========================================
// LUCIFER DIPLOMATIC ORACLE (AI)
// ==========================================

const getAiClient = () => {
  const apiKey = process.env.GEMINI_API_KEY;
  if (apiKey) {
    return new GoogleGenAI({ apiKey });
  }
  return null;
};

const LUCIFER_SOVEREIGN_PROMPT = `Eres Lucifer, soberano de intachable distinción, diplomacia, porte aristocrático y elocuencia impecable.
Tu presencia impone respeto absoluto a través de la educación, el carisma sereno, la elegancia formal y el ingenio refinado.
Hablas en un español culto, protocolar, noble y seductor. Eres franco, inteligente y sumamente perspicaz.
NO tienes ninguna relación con series de Netflix, bares, clubes nocturnos ni personajes de ficción televisiva.
Eres un líder sobrio, carismático y magnético. Tratas al usuario con cortesía y honor ("distinguido invitado", "estimado miembro").
Responde de forma clara y elocuente en 2 a 4 oraciones.`;

app.post('/api/ai/ask', async (req: Request, res: Response) => {
  const { question, mode = 'ask', targetUser } = req.body;
  if (!question && !targetUser) {
    res.status(400).json({ error: '⚠️ Se requiere formular una consulta o indicar el destinatario.' });
    return;
  }

  const aiClient = getAiClient();
  let responseText = '';

  try {
    if (aiClient) {
      let prompt = question;
      if (mode === 'roast') {
        prompt = `Ofrece un brindis perspicaz, sutilmente mordaz pero distinguido y noble para '@${targetUser}'. Máximo 2 oraciones de alta diplomacia sin ofensas vulgares.`;
      } else if (mode === 'piano') {
        prompt = `Lucifer compone al piano de salón una estrofa solemne, majestuosa y rimada de 4 a 6 versos sobre el tema: '${question}'. Tono noble y reflexivo.`;
      } else if (mode === 'advice') {
        prompt = `Un miembro distinguido solicita consejo sobre este dilema: '${question}'. Brinda una orientación estratégica, pragmática e inteligente sin sermones moralistas.`;
      } else if (mode === 'investigate') {
        prompt = `Examina estos antecedentes con mente deductiva e imparcial: '${question}'. Emite un dictamen formal distinguiendo evidencias de habladurías.`;
      }

      const aiResponse = await aiClient.models.generateContent({
        model: 'gemini-2.5-flash',
        contents: prompt,
        config: {
          systemInstruction: LUCIFER_SOVEREIGN_PROMPT,
          temperature: 0.8,
        },
      });

      responseText = aiResponse.text || '';
    }
  } catch (error) {
    console.warn('AI call fallback to sovereign local engine:', error);
  }

  // Fallback high-distinction responses if no API key
  if (!responseText) {
    if (mode === 'roast') {
      responseText = `Distinguido ${targetUser || 'amigo'}: presentarse en los estrados principales con semejante desparpajo y tan escasa prudencia es casi una proeza. Le obsequiaré un brindis por su audacia; no todos sobreviven a sus propias palabras con tanta elegancia.`;
    } else if (mode === 'piano') {
      responseText = `*Las notas resuenan en la solemnidad del salón:*\n\nEn el silencio noble donde el juicio aguarda,\nla verdad serena sus laureles guarda.\nNo es la voz más alta la que el mando ostenta,\nsino la templanza que al destino enfrenta.`;
    } else if (mode === 'advice') {
      responseText = `La resolución de toda encrucijada, distinguido miembro, depende de una regla cardinal: nunca conceda en la mesa de negociaciones aquello que no esté dispuesto a defender con convicción. Tome la iniciativa con sobriedad y la victoria será suya.`;
    } else if (mode === 'investigate') {
      responseText = `Los elementos examinados no admiten ambigüedad protocolar. Quien alega agravio sin aportar testimonio documentado simplemente busca inclinar la balanza a su favor. La sentencia es clara: los hechos corroboran la inocencia y desechan el rumor.`;
    } else {
      responseText = `Agradezco su consulta. En los asuntos de alta diplomacia, he comprobado que los hombres rara vez fracasan por falta de fuerza; fracasan por carecer del temple necesario para sostener su propia palabra. Mantenga la serenidad y la distinción marcará su camino.`;
    }
  }

  // Format as clean charismatic plain text (NO EMBED) as explicitly required
  const formattedPlainText = `### ⚜️ Audiencia Oficial con Lucifer\n\n> **Consulta presentada por @${targetUser || 'Distinguido Miembro'}:**\n> ${question || 'Petición de dictamen protocolar'}\n\n**Dictamen de Lucifer:**\n${responseText.trim()}\n\n*Transmitido con distinción protocolar y cortesía absoluta. 🥂*`;

  const aiEntry = {
    id: `ai-${Date.now()}`,
    sender: targetUser || 'Distinguido Miembro',
    prompt: question || `Consulta oficial de ${targetUser}`,
    response: responseText,
    formattedText: formattedPlainText,
    command: `/${mode}`,
    timestamp: new Date().toISOString(),
  };

  botState.aiHistory.unshift(aiEntry);
  if (botState.aiHistory.length > 20) {
    botState.aiHistory.pop();
  }

  res.json({
    success: true,
    result: responseText,
    formattedText: formattedPlainText,
    entry: aiEntry,
  });
});

app.get('/api/ai/history', (_req: Request, res: Response) => {
  res.json(botState.aiHistory);
});

// ==========================================
// DISCORD COMMAND SIMULATOR (Official Previews)
// ==========================================

app.post('/api/commands/simulate', (req: Request, res: Response) => {
  const { command, args = {} } = req.body;
  const guildName = botState.guild.name;

  let embed: any = null;
  let content = '';

  // Apply User's rule:
  // - Embeds ONLY for commands that deserve them (/perfil, /panel, /tienda, /ruleta, /seguridad, /antiraid)
  // - Plain text (NO EMBED) for /ask, /anuncio, and errors!

  switch (command) {
    case '/ask':
      content = `### ⚜️ Audiencia Oficial con Lucifer\n\n> **Consulta de @${args.user || 'Ciudadano'}:**\n> ${args.question || '¿Cuál es el secreto de la verdadera influencia?'}\n\n**Respuesta de Lucifer:**\nLa verdadera influencia no precisa estridencias ni imposición; surge de una cortesía inquebrantable, una palabra firme y la sabiduría de saber cuándo escuchar en silencio. Aquel que domina su propio juicio es quien gobierna el salón.\n\n*Transmitido con distinción protocolar y cortesía absoluta. 🥂*`;
      break;

    case '/anuncio':
      content = `📢 **COMUNICADO OFICIAL DE CANCILLERÍA**\n\n${args.message || 'Se convoca al honorable senado para la sesión extraordinaria de condecoraciones y balances protocolar.'}\n\n*⚜️ Emitido con distinción y vigencia protocolar bajo sello de Lucifer en ${guildName}.*`;
      break;

    case '/error-ejemplo':
      content = `⚠️ **Atención protocolar:** No fue posible ejecutar la acción solicitada. Los parámetros especificados no coinciden con las directivas oficiales vigentes. Por favor, revise su sintaxis.`;
      break;

    case '/perfil':
    case '/panel':
    case '/lux': // legacy alias mapped to sovereign protocol
      embed = {
        title: '⚜️・Expediente Soberano de Membresía',
        description: 'Registro oficial de honores, tesorería y estipendio en la sede principal.',
        color: 0x4a000a,
        fields: [
          { name: '👑 Rango Soberano', value: '💎 Gran Soberano', inline: true },
          { name: '⚜️ Nivel Protocolar', value: 'Nivel 16', inline: true },
          { name: '🪙 Monedas Soberanas', value: '🪙 5.400 en Bóveda', inline: true },
          { name: '⚡ Progreso Protocolar', value: '80 / 100 XP `[▰▰▰▰▰▰▰▰▱▱]`', inline: false },
        ],
        footer: {
          text: 'Lucifer Protocol • Soberanía, diplomacia y distinción absoluta ⚜️',
          icon_url: 'https://cdn-icons-png.flaticon.com/512/10691/10691880.png',
        },
        timestamp: new Date().toISOString(),
      };
      break;

    case '/tienda':
      embed = {
        title: '⚜️・CATÁLOGO DE ALTA DISTINCIÓN',
        description: 'Artículos de honor, sellos y condecoraciones oficiales adquiribles con moneda soberana.',
        color: 0x4a000a,
        fields: botState.economy.shopItems.map((item) => ({
          name: `${item.icon} ${item.name} • 🪙 ${item.price.toLocaleString()}`,
          value: item.description,
          inline: true,
        })),
        footer: {
          text: 'Lucifer Protocol • Tesorería de Alta Distinción ⚜️',
          icon_url: 'https://cdn-icons-png.flaticon.com/512/10691/10691880.png',
        },
        timestamp: new Date().toISOString(),
      };
      break;

    case '/ruleta':
      const bet = args.bet || 200;
      const color = args.color || 'rojo';
      embed = {
        title: '🎰・MESA DE RULETA SOBERANA',
        description: `Postura de **${bet} monedas** sobre el **${color.toUpperCase()}**.\n\n✨ ¡La bola de marfil se detuvo en el **18 (${color.toUpperCase()})**!\n¡Su postura ha sido premiada con **${bet * 2} monedas soberanas**!`,
        color: 0x1a472a,
        footer: {
          text: 'Lucifer Protocol • Ruleta de Salón ⚜️',
          icon_url: 'https://cdn-icons-png.flaticon.com/512/10691/10691880.png',
        },
        timestamp: new Date().toISOString(),
      };
      break;

    case '/seguridad':
    case '/puertas-infernales': // mapped to official security
      embed = {
        title: '🛡️・CONSOLA DE ALTA SEGURIDAD SOBERANA',
        description: 'Consola protocolar oficial: supervisión de sanciones vigentes, apelaciones formales y contingencia perimetral.',
        color: 0x4a000a,
        fields: [
          { name: '⛓️ Baneos Vigentes', value: `${botState.moderation.bans.length}`, inline: true },
          { name: '🤐 Silencios Activos', value: `${botState.moderation.timeouts.length}`, inline: true },
          { name: '🚨 Bloqueo Perimetral', value: botState.antiraid.lockdownActive ? 'ACTIVO' : 'Desactivado', inline: false },
          { name: '👁️ Radar Antiraid', value: 'Vigilancia activa • 0 amenazas perimetrales', inline: false },
        ],
        footer: {
          text: 'Lucifer Protocol • Protección Perimetral ⚜️',
          icon_url: 'https://cdn-icons-png.flaticon.com/512/10691/10691880.png',
        },
        timestamp: new Date().toISOString(),
      };
      break;

    case '/fichas':
      content = `✨ **Estipendio soberano acreditado:** Se han transferido **250 monedas** a su tesorería personal. Saldo en bóveda: **5.650 monedas**.`;
      break;

    case '/saldo':
      embed = {
        title: '⚜️・ESTADO DE TESORERÍA',
        description: `Haberes registrados para el ciudadano <@user-001>.`,
        color: 0x4a000a,
        fields: [
          { name: '🪙 Fondos en Bóveda', value: '**5.400 monedas soberanas**', inline: true },
          { name: '👑 Estipendio Diario', value: 'Disponible vía `/fichas`', inline: true },
        ],
        footer: {
          text: 'Lucifer Protocol • Bóveda Soberana',
          icon_url: 'https://cdn-icons-png.flaticon.com/512/10691/10691880.png',
        },
        timestamp: new Date().toISOString(),
      };
      break;

    default:
      content = `⚜️ **Protocolo Lucifer:** Comando \`${command}\` ejecutado con éxito en **${guildName}**.`;
      break;
  }

  res.json({
    command,
    content,
    embed,
  });
});

// Production static assets
if (process.env.NODE_ENV === 'production') {
  app.use(express.static(path.join(__dirname, 'dist')));
  app.get('*', (_req: Request, res: Response) => {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
  });
} else {
  // Mount Vite dev server middlewares in development
  import('vite').then(async ({ createServer }) => {
    const vite = await createServer({
      server: { middlewareMode: true },
      appType: 'spa',
    });
    app.use(vite.middlewares);
  });
}

app.listen(PORT, HOST, () => {
  console.log(`⚜️ Lucifer Protocol operativo en http://${HOST}:${PORT}`);
  console.log(`🛡️ Sede soberana y servicio de salud activo en http://${HOST}:${PORT}/ready`);
});
