import React, { useState } from 'react';
import {
  Flame,
  UserX,
  VolumeX,
  Scale,
  Check,
  X,
  PlusCircle,
  Search,
} from 'lucide-react';
import { Ban, Timeout, Appeal } from '../types.ts';

interface SecurityCenterTabProps {
  bans: Ban[];
  timeouts: Timeout[];
  appeals: Appeal[];
  onApplySanction: (action: string, targetUsername: string, reason: string) => void;
  onResolveAppeal: (appealId: number, status: 'accepted' | 'denied') => void;
}

export const SecurityCenterTab: React.FC<SecurityCenterTabProps> = ({
  bans,
  timeouts,
  appeals,
  onApplySanction,
  onResolveAppeal,
}) => {
  const [activeSubTab, setActiveSubTab] = useState<'bans' | 'timeouts' | 'appeals'>('bans');
  const [showSanctionModal, setShowSanctionModal] = useState(false);
  const [targetUsername, setTargetUsername] = useState('');
  const [sanctionAction, setSanctionAction] = useState('timeout');
  const [sanctionReason, setSanctionReason] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  const handleCreateSanction = (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetUsername.trim()) return;
    onApplySanction(sanctionAction, targetUsername.trim(), sanctionReason.trim() || 'Quebranto del protocolo soberano de cancillería');
    setTargetUsername('');
    setSanctionReason('');
    setShowSanctionModal(false);
  };

  const filteredBans = bans.filter(
    (b) =>
      b.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      b.reason.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const filteredTimeouts = timeouts.filter(
    (t) =>
      t.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.reason.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="rounded-2xl p-6 bg-gradient-to-r from-[#200006] via-[#35070e] to-[#120306] border border-[#d4af37]/30 shadow-xl flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Flame className="w-5 h-5 text-[#d4af37]" />
            <h2 className="text-xl font-bold font-cinzel text-white">
              TRIBUNAL DE HONOR • CONSOLA DE ALTA SEGURIDAD
            </h2>
          </div>
          <p className="text-xs text-stone-300 font-light">
            Tribunal Soberano: revisión de sanciones solemnes, apelaciones de miembros y edictos de cancillería.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowSanctionModal(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold bg-[#6a0712] hover:bg-[#850a18] text-white border border-[#d4af37]/40 shadow-md transition-all active:scale-95"
          >
            <PlusCircle className="w-4 h-4" />
            <span>Emitir Sanción Manual</span>
          </button>
        </div>
      </div>

      {/* Sub-tabs & Search Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center space-x-1 p-1 bg-[#12070a] rounded-xl border border-[#3b0008]">
          <button
            onClick={() => setActiveSubTab('bans')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              activeSubTab === 'bans'
                ? 'bg-[#510a14] text-white border border-[#d4af37]/40'
                : 'text-stone-400 hover:text-white'
            }`}
          >
            <UserX className="w-3.5 h-3.5" />
            <span>Baneos Activos ({bans.length})</span>
          </button>

          <button
            onClick={() => setActiveSubTab('timeouts')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              activeSubTab === 'timeouts'
                ? 'bg-[#510a14] text-white border border-[#d4af37]/40'
                : 'text-stone-400 hover:text-white'
            }`}
          >
            <VolumeX className="w-3.5 h-3.5" />
            <span>Silencios ({timeouts.length})</span>
          </button>

          <button
            onClick={() => setActiveSubTab('appeals')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              activeSubTab === 'appeals'
                ? 'bg-[#510a14] text-white border border-[#d4af37]/40'
                : 'text-stone-400 hover:text-white'
            }`}
          >
            <Scale className="w-3.5 h-3.5" />
            <span>Apelaciones ({appeals.filter((a) => a.status === 'pending').length})</span>
          </button>
        </div>

        <div className="relative">
          <Search className="w-4 h-4 text-stone-500 absolute left-3 top-2.5" />
          <input
            type="text"
            placeholder="Buscar por usuario o motivo..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9 pr-3 py-1.5 text-xs bg-[#12070a] border border-[#3b0008] rounded-xl text-stone-200 placeholder-stone-500 focus:outline-none focus:border-[#d4af37]/60 w-full sm:w-64"
          />
        </div>
      </div>

      {/* Subtab Content */}
      {activeSubTab === 'bans' && (
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl overflow-hidden shadow-lg">
          <div className="p-4 border-b border-[#3b0008]/50 flex items-center justify-between">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider font-cinzel">
              LISTA DE EXPULSIONES • TRIBUNAL DE CANCILLERÍA
            </h3>
            <span className="text-xs text-stone-400">Total: {filteredBans.length} registros</span>
          </div>

          <div className="divide-y divide-[#3b0008]/40">
            {filteredBans.length === 0 ? (
              <div className="p-8 text-center text-xs text-stone-500">
                No hay expulsiones activas registradas. El senado imperial permanece en orden.
              </div>
            ) : (
              filteredBans.map((ban) => (
                <div key={ban.userId} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-[#16080c]/60 transition-colors">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-white text-sm">@{ban.username}</span>
                      <span className="text-[10px] text-stone-500 font-mono">ID: {ban.userId}</span>
                    </div>
                    <p className="text-xs text-stone-300 font-light">
                      Motivo: <span className="text-red-300">{ban.reason}</span>
                    </p>
                    <div className="text-[11px] text-stone-500 flex items-center gap-3">
                      <span>Moderador: {ban.moderator}</span>
                      <span>•</span>
                      <span>Fecha: {new Date(ban.bannedAt).toLocaleDateString()}</span>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => onApplySanction('unban', ban.username, 'Revocación manual')}
                      className="px-3 py-1.5 rounded-lg text-xs font-medium bg-[#1d0a0f] hover:bg-[#2e0f17] text-[#d4af37] border border-[#d4af37]/30 transition-all"
                    >
                      Revocar Baneo
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {activeSubTab === 'timeouts' && (
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl overflow-hidden shadow-lg">
          <div className="p-4 border-b border-[#3b0008]/50 flex items-center justify-between">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider font-cinzel">
              SILENCIOS TEMPORALES ACTIVOS
            </h3>
            <span className="text-xs text-stone-400">Total: {filteredTimeouts.length}</span>
          </div>

          <div className="divide-y divide-[#3b0008]/40">
            {filteredTimeouts.length === 0 ? (
              <div className="p-8 text-center text-xs text-stone-500">
                No hay miembros silenciados en este momento.
              </div>
            ) : (
              filteredTimeouts.map((timeout) => (
                <div key={timeout.userId} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-[#16080c]/60 transition-colors">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-white text-sm">@{timeout.username}</span>
                      <span className="text-[10px] px-2 py-0.5 rounded bg-amber-950 text-amber-300 border border-amber-800">
                        Hasta {new Date(timeout.until).toLocaleTimeString()}
                      </span>
                    </div>
                    <p className="text-xs text-stone-300 font-light">
                      Motivo: <span className="text-stone-200">{timeout.reason}</span>
                    </p>
                    <div className="text-[11px] text-stone-500">
                      Moderador: {timeout.moderator}
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => onApplySanction('untimeout', timeout.username, 'Perdón de silencio')}
                      className="px-3 py-1.5 rounded-lg text-xs font-medium bg-[#1d0a0f] hover:bg-[#2e0f17] text-stone-300 border border-stone-700 transition-all"
                    >
                      Quitar Silencio
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {activeSubTab === 'appeals' && (
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl overflow-hidden shadow-lg">
          <div className="p-4 border-b border-[#3b0008]/50 flex items-center justify-between">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider font-cinzel">
              SOLICITUDES DE APELACIÓN
            </h3>
            <span className="text-xs text-stone-400">
              Aceptar retira automáticamente el castigo
            </span>
          </div>

          <div className="divide-y divide-[#3b0008]/40">
            {appeals.map((appeal) => (
              <div key={appeal.id} className="p-4 flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                <div className="space-y-1.5 max-w-xl">
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-white text-sm">#{appeal.id} • @{appeal.username}</span>
                    <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-[#250910] text-[#d4af37] border border-[#d4af37]/20">
                      Castigo: {appeal.sanction}
                    </span>
                    <span
                      className={`text-[10px] font-semibold px-2 py-0.5 rounded ${
                        appeal.status === 'pending'
                          ? 'bg-amber-950 text-amber-300 border border-amber-800'
                          : appeal.status === 'accepted'
                          ? 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                          : 'bg-red-950 text-red-300 border border-red-800'
                      }`}
                    >
                      {appeal.status.toUpperCase()}
                    </span>
                  </div>

                  <p className="text-xs text-stone-300 italic bg-[#17080d] p-2.5 rounded-lg border border-[#3b0008]/40">
                    &quot;{appeal.reason}&quot;
                  </p>
                  <div className="text-[10px] text-stone-500">
                    Enviada el: {new Date(appeal.createdAt).toLocaleString()}
                  </div>
                </div>

                {appeal.status === 'pending' && (
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => onResolveAppeal(appeal.id, 'accepted')}
                      className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold bg-emerald-800 hover:bg-emerald-700 text-white transition-all shadow-sm"
                    >
                      <Check className="w-3.5 h-3.5" />
                      <span>Aceptar &amp; Perdonar</span>
                    </button>

                    <button
                      onClick={() => onResolveAppeal(appeal.id, 'denied')}
                      className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold bg-red-900 hover:bg-red-800 text-white transition-all shadow-sm"
                    >
                      <X className="w-3.5 h-3.5" />
                      <span>Denegar</span>
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Manual Sanction Modal */}
      {showSanctionModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm">
          <div className="bg-[#140609] border border-[#d4af37]/40 rounded-2xl w-full max-w-md p-6 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between pb-3 border-b border-[#3b0008]">
              <div className="flex items-center gap-2">
                <Flame className="w-5 h-5 text-[#d4af37]" />
                <h3 className="font-cinzel font-bold text-white text-base">
                  EMITIR SANCIÓN
                </h3>
              </div>
              <button
                onClick={() => setShowSanctionModal(false)}
                className="text-stone-400 hover:text-white"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleCreateSanction} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-stone-300 mb-1">
                  Nombre de usuario o Tag
                </label>
                <input
                  type="text"
                  required
                  placeholder="ej. troll_99"
                  value={targetUsername}
                  onChange={(e) => setTargetUsername(e.target.value)}
                  className="w-full px-3 py-2 text-xs bg-[#0b0305] border border-[#3b0008] rounded-xl text-white focus:outline-none focus:border-[#d4af37]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-stone-300 mb-1">
                  Tipo de Sanción
                </label>
                <select
                  value={sanctionAction}
                  onChange={(e) => setSanctionAction(e.target.value)}
                  className="w-full px-3 py-2 text-xs bg-[#0b0305] border border-[#3b0008] rounded-xl text-white focus:outline-none focus:border-[#d4af37]"
                >
                  <option value="timeout">Silencio temporal (1 hora)</option>
                  <option value="ban">Baneo permanente</option>
                  <option value="kick">Expulsión de la corte</option>
                  <option value="warn">Aviso formal</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-stone-300 mb-1">
                  Motivo de la Sanción
                </label>
                <textarea
                  rows={3}
                  required
                  placeholder="Detalla la regla quebrantada..."
                  value={sanctionReason}
                  onChange={(e) => setSanctionReason(e.target.value)}
                  className="w-full px-3 py-2 text-xs bg-[#0b0305] border border-[#3b0008] rounded-xl text-white focus:outline-none focus:border-[#d4af37]"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2 border-t border-[#3b0008]/50">
                <button
                  type="button"
                  onClick={() => setShowSanctionModal(false)}
                  className="px-4 py-2 rounded-xl text-xs text-stone-400 hover:text-white"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-xl text-xs font-bold bg-[#6a0712] hover:bg-[#850a18] text-white border border-[#d4af37]/30 shadow-md"
                >
                  Decretar Sanción
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
