import React from 'react';
import {
  ShieldAlert,
  Lock,
  Unlock,
  Radio,
  Clock,
  UserCheck,
  Link,
  MessageSquare,
  Bomb,
  Flame,
  CheckCircle2,
} from 'lucide-react';
import { AntiRaidState } from '../types.ts';

interface AntiRaidTabProps {
  antiraid: AntiRaidState;
  onToggleLockdown: () => void;
  onToggleDetector: (key: string, enabled: boolean) => void;
}

export const AntiRaidTab: React.FC<AntiRaidTabProps> = ({
  antiraid,
  onToggleLockdown,
  onToggleDetector,
}) => {
  const getDetectorIcon = (key: string) => {
    switch (key) {
      case 'joinFlood':
        return UserCheck;
      case 'messageSpam':
        return MessageSquare;
      case 'mentionSpam':
        return Flame;
      case 'linkScan':
        return Link;
      case 'accountAge':
        return Clock;
      case 'nukeProtection':
        return Bomb;
      default:
        return ShieldAlert;
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Threat & Lockdown Banner */}
      <div
        className={`rounded-2xl p-6 border transition-all ${
          antiraid.lockdownActive
            ? 'bg-gradient-to-r from-red-950 via-[#3b0008] to-black border-red-600 shadow-2xl shadow-red-950/50'
            : 'bg-gradient-to-r from-[#170509] via-[#24080e] to-[#120407] border-[#3b0008] shadow-xl'
        }`}
      >
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span
                className={`px-2 py-0.5 rounded text-[11px] font-bold uppercase tracking-wider ${
                  antiraid.lockdownActive
                    ? 'bg-red-900 text-white animate-pulse'
                    : 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                }`}
              >
                {antiraid.lockdownActive ? '🚨 PROTOCOLO DE PÁNICO ACTIVO' : '🛡️ ESCUDO ACTIVO'}
              </span>
              <span className="text-xs text-stone-400">
                Nivel de Amenaza: <strong className="text-white">{antiraid.liveMetrics.threatLevel}</strong>
              </span>
            </div>
            <h2 className="text-2xl font-bold font-cinzel text-white">
              {antiraid.lockdownActive ? 'Servidor Bloqueado por Emergencia' : 'Protección de Entrada & Moderación Automática'}
            </h2>
            <p className="text-xs sm:text-sm text-stone-300 max-w-2xl font-light">
              {antiraid.lockdownActive
                ? `Los canales de texto principales están bloqueados para roles estándar hasta las ${
                    antiraid.lockdownUntil ? new Date(antiraid.lockdownUntil).toLocaleTimeString() : 'fin del temporizador'
                  }. Solo el staff con permisos puede comunicarse.`
                : 'El motor antiraid vigila constantemente las conexiones entrantes, ráfagas de mensajes, menciones masivas y patrones de demolición (nuke).'}
            </p>
          </div>

          <button
            onClick={onToggleLockdown}
            className={`flex items-center justify-center gap-2 px-6 py-3.5 rounded-xl text-sm font-bold tracking-wide transition-all active:scale-95 shadow-xl ${
              antiraid.lockdownActive
                ? 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-emerald-900/40'
                : 'bg-red-800 hover:bg-red-700 text-white shadow-red-950/60'
            }`}
          >
            {antiraid.lockdownActive ? (
              <>
                <Unlock className="w-5 h-5" />
                <span>Desactivar Bloqueo y Restaurar Permisos</span>
              </>
            ) : (
              <>
                <Lock className="w-5 h-5" />
                <span>Activar Bloqueo de Pánico (5m)</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Real-time radar metrics */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="bg-[#12070a] border border-[#3b0008] p-3.5 rounded-xl">
          <div className="text-[11px] text-stone-400 uppercase tracking-wider">Ingresos últimos 10s</div>
          <div className="text-xl font-bold text-white font-cinzel mt-1">
            {antiraid.liveMetrics.joinsLastTenSeconds} miembros
          </div>
          <div className="text-[10px] text-emerald-400 mt-1">Umbral seguro: &lt; 5</div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-3.5 rounded-xl">
          <div className="text-[11px] text-stone-400 uppercase tracking-wider">Canales Eliminados (1m)</div>
          <div className="text-xl font-bold text-white font-cinzel mt-1">
            {antiraid.liveMetrics.channelDeletesLastMin}
          </div>
          <div className="text-[10px] text-emerald-400 mt-1">Umbral nuke: 3 en 30s</div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-3.5 rounded-xl">
          <div className="text-[11px] text-stone-400 uppercase tracking-wider">Roles Eliminados (1m)</div>
          <div className="text-xl font-bold text-white font-cinzel mt-1">
            {antiraid.liveMetrics.roleDeletesLastMin}
          </div>
          <div className="text-[10px] text-emerald-400 mt-1">Umbral nuke: 3 en 30s</div>
        </div>

        <div className="bg-[#12070a] border border-[#3b0008] p-3.5 rounded-xl">
          <div className="text-[11px] text-stone-400 uppercase tracking-wider">Velocidad de Mensajes</div>
          <div className="text-xl font-bold text-white font-cinzel mt-1">
            {antiraid.liveMetrics.messagesPerSec} msg/s
          </div>
          <div className="text-[10px] text-emerald-400 mt-1">Flujo normal en el Lux</div>
        </div>
      </div>

      {/* Detectors Matrix */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Radio className="w-4 h-4 text-[#d4af37]" />
            <h3 className="font-cinzel font-bold text-white tracking-wide">
              MÓDULOS DETECTORES &amp; FILTROS
            </h3>
          </div>
          <span className="text-xs text-stone-400">
            Haz clic en el interruptor para pausar o activar cada detector
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Object.entries(antiraid.detectors).map(([key, detector]) => {
            const Icon = getDetectorIcon(key);
            return (
              <div
                key={key}
                className={`p-4 rounded-xl border transition-all ${
                  detector.enabled
                    ? 'bg-[#15070b] border-[#3b0008] hover:border-[#6a0712]'
                    : 'bg-[#0f0507] border-stone-800 opacity-60'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2.5">
                    <div className="p-2 rounded-lg bg-[#250910] border border-[#d4af37]/20 text-[#d4af37]">
                      <Icon className="w-4 h-4" />
                    </div>
                    <div>
                      <h4 className="text-xs font-bold text-white">{detector.name}</h4>
                      <p className="text-[11px] text-stone-400">
                        {detector.minDays
                          ? `Cuentas con antigüedad menor a ${detector.minDays} días`
                          : `Límite: ${detector.limit} eventos en ventana de ${detector.windowSec}s`}
                      </p>
                    </div>
                  </div>

                  <button
                    onClick={() => onToggleDetector(key, !detector.enabled)}
                    className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                      detector.enabled ? 'bg-[#8a1322]' : 'bg-stone-800'
                    }`}
                  >
                    <span
                      className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                        detector.enabled ? 'translate-x-4' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>

                <div className="mt-3 pt-2.5 border-t border-[#3b0008]/40 flex items-center justify-between text-[11px] text-stone-400">
                  <span>Activaciones registradas:</span>
                  <span className="font-bold text-[#d4af37]">{detector.triggeredCount} veces</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Incident Log */}
      <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldAlert className="w-4 h-4 text-rose-500" />
            <h3 className="font-cinzel font-bold text-white tracking-wide">
              REGISTRO DE INCIDENTES RECIENTES
            </h3>
          </div>
          <span className="text-xs text-stone-400">Historial de neutralizaciones</span>
        </div>

        <div className="space-y-2.5">
          {antiraid.incidents.map((incident) => (
            <div
              key={incident.id}
              className="p-3.5 rounded-xl bg-[#19090e] border border-[#3b0008]/60 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="font-bold text-white">{incident.type}</span>
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-[#2c0b13] text-[#d4af37] border border-[#d4af37]/30">
                    Canal: {incident.target}
                  </span>
                  <span
                    className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${
                      incident.severity === 'Crítica'
                        ? 'bg-red-950 text-red-400 border border-red-700'
                        : 'bg-amber-950 text-amber-400 border border-amber-700'
                    }`}
                  >
                    {incident.severity}
                  </span>
                </div>
                <p className="text-stone-300 font-light">{incident.details}</p>
              </div>

              <div className="flex sm:flex-col sm:items-end justify-between text-[11px] text-stone-400 whitespace-nowrap">
                <span className="text-emerald-400 font-medium flex items-center gap-1">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  {incident.action}
                </span>
                <span className="text-stone-500">
                  {new Date(incident.mitigatedAt).toLocaleTimeString()}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
