export interface GuildInfo {
  id: string;
  name: string;
  owner: string;
  memberCount: number;
  onlineCount: number;
  channelsCount: number;
  rolesCount: number;
  welcomeChannel: string;
  welcomeMessage: string;
  welcomeImageUrl: string;
  language: string;
  responsesPublic: boolean;
  notificationsEnabled: boolean;
}

export interface Detector {
  name: string;
  limit?: number;
  minDays?: number;
  windowSec?: number;
  enabled: boolean;
  triggeredCount: number;
}

export interface AntiRaidState {
  enabled: boolean;
  lockdownActive: boolean;
  lockdownUntil: string | null;
  lockdownChannels: string[];
  detectors: Record<string, Detector>;
  liveMetrics: {
    joinsLastTenSeconds: number;
    channelDeletesLastMin: number;
    roleDeletesLastMin: number;
    messagesPerSec: number;
    threatLevel: string;
  };
  incidents: {
    id: string;
    type: string;
    target: string;
    action: string;
    mitigatedAt: string;
    details: string;
    severity: string;
  }[];
}

export interface Member {
  id: string;
  username: string;
  balance: number;
  level: number;
  rank: string;
  xp: number;
  lastDaily: string;
  inventory: string[];
  isStaff?: boolean;
}

export interface ShopItem {
  id: string;
  key: string;
  name: string;
  description: string;
  price: number;
  stock: string;
  icon: string;
}

export interface Ban {
  userId: string;
  username: string;
  reason: string;
  bannedAt: string;
  moderator: string;
}

export interface Timeout {
  userId: string;
  username: string;
  reason: string;
  until: string;
  moderator: string;
}

export interface Appeal {
  id: number;
  targetId: string;
  username: string;
  sanction: string;
  reason: string;
  status: 'pending' | 'accepted' | 'denied';
  createdAt: string;
}

export interface AuditLog {
  id: string;
  actor: string;
  action: string;
  details: string;
  timestamp: string;
}

export interface AiInteraction {
  id: string;
  sender: string;
  prompt: string;
  response: string;
  command: string;
  timestamp: string;
}

export interface PresenceActivity {
  type: 'WATCHING' | 'LISTENING' | 'CUSTOM' | 'COMPETING' | 'PLAYING';
  typeLabel: string;
  text: string;
  badge: string;
  details: string;
  state: string;
}

export interface PresenceState {
  status: 'online' | 'idle' | 'dnd';
  currentActivityIndex: number;
  currentActivity: PresenceActivity;
  activities: PresenceActivity[];
  intervalSeconds: number;
  lastRotatedAt: string;
}
