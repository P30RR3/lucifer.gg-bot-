import React from 'react';
import {
  ShieldAlert,
  Flame,
  Coins,
  Crown,
  Sparkles,
  Activity,
  Scroll,
} from 'lucide-react';
import { PresenceState } from '../types.ts';

interface NavigationProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  isLockdown: boolean;
  presence?: PresenceState;
  onAdvancePresence?: () => void;
  isStaff?: boolean;
}

export const Navigation: React.FC<NavigationProps> = ({
  activeTab,
  setActiveTab,
  isLockdown,
  presence,
  isStaff = false,
}) => {
  
  // Pestañas filtradas: solo staff ve las de moderación y auditoría
  const allTabs = [
    { id: 'overview', label: 'Protocolo • Estado', icon: Activity, staffOnly: true },
    { id: 'antiraid', label: 'Escudo Antiraid', icon: ShieldAlert, badge: isLockdown ? 'BLOQUEO' : undefined, staffOnly: true },
    { id: 'security', label: 'Alta Seguridad', icon: Flame, staffOnly: true },
    { id: 'luxpanel', label: 'Expediente Soberano', icon: Crown, staffOnly: false },
    { id: 'economy', label: 'Tesorería & Casino', icon: Coins, staffOnly: false },
    { id: 'oracle', label: 'Oráculo Lucifer AI', icon: Sparkles, staffOnly: true },
    { id: 'audit', label: 'Libro Mayor', icon: Scroll, staffOnly: true },
  ];

  const tabs = allTabs.filter(tab => !tab.staffOnly || isStaff);

  const currentActivity = presence?.currentActivity;

  return (
    <header className="border-b border-[#3b0008]/60 bg-[#0d0709]/95 backdrop-blur-md sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo & Brand */}
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-[#6a0712] via-[#3b0008] to-[#120306] border border-[#d4af37]/50 flex items-center justify-center shadow-lg shadow-[#6a0712]/30">
              <span className="text-xl">⚜️</span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-cinzel font-bold text-lg text-white tracking-wider">
                  LUCIFER PROTOCOL
                </span>
                <span className="text-[10px] uppercase tracking-widest px-1.5 py-0.5 rounded bg-[#4a000a] text-[#d4af37] border border-[#d4af37]/40 font-semibold">
                  SOBERANO
                </span>
              </div>
              <p className="text-[11px] text-stone-400 font-light hidden sm:block">
                Soberanía, diplomacia y distinción absoluta ⚜️
              </p>
            </div>
          </div>

          {/* Dynamic Live Sovereign Presence Pill (Solo informativo, sin botón de cambiarlo) */}
          {currentActivity && (
            <div className="hidden xl:flex items-center gap-2 px-3 py-1 rounded-full bg-[#18080d] border border-[#d4af37]/35 shadow-inner text-xs">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
              <span className="text-stone-400 font-mono text-[11px] uppercase tracking-wider">
                {currentActivity.typeLabel}:
              </span>
              <span className="font-medium text-white truncate max-w-xs flex items-center gap-1.5">
                {currentActivity.text}
              </span>
            </div>
          )}

          {/* Navigation Tabs */}
          <nav className="hidden lg:flex items-center space-x-1">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-150 ${
                    isActive
                      ? 'bg-gradient-to-r from-[#510a14] to-[#3b0008] text-white border border-[#d4af37]/40 shadow-sm shadow-[#6a0712]/30'
                      : 'text-stone-300 hover:text-white hover:bg-[#1a0a0f]'
                  }`}
                >
                  <Icon className={`w-4 h-4 ${isActive ? 'text-[#d4af37]' : 'text-stone-400'}`} />
                  <span>{tab.label}</span>
                  {tab.badge && (
                    <span className="ml-1 px-1.5 py-0.2 text-[9px] font-bold rounded bg-red-950 text-red-400 border border-red-700 animate-pulse">
                      {tab.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </nav>

          {/* Discord Gateway Live Status Indicator */}
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[#18080d] border border-[#d4af37]/35 shadow-inner">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
              <span className="font-cinzel text-white text-[11px] hidden sm:inline">Bot Oficial</span>
              <span className="text-[10px] uppercase tracking-wider text-[#d4af37] font-mono">En Línea</span>
            </div>
          </div>
        </div>

        {/* Mobile Navigation Bar */}
        <div className="flex lg:hidden overflow-x-auto py-2 space-x-1 border-t border-[#3b0008]/40 no-scrollbar">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs whitespace-nowrap ${
                  isActive
                    ? 'bg-[#510a14] text-white border border-[#d4af37]/30'
                    : 'text-stone-400 hover:text-stone-200'
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>
      </div>
    </header>
  );
};