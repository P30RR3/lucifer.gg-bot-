import React, { useState, useEffect } from 'react';
import {
  ShieldAlert,
  Coins,
  Sparkles,
  Server,
  Radio,
  Lock,
  Unlock,
  Users,
  RotateCw,
  CheckCircle2,
  Crown,
} from 'lucide-react';
import { GuildInfo, AntiRaidState, PresenceState } from '../types.ts';

interface OverviewTabProps {
  guild: GuildInfo;
  antiraid: AntiRaidState;
  presence?: PresenceState;
  onToggleLockdown: () => void;
  onNavigate: (tab: string) => void;
  onAdvancePresence?: (targetIndex?: number) => void;
}

export const OverviewTab: React.FC<OverviewTabProps> = ({
  guild,
  antiraid,
  presence,
  onToggleLockdown,
  onNavigate,
  onAdvancePresence,
}) => {
  const [rotationProgress, setRotationProgress] = useState(0);

  // Client-side visual progress bar for rotation interval
  useEffect(() => {
    const interval = setInterval(() => {
      setRotationProgress((prev) => (prev >= 100 ? 0 : prev + 10));
    }, 1000);
    return () => clearInterval(interval);
  }, [presence?.currentActivityIndex]);

  const currentActivity = presence?.currentActivity;

  return (
    <div className="space-y-6">
      {/* Hero Welcome Banner */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-[#200006] via-[#35070e] to-[#120306] border border-[#d4af37]/35 p-6 sm:p-8 shadow-2xl">
        <div className="absolute top-0 right-0 w-96 h-96 bg-[#6a0712]/20 rounded-full blur-3xl pointer-events-none -mr-20 -mt-20"></div>

        <div className="relative z-10 flex flex-col lg:flex-row lg:items-center justify-between gap-6">
          <div className="space-y-2 max-w-2xl">
            <div className="flex items-center gap-2">
              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-950 text-emerald-400 border border-emerald-700/60">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 mr-1.5 animate-pulse"></span>
                Servidor en línea • Gateway activo
              </span>
              <span className="text-xs text-[#d4af37] font-serif">⚜️ Lucifer Protocol • Cancillería Real</span>
            </div>
            <h1 className="text-2xl sm:text-3xl font-bold font-cinzel text-white tracking-wide">
              {guild.name}
            </h1>
            <p className="text-sm text-stone-300 font-light leading-relaxed">
              &quot;{guild.welcomeMessage}&quot;
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <button
              onClick={onToggleLockdown}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold transition-all shadow-lg active:scale-95 ${
                antiraid.lockdownActive
                  ? 'bg-emerald-700 hover:bg-emerald-600 text-white shadow-emerald-900/40'
                  : 'bg-red-800 hover:bg-red-700 text-white shadow-red-950/50'
              }`}
            >
              {antiraid.lockdownActive ? (
                <>
                  <Unlock className="w-4 h-4" />
                  <span>Desactivar Bloqueo</span>
                </>
              ) : (
                <>
                  <Lock className="w-4 h-4" />
                  <span>Bloqueo de Pánico (5m)</span>
                </>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* DYNAMIC SOVEREIGN STATUS ROTATION CARD */}
      <div className="bg-[#12070a] border border-[#d4af37]/40 rounded-2xl p-6 shadow-xl relative overflow-hidden">
        <div className="absolute top-0 right-0 w-64 h-64 bg-[#d4af37]/5 rounded-full blur-2xl pointer-events-none"></div>

        <div className="relative z-10 space-y-5">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#3b0008] pb-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="text-lg">🔱</span>
                <h2 className="font-cinzel font-bold text-white text-base tracking-wide">
                  ROTACIÓN DE ESTADO SOBERANO • PRESENCIA DINÁMICA
                </h2>
                <span className="text-[10px] font-bold px-2 py-0.5 rounded bg-[#4a000a] text-[#d4af37] border border-[#d4af37]/40 uppercase tracking-widest">
                  EN VIVO
                </span>
              </div>
              <p className="text-xs text-stone-300 font-light">
                Mensajes dinámicos y sofisticados que rotan periódicamente en Discord reflejando autoridad, distinción y emojis animados.
              </p>
            </div>

            <button
              onClick={() => onAdvancePresence && onAdvancePresence()}
              className="flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-bold bg-[#260910] hover:bg-[#3d0f1a] text-[#d4af37] border border-[#d4af37]/40 transition-all shadow-sm active:scale-95"
            >
              <RotateCw className="w-3.5 h-3.5" />
              <span>Rotar al Siguiente Estado</span>
            </button>
          </div>

          {/* Current Active Presence Display Card (Discord Style) */}
          {currentActivity && (
            <div className="bg-[#19090e] border border-[#510a14] rounded-2xl p-4 sm:p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-lg">
              <div className="flex items-center gap-4">
                {/* Bot Profile Circle */}
                <div className="relative shrink-0">
                  <div className="w-14 h-14 rounded-full bg-gradient-to-br from-[#6a0712] via-[#3b0008] to-[#120306] border-2 border-[#d4af37]/60 flex items-center justify-center text-2xl shadow-xl">
                    ⚜️
                  </div>
                  <span className="absolute bottom-0 right-0 w-4 h-4 rounded-full bg-emerald-500 border-2 border-[#12070a] shadow-sm"></span>
                </div>

                {/* Status Details */}
                <div className="space-y-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-bold text-white text-base">Lucifer</span>
                    <span className="px-1.5 py-0.2 rounded text-[10px] font-bold bg-[#5865f2] text-white">
                      BOT
                    </span>
                    <span className="text-xs px-2 py-0.5 rounded bg-[#2e0b13] text-[#d4af37] font-semibold border border-[#d4af37]/30">
                      {currentActivity.typeLabel}
                    </span>
                  </div>

                  <div className="text-sm font-semibold text-stone-100 flex items-center gap-2">
                    <span className="text-white text-base">{currentActivity.text}</span>
                  </div>

                  <div className="text-xs text-stone-400 flex items-center gap-2">
                    <span>{currentActivity.details}</span>
                    <span>•</span>
                    <span className="text-[#d4af37]">{currentActivity.state}</span>
                  </div>
                </div>
              </div>

              {/* Progress & Timing */}
              <div className="flex md:flex-col items-center md:items-end justify-between border-t md:border-t-0 border-[#3b0008] pt-3 md:pt-0 text-xs text-stone-400 gap-2">
                <span className="text-stone-400 font-mono text-[11px]">
                  Ciclo de rotación: 10s
                </span>
                <div className="w-32 bg-[#0b0305] h-2 rounded-full overflow-hidden border border-[#3b0008]">
                  <div
                    className="bg-gradient-to-r from-[#6a0712] to-[#d4af37] h-full transition-all duration-1000 ease-linear rounded-full"
                    style={{ width: `${rotationProgress}%` }}
                  ></div>
                </div>
              </div>
            </div>
          )}

          {/* Catalog of all Sovereign Rotating States */}
          <div className="space-y-2.5">
            <div className="flex items-center justify-between text-xs text-stone-400">
              <span className="font-semibold uppercase tracking-wider text-stone-300">
                Estados Soberanos en Rotación Activa ({presence?.activities.length || 0}):
              </span>
              <span className="text-[11px] text-stone-400">
                Haz clic en cualquier estado para activarlo de inmediato
              </span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2.5">
              {presence?.activities.map((act, index) => {
                const isCurrent = presence.currentActivityIndex === index;
                return (
                  <button
                    key={index}
                    onClick={() => onAdvancePresence && onAdvancePresence(index)}
                    className={`p-3 rounded-xl border text-left transition-all duration-150 ${
                      isCurrent
                        ? 'bg-[#4a000a] border-[#d4af37] text-white shadow-md ring-1 ring-[#d4af37]/60'
                        : 'bg-[#15070a] border-[#3b0008]/60 text-stone-300 hover:bg-[#1f0910] hover:border-[#6a0712]'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-[10px] font-bold uppercase tracking-wider text-[#d4af37]">
                        {act.badge}
                      </span>
                      {isCurrent ? (
                        <span className="text-[10px] font-bold text-emerald-400 flex items-center gap-1">
                          <CheckCircle2 className="w-3 h-3" />
                          ACTIVO
                        </span>
                      ) : (
                        <span className="text-[10px] text-stone-500 font-mono">
                          {act.typeLabel}
                        </span>
                      )}
                    </div>
                    <div className="text-xs font-semibold text-white truncate">
                      {act.text}
                    </div>
                    <div className="text-[10px] text-stone-400 mt-1 line-clamp-1">
                      {act.details}
                    </div>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {/* Metric Cards Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-[#12070a] border border-[#3b0008] p-4 rounded-xl relative overflow-hidden">
          <div className="flex items-center justify-between text-stone-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Ciudadanos en el Senado</span>
            <Users className="w-4 h-4 text-[#d4af37]" />
          </div>
          <div className="text-2xl font-bold text-white font-cinzel">
            {guild.memberCount.toLocaleString()}
          </div>
          <div className="text-[11px] text-emerald-400 mt-1 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            {guild.onlineCount} miembros presentes ahora
          </div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-4 rounded-xl relative overflow-hidden">
          <div className="flex items-center justify-between text-stone-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Escudo Perimetral</span>
            <ShieldAlert className="w-4 h-4 text-rose-500" />
          </div>
          <div className="text-2xl font-bold text-white font-cinzel">
            {antiraid.liveMetrics.threatLevel}
          </div>
          <div className="text-[11px] text-stone-400 mt-1">
            {antiraid.incidents.length} incursiones neutralizadas
          </div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-4 rounded-xl relative overflow-hidden">
          <div className="flex items-center justify-between text-stone-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Bóveda Soberana</span>
            <Coins className="w-4 h-4 text-[#d4af37]" />
          </div>
          <div className="text-2xl font-bold text-[#d4af37] font-cinzel">
            🪙 2.45M
          </div>
          <div className="text-[11px] text-stone-400 mt-1">
            680K monedas en reserva de cancillería
          </div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-4 rounded-xl relative overflow-hidden">
          <div className="flex items-center justify-between text-stone-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Estructura &amp; Fueros</span>
            <Server className="w-4 h-4 text-purple-400" />
          </div>
          <div className="text-2xl font-bold text-white font-cinzel">
            {guild.channelsCount} / {guild.rolesCount}
          </div>
          <div className="text-[11px] text-stone-400 mt-1">
            {guild.channelsCount} canales • {guild.rolesCount} rangos
          </div>
        </div>
      </div>

      {/* Main Sections Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Antiraid Status Widget */}
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-[#d4af37]" />
              <h2 className="font-cinzel font-bold text-white text-sm tracking-wide">
                DEFENSA PERIMETRAL SOBERANA
              </h2>
            </div>
            <button
              onClick={() => onNavigate('antiraid')}
              className="text-xs text-[#d4af37] hover:underline"
            >
              Configurar →
            </button>
          </div>

          <div className="space-y-2.5">
            {Object.entries(antiraid.detectors).slice(0, 4).map(([key, d]) => (
              <div
                key={key}
                className="flex items-center justify-between p-2.5 rounded-lg bg-[#19090e] border border-[#3b0008]/40"
              >
                <div>
                  <div className="text-xs font-medium text-stone-200">{d.name}</div>
                  <div className="text-[10px] text-stone-500">
                    Límite: {d.limit || d.minDays} {d.minDays ? 'días' : 'eventos'} • {d.triggeredCount} activaciones
                  </div>
                </div>
                <span
                  className={`text-[10px] font-semibold px-2 py-0.5 rounded ${
                    d.enabled
                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                      : 'bg-stone-900 text-stone-500'
                  }`}
                >
                  {d.enabled ? 'ACTIVO' : 'PAUSADO'}
                </span>
              </div>
            ))}
          </div>

          <div className="p-3 rounded-xl bg-gradient-to-r from-[#200006] to-[#12070a] border border-[#6a0712]/40 text-xs text-stone-300">
            <div className="font-semibold text-[#d4af37] mb-1 flex items-center gap-1.5">
              <Radio className="w-3.5 h-3.5 animate-pulse text-red-500" />
              Estado del radar: Patrulla protocolar activa
            </div>
            Protección de miembros contra nukes, mensajes en cadena, multicuentas recientes e inyecciones de enlaces.
          </div>
        </div>

        {/* Oracle & Sovereign AI Widget */}
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-[#d4af37]" />
              <h2 className="font-cinzel font-bold text-white text-sm tracking-wide">
                ORÁCULO DIPLOMÁTICO DE LUCIFER
              </h2>
            </div>
            <button
              onClick={() => onNavigate('oracle')}
              className="text-xs text-[#d4af37] hover:underline"
            >
              Consultar →
            </button>
          </div>

          <div className="p-3.5 rounded-xl bg-[#19090e] border border-[#3b0008]/50 space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="font-semibold text-stone-200">👑 Audiencia Soberana</span>
              <span className="text-[10px] text-stone-500">/ask • Gemini 2.5 Flash</span>
            </div>
            <p className="text-xs text-stone-300 italic border-l-2 border-[#d4af37] pl-2.5 py-0.5">
              &quot;La verdadera diplomacia no radica en vencer en el debate, sino en estructurar los acuerdos de modo que todos se sientan honrados al cumplirlos.&quot;
            </p>
          </div>

          <div className="space-y-1.5 text-xs">
            <div className="text-stone-400 font-medium">Habilidades de cancillería operativas:</div>
            <div className="grid grid-cols-2 gap-2 text-[11px]">
              <div className="p-2 rounded bg-[#16080c] border border-[#3b0008]/40 text-stone-300">
                • <strong className="text-white">/ask:</strong> Audiencia formal directa
              </div>
              <div className="p-2 rounded bg-[#16080c] border border-[#3b0008]/40 text-stone-300">
                • <strong className="text-white">/roast:</strong> Brindis perspicaz
              </div>
              <div className="p-2 rounded bg-[#16080c] border border-[#3b0008]/40 text-stone-300">
                • <strong className="text-white">/piano:</strong> Composiciones solemnes
              </div>
              <div className="p-2 rounded bg-[#16080c] border border-[#3b0008]/40 text-stone-300">
                • <strong className="text-white">/consejo:</strong> Orientación estratégica
              </div>
            </div>
          </div>

          <button
            onClick={() => onNavigate('oracle')}
            className="w-full py-2 rounded-xl bg-gradient-to-r from-[#510a14] to-[#3b0008] text-white hover:brightness-110 text-xs font-semibold border border-[#d4af37]/30 transition-all"
          >
            Abrir Salón de Consultas con Lucifer
          </button>
        </div>

        {/* Sovereign Protocol & Economy Preview */}
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Coins className="w-4 h-4 text-[#d4af37]" />
              <h2 className="font-cinzel font-bold text-white text-sm tracking-wide">
                TESORERÍA &amp; HONORES
              </h2>
            </div>
            <button
              onClick={() => onNavigate('economy')}
              className="text-xs text-[#d4af37] hover:underline"
            >
              Ver Catálogo →
            </button>
          </div>

          <div className="space-y-2">
            <div className="p-3 rounded-xl bg-[#19090e] border border-[#3b0008]/40 flex items-center justify-between">
              <div>
                <div className="text-xs font-medium text-stone-200">Estipendio Diario (/fichas)</div>
                <div className="text-[10px] text-stone-400">250 monedas acreditadas cada 24 horas</div>
              </div>
              <span className="text-xs font-bold text-[#d4af37] font-cinzel">🪙 +250</span>
            </div>

            <div className="p-3 rounded-xl bg-[#19090e] border border-[#3b0008]/40 flex items-center justify-between">
              <div>
                <div className="text-xs font-medium text-stone-200">Ruleta de Salón (/ruleta)</div>
                <div className="text-[10px] text-stone-400">Rojo (2x), Negro (2x), Verde (36x)</div>
              </div>
              <span className="text-xs font-bold text-emerald-400">Hasta 36x</span>
            </div>

            <div className="p-3 rounded-xl bg-[#19090e] border border-[#3b0008]/40 flex items-center justify-between">
              <div>
                <div className="text-xs font-medium text-stone-200">Expediente Soberano (/panel)</div>
                <div className="text-[10px] text-stone-400">Rango, inventario y ajustes privados</div>
              </div>
              <button
                onClick={() => onNavigate('luxpanel')}
                className="text-xs text-[#d4af37] font-semibold hover:underline"
              >
                Consultar
              </button>
            </div>
          </div>

          <div className="pt-2 border-t border-[#3b0008]/50 flex items-center justify-between text-xs text-stone-400">
            <span>Canal oficial de recepción:</span>
            <span className="text-white font-medium">{guild.welcomeChannel}</span>
          </div>
        </div>
      </div>
    </div>
  );
};
