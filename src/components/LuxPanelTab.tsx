import React, { useState } from 'react';
import {
  Crown,
  Gift,
  Package,
  Users,
  Settings,
  Globe,
  Bell,
  Eye,
  Check,
} from 'lucide-react';
import { Member, GuildInfo } from '../types.ts';

interface LuxPanelTabProps {
  member: Member;
  guild: GuildInfo;
  onClaimDaily: () => void;
  onUpdateSetting: (key: string, value: any) => void;
  dailyClaimMessage: string | null;
  dailyError: string | null;
}

export const LuxPanelTab: React.FC<LuxPanelTabProps> = ({
  member,
  guild,
  onClaimDaily,
  onUpdateSetting,
  dailyClaimMessage,
  dailyError,
}) => {
  const [activeSection, setActiveSection] = useState<'profile' | 'inventory' | 'social' | 'settings'>('profile');

  return (
    <div className="space-y-6">
      {/* Banner */}
      <div className="rounded-2xl p-6 bg-gradient-to-r from-[#200006] via-[#35070e] to-[#120306] border border-[#d4af37]/30 shadow-xl flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Crown className="w-5 h-5 text-[#d4af37]" />
            <h2 className="text-xl font-bold font-cinzel text-white">
              EXPEDIENTE DE CIUDADANÍA • PROTOCOLO SOBERANO (/panel)
            </h2>
          </div>
          <p className="text-xs text-stone-300 font-light">
            Interfaz protocolar del dignatario: consulte su rango de honor, sus títulos en tesorería y sus preferencias de cancillería.
          </p>
        </div>

        <button
          onClick={onClaimDaily}
          className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-bold bg-gradient-to-r from-[#d4af37] to-[#b89025] text-black hover:brightness-110 active:scale-95 transition-all shadow-lg shadow-[#d4af37]/20"
        >
          <Gift className="w-4 h-4" />
          <span>Reclamar Recompensa Diaria (🪙 +250)</span>
        </button>
      </div>

      {dailyClaimMessage && (
        <div className="p-3.5 rounded-xl bg-emerald-950 border border-emerald-700/60 text-xs text-emerald-300 flex items-center gap-2">
          <Check className="w-4 h-4 text-emerald-400" />
          <span>{dailyClaimMessage}</span>
        </div>
      )}

      {dailyError && (
        <div className="p-3.5 rounded-xl bg-amber-950 border border-amber-700/60 text-xs text-amber-300">
          {dailyError}
        </div>
      )}

      {/* Discord Embed Window Style */}
      <div className="max-w-3xl mx-auto bg-[#100508] border-2 border-[#510a14] rounded-2xl p-6 shadow-2xl space-y-6">
        {/* Navigation Selector like Discord Select Menu */}
        <div className="space-y-2">
          <label className="text-xs font-bold text-stone-400 uppercase tracking-wider font-cinzel">
            ✦ Navegación del Expediente Soberano ✦
          </label>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
            <button
              onClick={() => setActiveSection('profile')}
              className={`p-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition-all ${
                activeSection === 'profile'
                  ? 'bg-[#510a14] text-white border-[#d4af37]'
                  : 'bg-[#18080d] text-stone-300 border-[#3b0008] hover:bg-[#200a12]'
              }`}
            >
              <Crown className="w-4 h-4 text-[#d4af37]" />
              <span>👤 Perfil</span>
            </button>

            <button
              onClick={() => setActiveSection('inventory')}
              className={`p-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition-all ${
                activeSection === 'inventory'
                  ? 'bg-[#510a14] text-white border-[#d4af37]'
                  : 'bg-[#18080d] text-stone-300 border-[#3b0008] hover:bg-[#200a12]'
              }`}
            >
              <Package className="w-4 h-4 text-[#d4af37]" />
              <span>🎒 Inventario</span>
            </button>

            <button
              onClick={() => setActiveSection('social')}
              className={`p-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition-all ${
                activeSection === 'social'
                  ? 'bg-[#510a14] text-white border-[#d4af37]'
                  : 'bg-[#18080d] text-stone-300 border-[#3b0008] hover:bg-[#200a12]'
              }`}
            >
              <Users className="w-4 h-4 text-[#d4af37]" />
              <span>🎭 Club Social</span>
            </button>

            <button
              onClick={() => setActiveSection('settings')}
              className={`p-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition-all ${
                activeSection === 'settings'
                  ? 'bg-[#510a14] text-white border-[#d4af37]'
                  : 'bg-[#18080d] text-stone-300 border-[#3b0008] hover:bg-[#200a12]'
              }`}
            >
              <Settings className="w-4 h-4 text-[#d4af37]" />
              <span>⚙️ Ajustes</span>
            </button>
          </div>
        </div>

        {/* Dynamic Section Display */}
        {activeSection === 'profile' && (
          <div className="bg-[#18080d] border border-[#3b0008] rounded-xl p-6 space-y-5">
            <div className="flex items-center justify-between border-b border-[#3b0008]/60 pb-4">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-full bg-gradient-to-tr from-[#6a0712] to-[#d4af37] p-0.5 shadow-lg">
                  <div className="w-full h-full rounded-full bg-[#120407] flex items-center justify-center text-lg">
                    🍸
                  </div>
                </div>
                <div>
                  <h3 className="font-cinzel font-bold text-white text-base">
                    {member.username}
                  </h3>
                  <span className="text-xs px-2 py-0.5 rounded bg-[#2e0b13] text-[#d4af37] font-semibold border border-[#d4af37]/30">
                    {member.rank}
                  </span>
                </div>
              </div>

              <div className="text-right">
                <span className="text-[11px] text-stone-400 block">Monedas Soberanas</span>
                <span className="text-xl font-cinzel font-bold text-[#d4af37]">
                  🪙 {member.balance.toLocaleString()}
                </span>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="p-3 rounded-lg bg-[#120508] border border-[#3b0008]/40">
                <div className="text-[11px] text-stone-400">Nivel de Membresía</div>
                <div className="text-lg font-bold text-white font-cinzel mt-0.5">
                  Nivel {member.level}
                </div>
              </div>

              <div className="p-3 rounded-lg bg-[#120508] border border-[#3b0008]/40">
                <div className="text-[11px] text-stone-400">Progreso para el Próximo Rango</div>
                <div className="text-lg font-bold text-stone-200 mt-0.5">
                  {member.xp} / 100 XP
                </div>
              </div>
            </div>

            {/* XP Bar */}
            <div className="space-y-1.5">
              <div className="flex justify-between text-[11px] text-stone-400">
                <span>Progreso hacia Nivel {member.level + 1}</span>
                <span>{member.xp}%</span>
              </div>
              <div className="w-full bg-[#0d0406] h-2.5 rounded-full overflow-hidden border border-[#3b0008]">
                <div
                  className="bg-gradient-to-r from-[#6a0712] to-[#d4af37] h-full rounded-full transition-all duration-300"
                  style={{ width: `${member.xp}%` }}
                ></div>
              </div>
            </div>
          </div>
        )}

        {activeSection === 'inventory' && (
          <div className="bg-[#18080d] border border-[#3b0008] rounded-xl p-6 space-y-4">
            <div className="border-b border-[#3b0008]/60 pb-3 flex items-center justify-between">
              <div>
                <h3 className="font-cinzel font-bold text-white text-sm">
                  🎒 Tu Inventario Soberano
                </h3>
                <p className="text-xs text-stone-400">
                  Títulos de honor, decretos e insignias adquiridas en la tesorería imperial.
                </p>
              </div>
              <span className="text-xs text-[#d4af37]">{member.inventory.length} artículo(s)</span>
            </div>

            {member.inventory.length === 0 ? (
              <div className="p-8 text-center text-xs text-stone-500">
                Aún no posee títulos ni objetos especiales. Visite la Tesorería Soberana para adquirir credenciales e insignias.
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {member.inventory.map((item, idx) => (
                  <div
                    key={idx}
                    className="p-3.5 rounded-xl bg-[#120508] border border-[#3b0008] flex items-center justify-between"
                  >
                    <div className="space-y-0.5">
                      <div className="text-xs font-bold text-white">{item}</div>
                      <div className="text-[10px] text-stone-400">Posees: 1 unidad</div>
                    </div>
                    <span className="text-lg">✨</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeSection === 'social' && (
          <div className="bg-[#18080d] border border-[#3b0008] rounded-xl p-6 space-y-4">
            <h3 className="font-cinzel font-bold text-white text-sm">
              ⚜️ Senado Social • Cámara de Dignatarios
            </h3>
            <p className="text-xs text-stone-300 font-light">
              Permite a otros dignatarios conocer su disponibilidad para audiencias, deliberaciones en canales solemnes o eventos de honor.
            </p>

            <div className="p-4 rounded-xl bg-[#120508] border border-[#3b0008] flex items-center justify-between">
              <div>
                <div className="text-xs font-bold text-white">Estado: Buscando Deliberación</div>
                <div className="text-[11px] text-stone-400">Visible en las listas de miembros activos</div>
              </div>
              <span className="px-2.5 py-1 rounded bg-emerald-950 text-emerald-400 text-xs font-bold border border-emerald-800">
                ACTIVO
              </span>
            </div>
          </div>
        )}

        {activeSection === 'settings' && (
          <div className="bg-[#18080d] border border-[#3b0008] rounded-xl p-6 space-y-4">
            <h3 className="font-cinzel font-bold text-white text-sm">
              ⚙️ Ajustes Personales de Protocolo
            </h3>

            <div className="space-y-3">
              <div className="p-3 rounded-xl bg-[#120508] border border-[#3b0008] flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <Globe className="w-4 h-4 text-[#d4af37]" />
                  <div>
                    <div className="text-xs font-bold text-white">Idioma del Panel</div>
                    <div className="text-[10px] text-stone-400">
                      Actual: {guild.language === 'es' ? 'Español' : 'English'}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => onUpdateSetting('language', guild.language === 'es' ? 'en' : 'es')}
                  className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-[#2e0b13] text-[#d4af37] border border-[#d4af37]/30 hover:bg-[#3d0f1a]"
                >
                  Cambiar a {guild.language === 'es' ? 'English' : 'Español'}
                </button>
              </div>

              <div className="p-3 rounded-xl bg-[#120508] border border-[#3b0008] flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <Eye className="w-4 h-4 text-[#d4af37]" />
                  <div>
                    <div className="text-xs font-bold text-white">Visibilidad de Respuestas (/ask, /panel)</div>
                    <div className="text-[10px] text-stone-400">
                      {guild.responsesPublic ? 'Públicas en el canal' : 'Efímeras (solo tú)'}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => onUpdateSetting('responsesPublic', !guild.responsesPublic)}
                  className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-[#2e0b13] text-[#d4af37] border border-[#d4af37]/30 hover:bg-[#3d0f1a]"
                >
                  {guild.responsesPublic ? 'Hacer Privadas' : 'Hacer Públicas'}
                </button>
              </div>

              <div className="p-3 rounded-xl bg-[#120508] border border-[#3b0008] flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <Bell className="w-4 h-4 text-[#d4af37]" />
                  <div>
                    <div className="text-xs font-bold text-white">Notificaciones &amp; Avisos</div>
                    <div className="text-[10px] text-stone-400">
                      {guild.notificationsEnabled ? 'Activadas' : 'Desactivadas'}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => onUpdateSetting('notificationsEnabled', !guild.notificationsEnabled)}
                  className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-[#2e0b13] text-[#d4af37] border border-[#d4af37]/30 hover:bg-[#3d0f1a]"
                >
                  {guild.notificationsEnabled ? 'Desactivar' : 'Activar'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
