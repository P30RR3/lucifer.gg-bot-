import React from 'react';
import {
  Scroll,
  Shield,
  Download,
  Copy,
  Check,
  Sparkles,
} from 'lucide-react';
import { AuditLog, GuildInfo } from '../types.ts';

interface AuditTabProps {
  logs: AuditLog[];
  guild: GuildInfo;
}

export const AuditTab: React.FC<AuditTabProps> = ({ logs, guild }) => {
  const [copied, setCopied] = React.useState(false);

  const handleCopySnapshot = () => {
    const backupData = {
      guild: guild.name,
      id: guild.id,
      exportedAt: new Date().toISOString(),
      channels: guild.channelsCount,
      roles: guild.rolesCount,
      welcomeConfig: {
        channel: guild.welcomeChannel,
        message: guild.welcomeMessage,
        imageUrl: guild.welcomeImageUrl,
      },
      auditCount: logs.length,
    };
    navigator.clipboard.writeText(JSON.stringify(backupData, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  return (
    <div className="space-y-6">
      {/* Banner */}
      <div className="rounded-2xl p-6 bg-gradient-to-r from-[#200006] via-[#35070e] to-[#120306] border border-[#d4af37]/30 shadow-xl flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Scroll className="w-5 h-5 text-[#d4af37]" />
            <h2 className="text-xl font-bold font-cinzel text-white">
              LIBRO MAYOR DE AUDITORÍA &amp; COPIAS DE SEGURIDAD
            </h2>
          </div>
          <p className="text-xs text-stone-300 font-light">
            Registro inmutable de decretos, resoluciones de cancillería y preservación de estados del senado.
          </p>
        </div>

        <button
          onClick={handleCopySnapshot}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-bold bg-[#2e0b13] hover:bg-[#3d0f1a] text-[#d4af37] border border-[#d4af37]/40 shadow-md transition-all active:scale-95"
        >
          {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
          <span>{copied ? 'Copia Copiada al Portapapeles' : 'Exportar Respaldo JSON'}</span>
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Audit Log Stream */}
        <div className="lg:col-span-7 bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4 shadow-lg">
          <div className="flex items-center justify-between border-b border-[#3b0008]/50 pb-3">
            <h3 className="font-cinzel font-bold text-white text-xs uppercase tracking-wider">
              REGISTRO DE ACCIONES DE STAFF
            </h3>
            <span className="text-xs text-stone-400">Total: {logs.length} eventos</span>
          </div>

          <div className="space-y-2.5 max-h-[500px] overflow-y-auto pr-1">
            {logs.map((log) => (
              <div
                key={log.id}
                className="p-3.5 rounded-xl bg-[#17070c] border border-[#3b0008]/40 space-y-1 text-xs"
              >
                <div className="flex items-center justify-between">
                  <span className="font-bold text-white">{log.action}</span>
                  <span className="text-[10px] text-stone-500">
                    {new Date(log.timestamp).toLocaleTimeString()}
                  </span>
                </div>
                <p className="text-stone-300 font-light">{log.details}</p>
                <div className="text-[10px] text-[#d4af37] pt-1">
                  Responsable: {log.actor}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Welcome Message Protocol Card Preview */}
        <div className="lg:col-span-5 space-y-4">
          <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4 shadow-lg">
            <div className="flex items-center justify-between border-b border-[#3b0008]/50 pb-3">
              <h3 className="font-cinzel font-bold text-white text-xs uppercase tracking-wider">
                VISTA PREVIA DE BIENVENIDA OFICIAL
              </h3>
              <span className="text-[11px] text-[#d4af37] font-semibold">{guild.welcomeChannel}</span>
            </div>

            <div className="border border-[#6a0712] rounded-xl overflow-hidden bg-[#18080d] shadow-xl">
              <div
                className="h-28 bg-cover bg-center relative"
                style={{ backgroundImage: `url(${guild.welcomeImageUrl})` }}
              >
                <div className="absolute inset-0 bg-gradient-to-t from-[#18080d] via-black/40 to-transparent"></div>
              </div>

              <div className="p-4 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-lg">⚜️</span>
                  <h4 className="font-cinzel font-bold text-white text-sm">
                    Recepción de Honor al Senado
                  </h4>
                </div>
                <p className="text-xs text-stone-300 leading-relaxed font-light">
                  {guild.welcomeMessage}
                </p>
                <div className="pt-2 border-t border-[#3b0008]/60 text-[10px] text-stone-400">
                  Lucifer Protocol • Protocolo de Distinción Absoluta 🥂
                </div>
              </div>
            </div>

            <div className="p-3 rounded-xl bg-[#15070a] border border-[#3b0008]/40 text-xs text-stone-400 space-y-1">
              <div className="font-bold text-stone-200">Parámetros Activos:</div>
              <div>• Idioma predeterminado: {guild.language === 'es' ? 'Español' : 'English'}</div>
              <div>• Respuestas públicas: {guild.responsesPublic ? 'Activadas' : 'Desactivadas (solo tú)'}</div>
              <div>• Avisos de cancillería: {guild.notificationsEnabled ? 'Habilitados' : 'Pausados'}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
