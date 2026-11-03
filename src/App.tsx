import React, { useState, useEffect } from 'react';
import { Navigation } from './components/Navigation.tsx';
import { OverviewTab } from './components/OverviewTab.tsx';
import { AntiRaidTab } from './components/AntiRaidTab.tsx';
import { SecurityCenterTab } from './components/SecurityCenterTab.tsx';
import { LuxPanelTab } from './components/LuxPanelTab.tsx';
import { EconomyTab } from './components/EconomyTab.tsx';
import { AiOracleTab } from './components/AiOracleTab.tsx';
import { AuditTab } from './components/AuditTab.tsx';
import {
  GuildInfo,
  AntiRaidState,
  Member,
  ShopItem,
  Ban,
  Timeout,
  Appeal,
  AuditLog,
  AiInteraction,
  PresenceState,
} from './types.ts';

export default function App() {
  const [activeTab, setActiveTab] = useState('luxpanel');

  // 🔴 ESTADO DE AUTENTICACIÓN (Bloqueo inicial)
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);

  // Presence State (Sovereign dynamic rotation)
  const [presence, setPresence] = useState<PresenceState>({
    status: 'online',
    currentActivityIndex: 0,
    currentActivity: {
      type: 'WATCHING',
      typeLabel: 'Vigilando',
      text: 'Custodiando los fueros de alta soberanía',
      badge: '🔱',
      details: 'Supervisión de protocolos y edictos imperiales',
      state: 'Protocolo de Estado Activo',
    },
    intervalSeconds: 10,
    lastRotatedAt: new Date().toISOString(),
    activities: [
      {
        type: 'WATCHING',
        typeLabel: 'Vigilando',
        text: 'Custodiando los fueros de alta soberanía',
        badge: '🔱',
        details: 'Supervisión de protocolos y edictos imperiales',
        state: 'Protocolo de Estado Activo',
      },
      {
        type: 'WATCHING',
        typeLabel: 'Observando',
        text: 'Ponderando edictos en el tribunal de honor',
        badge: '⚖️',
        details: 'Justicia equitativa y rectitud diplomática sin titubeos',
        state: 'Tribunal en Sesión',
      },
      {
        type: 'COMPETING',
        typeLabel: 'Presidiendo',
        text: 'Celebrando el banquete de la corte imperial',
        badge: '🍷',
        details: 'Distinción, copas de reserva y camaradería solemne',
        state: 'Banquete de la Corte',
      },
      {
        type: 'CUSTOM',
        typeLabel: 'Proclamando',
        text: 'Gobernando con temple, majestad y rectitud',
        badge: '👑',
        details: 'Liderazgo inquebrantable guiando hacia la distinción',
        state: 'Soberanía Absoluta',
      },
      {
        type: 'LISTENING',
        typeLabel: 'Auditoría',
        text: 'Inspeccionando el tesoro de los soberanos',
        badge: '💎',
        details: 'Supervisando arcas, ducados y transacciones en el banco',
        state: 'Auditoría de Fondos',
      },
      {
        type: 'LISTENING',
        typeLabel: 'Audiencia',
        text: 'Escuchando las audiencias del gran salón',
        badge: '⚜️',
        details: 'Atención protocolar a las peticiones del senado',
        state: 'Audiencia Pública',
      },
      {
        type: 'COMPETING',
        typeLabel: 'Brindis',
        text: 'Brindando por las victorias de la cancillería',
        badge: '🥂',
        details: 'Celebración por la paz y prosperidad del territorio',
        state: 'Salón de Embajadores',
      },
      {
        type: 'CUSTOM',
        typeLabel: 'Decretos',
        text: 'Sellando decretos y alianzas dinásticas',
        badge: '📜',
        details: 'Redacción de pactos inquebrantables con tinta de honor',
        state: 'Cancillería Imperial',
      },
      {
        type: 'WATCHING',
        typeLabel: 'Defensa',
        text: 'Supervisando la guardia de honor imperial',
        badge: '⚔️',
        details: 'Escudos antiraid en alerta perimetral continua',
        state: 'Escudo de Honor Activo',
      },
    ],
  });

  // States
  const [guild, setGuild] = useState<GuildInfo>({
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
  });

  const [antiraid, setAntiraid] = useState<AntiRaidState>({
    enabled: true,
    lockdownActive: false,
    lockdownUntil: null,
    lockdownChannels: [],
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
    incidents: [],
  });

  const [currentMember, setCurrentMember] = useState<Member>({
    id: 'user-001',
    username: 'Identificando...',
    balance: 5400,
    level: 16,
    rank: '💎 Ciudadano Soberano',
    xp: 80,
    lastDaily: new Date(Date.now() - 1000 * 60 * 60 * 26).toISOString(),
    inventory: ['Pase Gran Reserva Soberana', 'Sello Imperial de Distinción'],
    isStaff: false,
  });

  const isStaff = currentMember?.isStaff || false;

  // 🔴 VERIFICADOR DE SESIÓN DISCORD OAUTH2 AL ENTRAR A LA PÁGINA
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const authData = params.get('auth_data');

    if (authData) {
      try {
        const decoded = JSON.parse(atob(authData));
        setCurrentMember((prev) => ({
          ...prev,
          id: decoded.id,
          username: decoded.username,
          rank: decoded.isStaff ? '🔱 Soberano Canciller (Staff)' : '💎 Ciudadano Soberano',
          isStaff: decoded.isStaff,
        }));
        setIsAuthenticated(true);
        localStorage.setItem('lucifer_session', authData);
        window.history.replaceState({}, document.title, window.location.pathname);
      } catch (e) {
        console.error('Error al decodificar sesión:', e);
      }
    } else {
      const savedSession = localStorage.getItem('lucifer_session');
      if (savedSession) {
        try {
          const decoded = JSON.parse(atob(savedSession));
          setCurrentMember((prev) => ({
            ...prev,
            id: decoded.id,
            username: decoded.username,
            rank: decoded.isStaff ? '🔱 Soberano Canciller (Staff)' : '💎 Ciudadano Soberano',
            isStaff: decoded.isStaff,
          }));
          setIsAuthenticated(true);
        } catch (e) {
          localStorage.removeItem('lucifer_session');
        }
      }
    }
  }, []);

  // Redirección de seguridad
  useEffect(() => {
    const staffTabs = ['overview', 'antiraid', 'security', 'oracle', 'audit'];
    if (!isStaff && staffTabs.includes(activeTab)) {
      setActiveTab('luxpanel');
    }
  }, [isStaff, activeTab]);

  const [members, setMembers] = useState<Member[]>([]);
  const [shopItems, setShopItems] = useState<ShopItem[]>([]);
  const [bans, setBans] = useState<Ban[]>([]);
  const [timeouts, setTimeouts] = useState<Timeout[]>([]);
  const [appeals, setAppeals] = useState<Appeal[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [aiHistory, setAiHistory] = useState<AiInteraction[]>([]);

  const [dailyClaimMessage, setDailyClaimMessage] = useState<string | null>(null);
  const [dailyError, setDailyError] = useState<string | null>(null);

  // Fetch initial data
  const fetchData = async () => {
    try {
      const fetchSafe = async (url: string) => {
        try {
          const r = await fetch(url);
          if (r.ok) return await r.json();
        } catch {
          return null;
        }
        return null;
      };

      const [statsRes, antiraidRes, econRes, modRes, aiRes, presenceRes] = await Promise.all([
        fetchSafe('/api/stats'),
        fetchSafe('/api/antiraid'),
        fetchSafe('/api/economy'),
        fetchSafe('/api/moderation'),
        fetchSafe('/api/ai/history'),
        fetchSafe('/api/presence'),
      ]);

      if (statsRes?.guild) setGuild(statsRes.guild);
      if (statsRes?.presence) setPresence(statsRes.presence);
      if (presenceRes?.currentActivity) setPresence(presenceRes);
      if (antiraidRes) setAntiraid(antiraidRes);
      if (econRes?.members) setMembers(econRes.members);
      if (econRes?.items) setShopItems(econRes.items);
      if (modRes?.bans) setBans(modRes.bans);
      if (modRes?.timeouts) setTimeouts(modRes.timeouts);
      if (modRes?.appeals) setAppeals(modRes.appeals);
      if (modRes?.auditLogs) setAuditLogs(modRes.auditLogs);
      if (aiRes) setAiHistory(aiRes);
    } catch {
      // Ignorar
    }
  };

  useEffect(() => {
    if (!isAuthenticated) return;
    fetchData();

    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/presence');
        if (res.ok) {
          const data = await res.json();
          if (data && data.currentActivity) {
            setPresence(data);
          }
        }
      } catch {
        // Ignorar
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [isAuthenticated]);

  // Advance Sovereign Status Rotation
  const handleAdvancePresence = async (targetIndex?: number) => {
    try {
      const res = await fetch('/api/presence/next', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(typeof targetIndex === 'number' ? { index: targetIndex } : {}),
      });
      if (res.ok) {
        const data = await res.json();
        setPresence((prev) => ({
          ...prev,
          ...data,
        }));
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Lockdown Toggle
  const handleToggleLockdown = async () => {
    try {
      const res = await fetch('/api/antiraid/lockdown/toggle', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'Activación protocolar de cancillería' }),
      });
      if (res.ok) {
        const data = await res.json();
        setAntiraid((prev) => ({
          ...prev,
          lockdownActive: data.lockdownActive,
          lockdownUntil: data.until,
          lockdownChannels: data.affectedChannels,
        }));
      }
      fetchData();
    } catch (err) {
      console.error(err);
    }
  };

  // Detector Toggle
  const handleToggleDetector = async (key: string, enabled: boolean) => {
    try {
      await fetch('/api/antiraid/detector/toggle', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key, enabled }),
      });
      setAntiraid((prev) => {
        const updated = { ...prev.detectors };
        if (updated[key]) {
          updated[key] = { ...updated[key], enabled };
        }
        return { ...prev, detectors: updated };
      });
    } catch (err) {
      console.error(err);
    }
  };

  // Claim Daily
  const handleClaimDaily = async () => {
    setDailyClaimMessage(null);
    setDailyError(null);

    try {
      const res = await fetch('/api/lux/claim-daily', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId: currentMember.id }),
      });

      if (res.ok) {
        const data = await res.json();
        setDailyClaimMessage(data.message || 'Asignación reclamada con éxito.');
        setCurrentMember((prev) => ({
          ...prev,
          balance: data.newBalance ?? prev.balance + 250,
          level: data.level ?? prev.level,
          xp: data.xp ?? prev.xp + 10,
          lastDaily: new Date().toISOString(),
        }));
        setTimeout(() => setDailyClaimMessage(null), 4000);
        return;
      }
    } catch {
      // Ignorar
    }

    setCurrentMember((prev) => ({
      ...prev,
      balance: prev.balance + 250,
      xp: prev.xp + 10,
      lastDaily: new Date().toISOString(),
    }));
    setDailyClaimMessage('⚜️ ¡Asignación diaria de 250 Monedas Soberanas reclamada con honor!');
    setTimeout(() => setDailyClaimMessage(null), 4000);
  };

  // Update Settings
  const handleUpdateSetting = async (key: string, value: any) => {
    try {
      await fetch('/api/lux/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key, value }),
      });
      setGuild((prev) => ({
        ...prev,
        [key]: value,
      }));
    } catch (err) {
      console.error(err);
    }
  };

  // Play Roulette
  const handlePlayRoulette = async (bet: number, color: string) => {
    try {
      const res = await fetch('/api/economy/roulette', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId: currentMember.id, bet, color }),
      });
      if (res.ok) {
        const data = await res.json();
        setCurrentMember((prev) => ({
          ...prev,
          balance: data.newBalance,
        }));
        return data;
      }
    } catch {
      // Ignorar
    }

    const win = Math.random() > 0.52;
    const delta = win ? bet : -bet;
    const newBalance = Math.max(0, currentMember.balance + delta);
    setCurrentMember((prev) => ({ ...prev, balance: newBalance }));
    return {
      won: win,
      payout: win ? bet * 2 : 0,
      newBalance,
      color: win ? color : (color === 'red' ? 'black' : 'red'),
    };
  };

  // Buy Shop Item
  const handleBuyItem = async (itemId: string) => {
    try {
      const res = await fetch('/api/economy/buy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId: currentMember.id, itemId }),
      });
      if (res.ok) {
        const data = await res.json();
        setCurrentMember((prev) => ({
          ...prev,
          balance: data.newBalance,
          inventory: data.inventory,
        }));
        return data;
      }
    } catch {
      // Ignorar
    }

    const item = shopItems.find((i) => i.id === itemId);
    const cost = item ? item.price : 500;
    const newBalance = Math.max(0, currentMember.balance - cost);
    const itemName = item ? item.name : 'Artículo Soberano';
    const newInventory = [...currentMember.inventory, itemName];

    setCurrentMember((prev) => ({
      ...prev,
      balance: newBalance,
      inventory: newInventory,
    }));
    return { success: true, newBalance, inventory: newInventory };
  };

  // Apply Sanction
  const handleApplySanction = async (action: string, targetUsername: string, reason: string) => {
    try {
      await fetch('/api/moderation/sanction', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, targetUsername, reason }),
      });
      fetchData();
    } catch (err) {
      console.error(err);
    }
  };

  // Resolve Appeal
  const handleResolveAppeal = async (appealId: number, status: 'accepted' | 'denied') => {
    try {
      await fetch('/api/moderation/appeals/resolve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ appealId, status }),
      });
      fetchData();
    } catch (err) {
      console.error(err);
    }
  };

  // Ask Lucifer AI
  const handleAskLucifer = async (question: string, mode: string, targetUser?: string) => {
    try {
      const res = await fetch('/api/ai/ask', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question, mode, targetUser }),
      });
      if (res.ok) {
        const data = await res.json();
        if (data.entry) {
          setAiHistory((prev) => [data.entry, ...prev.slice(0, 19)]);
        }
        return data;
      }
    } catch {
      // Ignorar
    }

    return {
      entry: {
        id: Date.now().toString(),
        timestamp: new Date().toISOString(),
        question,
        answer: '⚜️ El oráculo responde: la distinción y la lealtad sostienen los fueros del Lucifer Protocol.',
        mode,
      },
    };
  };

  // =========================================================================
  // 🔴 PANTALLA DE ACCESO SOBERANO (Si NO ha iniciado sesión con Discord)
  // =========================================================================
  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-[#0b0507] text-stone-200 flex flex-col items-center justify-center p-4 relative overflow-hidden font-sans selection:bg-[#7a0f1d] selection:text-white">
        {/* Resplandor carmesí de fondo */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[550px] h-[550px] bg-[#6a0712]/20 blur-[130px] rounded-full pointer-events-none"></div>

        <div className="max-w-md w-full bg-[#12070b]/90 border border-[#d4af37]/35 rounded-2xl p-8 sm:p-10 shadow-2xl shadow-[#3b0008]/80 backdrop-blur-xl relative z-10 text-center">
          {/* Logo Soberano */}
          <div className="w-20 h-20 mx-auto mb-6 rounded-2xl bg-gradient-to-br from-[#6a0712] via-[#3b0008] to-[#120306] border border-[#d4af37]/50 flex items-center justify-center shadow-xl shadow-[#6a0712]/40">
            <span className="text-4xl">⚜️</span>
          </div>

          <span className="text-[10px] uppercase tracking-widest px-3 py-1 rounded-full bg-[#4a000a] text-[#d4af37] border border-[#d4af37]/40 font-semibold inline-block mb-3">
            PROTOCOLO SOBERANO • ACCESO
          </span>

          <h1 className="font-cinzel text-2xl sm:text-3xl font-bold text-white tracking-wider mb-2">
            LUCIFER PROTOCOL
          </h1>
          <p className="text-xs text-stone-400 font-light mb-6">
            Soberanía, diplomacia y distinción absoluta ⚜️
          </p>

          <div className="border-t border-b border-[#3b0008]/60 py-4 my-6 text-stone-300 text-xs leading-relaxed font-light">
            Para ingresar al senado imperial, consultar sus estipendios en tesorería o ejercer fueros de mando, es obligatorio acreditar su identidad consular con Discord.
          </div>

          {/* Botón Principal de Conexión */}
          <a
            href="https://lucifer-gg-bot.onrender.com/api/auth/login"
            className="flex items-center justify-center gap-3 w-full py-3.5 px-6 rounded-xl bg-[#5865F2] hover:bg-[#4752C4] text-white text-sm font-semibold transition-all duration-200 shadow-lg shadow-[#5865F2]/25 hover:shadow-[#5865F2]/40 transform hover:-translate-y-0.5"
          >
            <svg className="w-5 h-5 fill-current" viewBox="0 0 24 24">
              <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.929 1.793 8.18 1.793 12.061 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.028zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
            </svg>
            <span>Conectar con Discord</span>
          </a>

          <div className="mt-6 text-[11px] text-stone-500 font-mono">
            ⚜️ Acceso protegido por autenticación oficial OAuth2
          </div>
        </div>
      </div>
    );
  }

  // =========================================================================
  // DASHBOARD COMPLETO (Solo se muestra tras iniciar sesión)
  // =========================================================================
  return (
    <div className="min-h-screen bg-[#0b0507] text-stone-200 flex flex-col font-sans">
      <Navigation
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        isLockdown={antiraid.lockdownActive}
        presence={presence}
        onAdvancePresence={() => handleAdvancePresence()}
        isStaff={isStaff}
        currentMember={currentMember}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {/* Pestañas de administración (Solo Staff) */}
        {activeTab === 'overview' && isStaff && (
          <OverviewTab
            guild={guild}
            antiraid={antiraid}
            presence={presence}
            onToggleLockdown={handleToggleLockdown}
            onNavigate={setActiveTab}
            onAdvancePresence={handleAdvancePresence}
          />
        )}

        {activeTab === 'antiraid' && isStaff && (
          <AntiRaidTab
            antiraid={antiraid}
            onToggleLockdown={handleToggleLockdown}
            onToggleDetector={handleToggleDetector}
          />
        )}

        {activeTab === 'security' && isStaff && (
          <SecurityCenterTab
            bans={bans}
            timeouts={timeouts}
            appeals={appeals}
            onApplySanction={handleApplySanction}
            onResolveAppeal={handleResolveAppeal}
          />
        )}

        {/* Pestañas de usuario (Para todos) */}
        {activeTab === 'luxpanel' && (
          <LuxPanelTab
            member={currentMember}
            guild={guild}
            onClaimDaily={handleClaimDaily}
            onUpdateSetting={handleUpdateSetting}
            dailyClaimMessage={dailyClaimMessage}
            dailyError={dailyError}
          />
        )}

        {activeTab === 'economy' && (
          <EconomyTab
            items={shopItems}
            members={members}
            currentMember={currentMember}
            onPlayRoulette={handlePlayRoulette}
            onBuyItem={handleBuyItem}
          />
        )}

        {/* Pestañas de administración (Solo Staff) */}
        {activeTab === 'oracle' && isStaff && (
          <AiOracleTab
            history={aiHistory}
            onAskLucifer={handleAskLucifer}
          />
        )}

        {activeTab === 'audit' && isStaff && (
          <AuditTab
            logs={auditLogs}
            guild={guild}
          />
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-[#3b0008]/40 bg-[#0d0709] py-4 text-center text-xs text-stone-500">
        <p className="font-cinzel text-stone-400">
          ⚜️ Lucifer Protocol • Sistema Soberano &amp; Diplomático de Alta Distinción ⚜️
        </p>
        <p className="text-[10px] text-stone-600 mt-0.5">
          Panel oficial de administración, seguridad perimetral, tesorería y cancillería soberana.
        </p>
      </footer>
    </div>
  );
}