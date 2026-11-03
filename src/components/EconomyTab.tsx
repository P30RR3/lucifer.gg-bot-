import React, { useState } from 'react';
import {
  Coins,
  ShoppingBag,
  Dices,
  Trophy,
  CheckCircle2,
  AlertCircle,
  Sparkles,
} from 'lucide-react';
import { ShopItem, Member } from '../types.ts';

interface EconomyTabProps {
  items: ShopItem[];
  members: Member[];
  currentMember: Member;
  onPlayRoulette: (bet: number, color: string) => Promise<any>;
  onBuyItem: (itemId: string) => Promise<any>;
}

export const EconomyTab: React.FC<EconomyTabProps> = ({
  items,
  members,
  currentMember,
  onPlayRoulette,
  onBuyItem,
}) => {
  const [activeSection, setActiveSection] = useState<'roulette' | 'shop' | 'leaderboard'>('roulette');
  const [betAmount, setBetAmount] = useState<number>(100);
  const [selectedColor, setSelectedColor] = useState<string>('rojo');
  const [spinning, setSpinning] = useState<boolean>(false);
  const [rouletteResult, setRouletteResult] = useState<any>(null);
  const [shopStatus, setShopStatus] = useState<string | null>(null);

  const handleSpin = async () => {
    if (betAmount < 1 || betAmount > currentMember.balance) return;
    setSpinning(true);
    setRouletteResult(null);

    try {
      const res = await onPlayRoulette(betAmount, selectedColor);
      setTimeout(() => {
        setRouletteResult(res);
        setSpinning(false);
      }, 1000);
    } catch (err: any) {
      setRouletteResult({ error: err.message || 'Error jugando a la ruleta' });
      setSpinning(false);
    }
  };

  const handleBuy = async (item: ShopItem) => {
    if (currentMember.balance < item.price) {
      setShopStatus(`💸 Saldo insuficiente para ${item.name}`);
      return;
    }
    try {
      await onBuyItem(item.id);
      setShopStatus(`✨ ¡Has adquirido '${item.name}' con éxito!`);
      setTimeout(() => setShopStatus(null), 3500);
    } catch (err: any) {
      setShopStatus(`Error en compra: ${err.message}`);
    }
  };

  return (
    <div className="space-y-6">
      {/* Banner */}
      <div className="rounded-2xl p-6 bg-gradient-to-r from-[#250005] via-[#3b0008] to-[#120306] border border-[#d4af37]/30 shadow-xl flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Coins className="w-5 h-5 text-[#d4af37]" />
            <h2 className="text-xl font-bold font-cinzel text-white">
              TESORERÍA SOBERANA • SALÓN DE JUEGOS &amp; TIENDA
            </h2>
          </div>
          <p className="text-xs text-stone-300 font-light">
            Monedas de alta reserva, ruleta de salón y catálogo exclusivo de títulos e insignias imperiales.
          </p>
        </div>

        <div className="flex items-center gap-3 bg-[#140609] px-4 py-2.5 rounded-xl border border-[#d4af37]/40 shadow-inner">
          <span className="text-xs text-stone-400">Tu Saldo Actual:</span>
          <span className="font-cinzel font-bold text-base text-[#d4af37]">
            🪙 {currentMember.balance.toLocaleString()} fichas
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center space-x-2 border-b border-[#3b0008] pb-3">
        <button
          onClick={() => setActiveSection('roulette')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeSection === 'roulette'
              ? 'bg-[#510a14] text-white border border-[#d4af37]/50 shadow-md'
              : 'text-stone-400 hover:text-white'
          }`}
        >
          <Dices className="w-4 h-4 text-[#d4af37]" />
          <span>Ruleta Soberana (/ruleta)</span>
        </button>

        <button
          onClick={() => setActiveSection('shop')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeSection === 'shop'
              ? 'bg-[#510a14] text-white border border-[#d4af37]/50 shadow-md'
              : 'text-stone-400 hover:text-white'
          }`}
        >
          <ShoppingBag className="w-4 h-4 text-[#d4af37]" />
          <span>Tienda de Coleccionables (/tienda)</span>
        </button>

        <button
          onClick={() => setActiveSection('leaderboard')}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeSection === 'leaderboard'
              ? 'bg-[#510a14] text-white border border-[#d4af37]/50 shadow-md'
              : 'text-stone-400 hover:text-white'
          }`}
        >
          <Trophy className="w-4 h-4 text-[#d4af37]" />
          <span>Clasificación del Club</span>
        </button>
      </div>

      {/* Roulette Section */}
      {activeSection === 'roulette' && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <div className="lg:col-span-7 bg-[#12070a] border border-[#3b0008] rounded-2xl p-6 space-y-6">
            <div className="flex items-center justify-between border-b border-[#3b0008]/50 pb-3">
              <div>
                <h3 className="font-cinzel font-bold text-white text-base">
                  MESA DE RULETA FRANCESA
                </h3>
                <p className="text-xs text-stone-400">
                  Elige color y fichas. La suerte se sirve con una copa de Borgoña.
                </p>
              </div>
              <span className="text-xs text-[#d4af37] font-semibold">Casa: 0 (Verde)</span>
            </div>

            {/* Color Selector */}
            <div className="space-y-2">
              <label className="text-xs font-medium text-stone-300">Selecciona el color:</label>
              <div className="grid grid-cols-3 gap-3">
                <button
                  type="button"
                  onClick={() => setSelectedColor('rojo')}
                  className={`p-4 rounded-xl border flex flex-col items-center justify-center gap-1 transition-all ${
                    selectedColor === 'rojo'
                      ? 'bg-red-950 border-red-500 shadow-lg shadow-red-950/60 ring-2 ring-red-500'
                      : 'bg-[#18070b] border-[#3b0008] hover:border-red-900'
                  }`}
                >
                  <span className="w-6 h-6 rounded-full bg-red-600 block shadow"></span>
                  <span className="font-bold text-xs text-white mt-1">ROJO</span>
                  <span className="text-[10px] text-red-300">Paga 2x</span>
                </button>

                <button
                  type="button"
                  onClick={() => setSelectedColor('negro')}
                  className={`p-4 rounded-xl border flex flex-col items-center justify-center gap-1 transition-all ${
                    selectedColor === 'negro'
                      ? 'bg-neutral-900 border-neutral-400 shadow-lg shadow-black/80 ring-2 ring-neutral-400'
                      : 'bg-[#140609] border-[#3b0008] hover:border-neutral-700'
                  }`}
                >
                  <span className="w-6 h-6 rounded-full bg-neutral-900 border border-neutral-600 block shadow"></span>
                  <span className="font-bold text-xs text-white mt-1">NEGRO</span>
                  <span className="text-[10px] text-neutral-300">Paga 2x</span>
                </button>

                <button
                  type="button"
                  onClick={() => setSelectedColor('verde')}
                  className={`p-4 rounded-xl border flex flex-col items-center justify-center gap-1 transition-all ${
                    selectedColor === 'verde'
                      ? 'bg-emerald-950 border-emerald-500 shadow-lg shadow-emerald-950/60 ring-2 ring-emerald-500'
                      : 'bg-[#0e1810] border-[#14301d] hover:border-emerald-800'
                  }`}
                >
                  <span className="w-6 h-6 rounded-full bg-emerald-600 block shadow"></span>
                  <span className="font-bold text-xs text-emerald-300 mt-1">VERDE (0)</span>
                  <span className="text-[10px] text-emerald-400">Paga 36x</span>
                </button>
              </div>
            </div>

            {/* Bet Input */}
            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs text-stone-300">
                <span>Cantidad a apostar:</span>
                <span className="text-[#d4af37]">Máx: {currentMember.balance}</span>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  min={10}
                  max={currentMember.balance}
                  value={betAmount}
                  onChange={(e) => setBetAmount(parseInt(e.target.value, 10) || 0)}
                  className="flex-1 px-4 py-2.5 bg-[#0b0305] border border-[#3b0008] rounded-xl text-white font-mono text-sm focus:outline-none focus:border-[#d4af37]"
                />
                <button
                  type="button"
                  onClick={() => setBetAmount(100)}
                  className="px-2.5 py-2 text-xs bg-[#1f0a10] border border-[#3b0008] rounded-lg text-stone-300 hover:text-white"
                >
                  100
                </button>
                <button
                  type="button"
                  onClick={() => setBetAmount(500)}
                  className="px-2.5 py-2 text-xs bg-[#1f0a10] border border-[#3b0008] rounded-lg text-stone-300 hover:text-white"
                >
                  500
                </button>
                <button
                  type="button"
                  onClick={() => setBetAmount(Math.min(2000, currentMember.balance))}
                  className="px-2.5 py-2 text-xs bg-[#1f0a10] border border-[#3b0008] rounded-lg text-stone-300 hover:text-white"
                >
                  2K
                </button>
              </div>
            </div>

            {/* Spin Button */}
            <button
              onClick={handleSpin}
              disabled={spinning || betAmount <= 0 || betAmount > currentMember.balance}
              className={`w-full py-3.5 rounded-xl text-sm font-bold tracking-wider font-cinzel transition-all shadow-xl active:scale-98 ${
                spinning
                  ? 'bg-stone-800 text-stone-500 cursor-not-allowed'
                  : 'bg-gradient-to-r from-[#d4af37] via-[#f3d26b] to-[#b38f2a] text-black hover:brightness-110'
              }`}
            >
              {spinning ? 'Girando la Ruleta del Diablo...' : `Girar la Ruleta con ${betAmount} Fichas`}
            </button>
          </div>

          {/* Outcome & Animation Box */}
          <div className="lg:col-span-5 bg-[#12070a] border border-[#3b0008] rounded-2xl p-6 flex flex-col items-center justify-center text-center space-y-4">
            <div className="w-28 h-28 rounded-full bg-gradient-to-br from-[#200006] via-[#3b0008] to-black border-4 border-[#d4af37]/60 flex items-center justify-center shadow-2xl relative">
              {spinning ? (
                <div className="w-20 h-20 rounded-full border-4 border-dashed border-[#d4af37] animate-spin"></div>
              ) : rouletteResult ? (
                <div className="text-center">
                  <span className="text-3xl font-bold font-cinzel text-white block">
                    {rouletteResult.number}
                  </span>
                  <span className="text-[10px] uppercase font-bold text-[#d4af37]">
                    {rouletteResult.color}
                  </span>
                </div>
              ) : (
                <span className="text-3xl">🎰</span>
              )}
            </div>

            <div className="space-y-1 max-w-xs">
              <h4 className="font-cinzel font-bold text-white text-sm">
                {spinning ? 'La bola está en movimiento...' : rouletteResult ? (rouletteResult.win ? '¡Victoria en la Corte!' : 'La Cancillería Gana') : 'Esperando su Apuesta'}
              </h4>
              <p className="text-xs text-stone-300">
                {rouletteResult ? rouletteResult.message : 'Pruebe su temple ante la corte. La fortuna favorece a los dignatarios decididos.'}
              </p>
            </div>

            {rouletteResult && rouletteResult.win && (
              <div className="p-3 rounded-xl bg-emerald-950/80 border border-emerald-700 text-xs text-emerald-300 font-bold flex items-center gap-1.5 animate-bounce">
                <Sparkles className="w-4 h-4" />
                <span>+🪙 {rouletteResult.payout.toLocaleString()} fichas acreditadas</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Shop Section */}
      {activeSection === 'shop' && (
        <div className="space-y-4">
          {shopStatus && (
            <div className="p-3.5 rounded-xl bg-[#2e0b13] border border-[#d4af37]/40 text-xs text-[#d4af37] font-semibold flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4" />
              <span>{shopStatus}</span>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {items.map((item) => (
              <div
                key={item.id}
                className="bg-[#140609] border border-[#3b0008] hover:border-[#6a0712] rounded-2xl p-5 flex flex-col justify-between space-y-4 transition-all shadow-md"
              >
                <div className="space-y-2">
                  <div className="flex items-start justify-between">
                    <span className="text-3xl p-2 rounded-xl bg-[#200910] border border-[#3b0008]">
                      {item.icon}
                    </span>
                    <span className="text-xs font-cinzel font-bold text-[#d4af37] px-2 py-1 rounded bg-[#2a0b12] border border-[#d4af37]/30">
                      🪙 {item.price.toLocaleString()}
                    </span>
                  </div>

                  <h4 className="font-bold text-white text-sm mt-2">{item.name}</h4>
                  <p className="text-xs text-stone-300 font-light leading-relaxed">
                    {item.description}
                  </p>
                </div>

                <div className="pt-3 border-t border-[#3b0008]/50 flex items-center justify-between">
                  <span className="text-[10px] text-stone-500 font-medium">{item.stock}</span>
                  <button
                    onClick={() => handleBuy(item)}
                    disabled={currentMember.balance < item.price}
                    className={`px-4 py-2 rounded-xl text-xs font-bold transition-all ${
                      currentMember.balance >= item.price
                        ? 'bg-[#510a14] hover:bg-[#6e0e1c] text-white border border-[#d4af37]/40'
                        : 'bg-stone-900 text-stone-500 cursor-not-allowed'
                    }`}
                  >
                    Adquirir
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Leaderboard Section */}
      {activeSection === 'leaderboard' && (
        <div className="bg-[#12070a] border border-[#3b0008] rounded-2xl overflow-hidden shadow-lg">
          <div className="p-4 border-b border-[#3b0008]/60 flex items-center justify-between">
            <h3 className="font-cinzel font-bold text-white text-sm tracking-wide">
              TABLA DE HONOR IMPERIAL • DIGNATARIOS DESTACADOS
            </h3>
            <span className="text-xs text-stone-400">Rango &amp; Fichas acumuladas</span>
          </div>

          <div className="divide-y divide-[#3b0008]/40">
            {members.map((m, idx) => (
              <div key={m.id} className="p-4 flex items-center justify-between hover:bg-[#16080c]/50 transition-colors">
                <div className="flex items-center gap-3">
                  <span className={`w-6 text-center font-bold font-cinzel ${
                    idx === 0 ? 'text-[#d4af37] text-lg' : idx === 1 ? 'text-stone-300' : idx === 2 ? 'text-amber-700' : 'text-stone-500 text-xs'
                  }`}>
                    {idx + 1}
                  </span>
                  <div>
                    <div className="font-bold text-white text-xs sm:text-sm">@{m.username}</div>
                    <div className="text-[11px] text-stone-400">
                      Nivel {m.level} • {m.rank}
                    </div>
                  </div>
                </div>

                <div className="text-right">
                  <div className="font-cinzel font-bold text-[#d4af37] text-xs sm:text-sm">
                    🪙 {m.balance.toLocaleString()}
                  </div>
                  <div className="text-[10px] text-stone-500">
                    {m.inventory.length} artículo(s)
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
