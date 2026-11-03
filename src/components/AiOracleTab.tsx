import React, { useState } from 'react';
import {
  Sparkles,
  Send,
  MessageSquare,
  Flame,
  Music,
  Crown,
  Scale,
  Languages,
  CheckCircle2,
  Clock,
} from 'lucide-react';
import { AiInteraction } from '../types.ts';

interface AiOracleTabProps {
  history: AiInteraction[];
  onAskLucifer: (question: string, mode: string, targetUser?: string) => Promise<any>;
}

export const AiOracleTab: React.FC<AiOracleTabProps> = ({
  history,
  onAskLucifer,
}) => {
  const [question, setQuestion] = useState('');
  const [targetUser, setTargetUser] = useState('Dignatario_Visitante');
  const [activeMode, setActiveMode] = useState<'ask' | 'roast' | 'piano' | 'advice' | 'investigate' | 'translate'>('ask');
  const [loading, setLoading] = useState(false);
  const [progressSeconds, setProgressSeconds] = useState(0);
  const [latestResult, setLatestResult] = useState<string | null>(null);

  const modes = [
    { id: 'ask', label: '/ask • Audiencia Oficial', icon: MessageSquare, placeholder: 'Plantee su consulta formal para Lucifer...', desc: 'Audiencia formal de alta distinción' },
    { id: 'roast', label: '/roast • Brindis Perspicaz', icon: Flame, placeholder: 'Comentario noble para el distinguido miembro...', desc: 'Sarcasmo de salón de alta cuna' },
    { id: 'piano', label: '/piano • Composición', icon: Music, placeholder: 'Tema solemne (ej. la inmortalidad, el tiempo, la lealtad)...', desc: 'Versos rimados tocados al piano de cola' },
    { id: 'advice', label: '/consejo • Cancillería', icon: Crown, placeholder: 'Describa su dilema o encrucijada estratégica...', desc: 'Orientación pragmática sin rodeos' },
    { id: 'investigate', label: '/investigar • Caso', icon: Scale, placeholder: 'Antecedentes, pruebas o disputa a examinar...', desc: 'Dictamen imparcial deduciendo hechos' },
    { id: 'translate', label: '/traducir • Diplomacia', icon: Languages, placeholder: 'Texto a traducir con porte diplomático...', desc: 'Traducción de alto estándar formal' },
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!question.trim()) return;

    setLoading(true);
    setProgressSeconds(0);
    setLatestResult(null);

    const interval = setInterval(() => {
      setProgressSeconds((prev) => prev + 1);
    }, 1000);

    try {
      const res = await onAskLucifer(question.trim(), activeMode, targetUser);
      setLatestResult(res.formattedText || res.result);
      setQuestion('');
    } catch (err: any) {
      setLatestResult(`⚠️ **Aviso Protocolar:** No fue posible concluir la deliberación.`);
    } finally {
      clearInterval(interval);
      setLoading(false);
    }
  };

  const getProgressBar = (seconds: number) => {
    const total = 12;
    const filled = (seconds * 2) % (total + 1);
    return '▰'.repeat(filled) + '▱'.repeat(total - filled);
  };

  return (
    <div className="space-y-6">
      {/* Banner */}
      <div className="rounded-2xl p-6 bg-gradient-to-r from-[#200006] via-[#35070e] to-[#120306] border border-[#d4af37]/30 shadow-xl flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-[#d4af37]" />
            <h2 className="text-xl font-bold font-cinzel text-white">
              ORÁCULO DIPLOMÁTICO DE LUCIFER
            </h2>
          </div>
          <p className="text-xs text-stone-300 font-light">
            Audiencias privadas, dictámenes solemnes y composiciones al piano con elocuencia protocolar y carisma distinguido.
          </p>
        </div>

        <div className="px-3.5 py-1.5 rounded-xl bg-[#140609] border border-[#d4af37]/30 text-xs text-[#d4af37] font-semibold flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>Modelo Soberano: Gemini 2.5 Flash / Local Active</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Input & Form Box */}
        <div className="lg:col-span-7 space-y-4">
          {/* Mode Selector */}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {modes.map((m) => {
              const Icon = m.icon;
              const isActive = activeMode === m.id;
              return (
                <button
                  key={m.id}
                  type="button"
                  onClick={() => setActiveMode(m.id as any)}
                  className={`p-3 rounded-xl border text-left transition-all ${
                    isActive
                      ? 'bg-[#510a14] border-[#d4af37] text-white shadow-md'
                      : 'bg-[#12070a] border-[#3b0008] text-stone-300 hover:bg-[#180a0f]'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-[#d4af37]' : 'text-stone-400'}`} />
                    <span className="text-xs font-bold">{m.label.split(' • ')[0]}</span>
                  </div>
                  <div className="text-[10px] text-stone-400 mt-1 line-clamp-1">{m.desc}</div>
                </button>
              );
            })}
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4 shadow-lg">
            {activeMode === 'roast' && (
              <div>
                <label className="block text-xs font-medium text-stone-300 mb-1">
                  Destinatario del Brindis de Honor:
                </label>
                <input
                  type="text"
                  value={targetUser}
                  onChange={(e) => setTargetUser(e.target.value)}
                  placeholder="Nombre de usuario"
                  className="w-full px-3.5 py-2 text-xs bg-[#0b0305] border border-[#3b0008] rounded-xl text-white focus:outline-none focus:border-[#d4af37]"
                />
              </div>
            )}

            <div>
              <label className="block text-xs font-medium text-stone-300 mb-1">
                {modes.find((m) => m.id === activeMode)?.label}
              </label>
              <textarea
                rows={3}
                required
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder={modes.find((m) => m.id === activeMode)?.placeholder}
                className="w-full px-3.5 py-2.5 text-xs bg-[#0b0305] border border-[#3b0008] rounded-xl text-white focus:outline-none focus:border-[#d4af37]"
              />
            </div>

            <button
              type="submit"
              disabled={loading || !question.trim()}
              className={`w-full py-3 rounded-xl text-xs font-bold tracking-wider font-cinzel transition-all shadow-lg flex items-center justify-center gap-2 ${
                loading || !question.trim()
                  ? 'bg-stone-800 text-stone-500 cursor-not-allowed'
                  : 'bg-gradient-to-r from-[#d4af37] via-[#e5c158] to-[#b38f2a] text-black hover:brightness-110 active:scale-98'
              }`}
            >
              <Send className="w-3.5 h-3.5" />
              <span>{loading ? 'Consultando con Lucifer...' : 'Transmitir Consulta Soberana'}</span>
            </button>
          </form>

          {/* Live Progress Bar indicator */}
          {loading && (
            <div className="p-4 rounded-xl bg-[#19090e] border border-[#d4af37]/40 space-y-2 animate-pulse">
              <div className="flex items-center justify-between text-xs text-stone-200">
                <span className="font-semibold flex items-center gap-2">
                  <Clock className="w-3.5 h-3.5 text-[#d4af37]" />
                  Lucifer delibera sobre su consulta con rigor protocolar...
                </span>
                <span className="text-[#d4af37] font-mono">{progressSeconds} s</span>
              </div>
              <div className="font-mono text-xs text-[#d4af37]">
                [{getProgressBar(progressSeconds)}] ✨ Sincronizando registros diplomáticos
              </div>
            </div>
          )}

          {/* Latest Response Card (Formatted Plain Text NO EMBED) */}
          {latestResult && !loading && (
            <div className="bg-[#18080d] border border-[#d4af37]/40 rounded-2xl p-5 space-y-3 shadow-xl">
              <div className="flex items-center justify-between border-b border-[#3b0008]/50 pb-2">
                <span className="text-xs font-bold text-[#d4af37] font-cinzel flex items-center gap-1.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  RESPUESTA OFICIAL RECIBIDA
                </span>
                <span className="text-[10px] text-stone-400">Audiencia Soberana • Despacho Oficial</span>
              </div>
              <div className="text-xs text-stone-200 whitespace-pre-wrap leading-relaxed font-sans">
                {latestResult}
              </div>
            </div>
          )}
        </div>

        {/* History Stream */}
        <div className="lg:col-span-5 bg-[#12070a] border border-[#3b0008] rounded-2xl p-5 space-y-4 shadow-lg flex flex-col justify-between max-h-[620px]">
          <div className="space-y-1">
            <h3 className="font-cinzel font-bold text-white text-sm">
              REGISTRO DE AUDIENCIAS RECIENTES
            </h3>
            <p className="text-[11px] text-stone-400">
              Interacciones diplomáticas emitidas bajo el sello soberano.
            </p>
          </div>

          <div className="flex-1 overflow-y-auto space-y-3 pr-1 divide-y divide-[#3b0008]/30">
            {history.map((item) => (
              <div key={item.id} className="pt-3 space-y-1.5">
                <div className="flex items-center justify-between text-[11px]">
                  <span className="font-bold text-[#d4af37]">@{item.sender}</span>
                  <span className="text-stone-500 font-mono">{item.command}</span>
                </div>
                <p className="text-xs text-stone-400 italic">
                  &quot;{item.prompt}&quot;
                </p>
                <div className="p-2.5 rounded-lg bg-[#0e0407] border border-[#3b0008]/40 text-xs text-stone-200 leading-relaxed">
                  {item.response}
                </div>
              </div>
            ))}
          </div>

          <div className="pt-3 border-t border-[#3b0008]/50 text-center text-[10px] text-stone-500">
            Sello de Lucifer Protocol • Distinción, Diplomacia y Soberanía
          </div>
        </div>
      </div>
    </div>
  );
};
