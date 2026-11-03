package antiraid

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/cerebras"
	"krypton.dev/internal/config"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

// DetectorResult es el resultado devuelto por los detectores de seguridad
type DetectorResult struct {
	Triggered  bool           `json:"triggered"`
	Detector   string         `json:"detector"`
	Score      float64        `json:"score"`
	Severity   int            `json:"severity"`
	Confidence float64        `json:"confidence"`
	Action     string         `json:"action"`
	Reason     string         `json:"reason"`
	Details    string         `json:"details,omitempty"`
	Evidence   any            `json:"evidence,omitempty"`
	DeleteMsg  bool           `json:"delete_msg,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp,omitempty"`
}

// Event representa un suceso registrado en la ventana temporal del antiraid
type Event struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	GuildID   string            `json:"guild_id"`
	ChannelID string            `json:"channel_id"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Time      time.Time         `json:"time"`
	CreatedAt time.Time         `json:"created_at"`
	Data      any               `json:"data,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"` // map[string]string requerido por los detectores
}

type Detector interface{}

type userSpamRecord struct {
	lastMessages []time.Time
}

type guildSecurityState struct {
	joinsWindow    []time.Time
	channelDeletes []time.Time
	roleDeletes    []time.Time
	lockdownActive bool
	lockdownUntil  time.Time
}

type Engine struct {
	cfg      config.AntiraidConfig
	session  *discordgo.Session
	store    *storage.Store
	theme    *ui.Theme
	aiClient *cerebras.Client
	logger   *slog.Logger
	lockdown *LockdownManager

	inviteRegex *regexp.Regexp
	phishRegex  *regexp.Regexp

	userSpam   map[string]*userSpamRecord
	guildState map[string]*guildSecurityState
	mu         sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

type SecuritySnapshot struct {
	JoinsLastTenSeconds   int
	ChannelDeletesLastMin int
	RoleDeletesLastMin    int
	AutoLockdownActive    bool
	AutoLockdownUntil     time.Time
}

func New(
	cfg config.AntiraidConfig,
	session *discordgo.Session,
	store *storage.Store,
	theme *ui.Theme,
	aiClient *cerebras.Client,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		cfg:         cfg,
		session:     session,
		store:       store,
		theme:       theme,
		aiClient:    aiClient,
		logger:      logger,
		inviteRegex: regexp.MustCompile(`(?i)(discord\.gg/|discord\.com/invite/|discordapp\.com/invite/)[a-zA-Z0-9]+`),
		phishRegex:  regexp.MustCompile(`(?i)(steamcommunity\.com\.[\w-]+\.ru|free-nitro|steam-nitro|dlscord\.gift|discorcl\.gift)`),
		userSpam:    make(map[string]*userSpamRecord),
		guildState:  make(map[string]*guildSecurityState),
	}
}

func (e *Engine) SetLockdownManager(manager *LockdownManager) {
	e.mu.Lock()
	e.lockdown = manager
	e.mu.Unlock()
}

func (e *Engine) SecurityStatus(guildID string) SecuritySnapshot {
	now := time.Now()
	e.mu.RLock()
	defer e.mu.RUnlock()
	state := e.guildState[guildID]
	if state == nil {
		return SecuritySnapshot{}
	}
	snapshot := SecuritySnapshot{
		AutoLockdownActive: state.lockdownActive && now.Before(state.lockdownUntil),
		AutoLockdownUntil:  state.lockdownUntil,
	}
	for _, occurredAt := range state.joinsWindow {
		if now.Sub(occurredAt) <= 10*time.Second {
			snapshot.JoinsLastTenSeconds++
		}
	}
	for _, occurredAt := range state.channelDeletes {
		if now.Sub(occurredAt) <= time.Minute {
			snapshot.ChannelDeletesLastMin++
		}
	}
	for _, occurredAt := range state.roleDeletes {
		if now.Sub(occurredAt) <= time.Minute {
			snapshot.RoleDeletesLastMin++
		}
	}
	return snapshot
}

func (e *Engine) RegisterDetector(d Detector) {}

func (e *Engine) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.logger.Info("🛡️ Lucifer ha activado la guardia del club (Antiraid y Automod operativos)")

	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-e.ctx.Done():
				return
			case <-ticker.C:
				e.cleanupStaleRecords()
			}
		}
	}()
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.logger.Info("🛡️ Protocolos defensivos del Lux desactivados.")
}

func (e *Engine) InvalidateGuild(guildID string) {
	e.mu.Lock()
	delete(e.guildState, guildID)
	e.mu.Unlock()
}

func (e *Engine) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.GuildID == "" {
		return
	}

	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return
	}

	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		return
	}

	if e.isStaff(guild, member) {
		return
	}

	if e.phishRegex.MatchString(m.Content) {
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
		_ = s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "Automod: Enlace malicioso o phishing detectado", 0)
		e.notifyIncident(s, m.GuildID, m.ChannelID, "🔥 Miembro expulsado por estafa",
			fmt.Sprintf("Lucifer expulsó a %s por publicar enlaces sospechosos o falsos sorteos.", m.Author.Mention()),
			e.theme.ColorDanger)
		return
	}

	if e.inviteRegex.MatchString(m.Content) {
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
		e.warnUser(s, m.ChannelID, m.Author, "Las promociones no autorizadas están prohibidas en este club. Tu mensaje fue incinerado.")
		return
	}

	if len(m.Mentions) >= 4 {
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
		timeoutDuration := time.Now().Add(10 * time.Minute)
		_ = s.GuildMemberTimeout(m.GuildID, m.Author.ID, &timeoutDuration)

		e.notifyIncident(s, m.GuildID, m.ChannelID, "🤐 Lucifer aplicó un silencio",
			fmt.Sprintf("%s fue silenciado por **10 minutos** tras intentar un bombardeo de menciones masivas (%d usuarios).", m.Author.Mention(), len(m.Mentions)),
			e.theme.ColorWarning)
		return
	}

	if e.checkUserSpam(m.GuildID, m.Author.ID) {
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
		timeoutDuration := time.Now().Add(5 * time.Minute)
		_ = s.GuildMemberTimeout(m.GuildID, m.Author.ID, &timeoutDuration)

		e.notifyIncident(s, m.GuildID, m.ChannelID, "🤐 Silencio por Flood de Mensajes",
			fmt.Sprintf("%s fue silenciado por **5 minutos** tras saturar el chat a una velocidad sobrehumana.", m.Author.Mention()),
			e.theme.ColorWarning)
		return
	}
}

func (e *Engine) OnMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if m.GuildID == "" || m.User == nil {
		return
	}

	now := time.Now()
	e.mu.Lock()
	state, exists := e.guildState[m.GuildID]
	if !exists {
		state = &guildSecurityState{}
		e.guildState[m.GuildID] = state
	}

	state.joinsWindow = append(state.joinsWindow, now)
	var recentJoins []time.Time
	for _, t := range state.joinsWindow {
		if now.Sub(t) <= 10*time.Second {
			recentJoins = append(recentJoins, t)
		}
	}
	state.joinsWindow = recentJoins
	joinCount := len(recentJoins)
	e.mu.Unlock()

	if joinCount >= 6 {
		e.handleJoinRaid(s, m.GuildID, joinCount)
	}

	createdAt := getSnowflakeTime(m.User.ID)
	accountAge := time.Since(createdAt)
	if accountAge < 24*time.Hour && !m.User.Bot {
		e.logger.Warn("⚠️ Cuenta recién nacida detectada en el club", "user", m.User.Username, "age", accountAge.String())
	}
}

func (e *Engine) OnChannelDelete(s *discordgo.Session, c *discordgo.ChannelDelete) {
	e.checkNukeThreat(s, c.GuildID, "canal")
}

func (e *Engine) OnRoleDelete(s *discordgo.Session, r *discordgo.GuildRoleDelete) {
	e.checkNukeThreat(s, r.GuildID, "rol")
}

func (e *Engine) OnGuildBanAdd(s *discordgo.Session, b *discordgo.GuildBanAdd) {}

func (e *Engine) handleJoinRaid(s *discordgo.Session, guildID string, count int) {
	e.mu.Lock()
	state := e.guildState[guildID]
	if state.lockdownActive {
		e.mu.Unlock()
		return
	}
	state.lockdownActive = true
	state.lockdownUntil = time.Now().Add(5 * time.Minute)
	e.mu.Unlock()

	guild, err := s.Guild(guildID)
	if err != nil {
		return
	}

	e.logger.Warn("🚨 ¡ATAQUE DE RAID DETECTADO EN EL CLUB!", "guild", guild.Name, "joins_en_10s", count)
	e.mu.RLock()
	lockdown := e.lockdown
	e.mu.RUnlock()
	if lockdown != nil {
		go func() {
			if err := lockdown.Activate(context.Background(), guildID, 5*time.Minute); err != nil {
				e.logger.Error("no se pudo aplicar el bloqueo automático tras detectar un raid", "guild", guildID, "err", err)
			}
		}()
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🚨 ALERTA MÁXIMA: ATAQUE DETECTADO",
		Description: fmt.Sprintf("Lucifer detectó una avalancha inusual de ingresos (**%d miembros en 10 segundos**).\n\n**Acciones tomadas automáticamente:**\n• Cerrojos del Lux bloqueados temporalmente.\n• Monitoreo intensivo de nuevos usuarios activado por 5 minutos.", count),
		Color:       e.theme.ColorDanger,
		Image:       &discordgo.MessageEmbedImage{URL: "https://media.giphy.com/media/l0HlCqV35hdEg2GUo/giphy.gif"},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Protocolo de Defensa Activa • El Lux"},
	}

	e.sendGuildAlert(s, guild, embed)

	go func() {
		time.Sleep(5 * time.Minute)
		e.mu.Lock()
		if st, ok := e.guildState[guildID]; ok {
			st.lockdownActive = false
		}
		e.mu.Unlock()
	}()
}

func (e *Engine) checkNukeThreat(s *discordgo.Session, guildID, entityType string) {
	now := time.Now()
	e.mu.Lock()
	state, exists := e.guildState[guildID]
	if !exists {
		state = &guildSecurityState{}
		e.guildState[guildID] = state
	}

	if entityType == "canal" {
		state.channelDeletes = append(state.channelDeletes, now)
	} else {
		state.roleDeletes = append(state.roleDeletes, now)
	}

	var recentDeletes int
	list := state.channelDeletes
	if entityType == "rol" {
		list = state.roleDeletes
	}

	for _, t := range list {
		if now.Sub(t) <= 15*time.Second {
			recentDeletes++
		}
	}
	e.mu.Unlock()

	if recentDeletes >= 3 {
		guild, err := s.Guild(guildID)
		if err != nil {
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "☢️ INTENTO DE DESTRUCCIÓN (ANTI-NUKE)",
			Description: fmt.Sprintf("Se detectó el borrado masivo de %ss en menos de 15 segundos. Se recomienda revisar los registros de auditoría inmediatamente.", entityType),
			Color:       e.theme.ColorDanger,
			Footer:      &discordgo.MessageEmbedFooter{Text: "Alerta Crítica de Seguridad"},
		}
		e.sendGuildAlert(s, guild, embed)
	}
}

func (e *Engine) checkUserSpam(guildID, userID string) bool {
	key := guildID + ":" + userID
	now := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	record, exists := e.userSpam[key]
	if !exists {
		record = &userSpamRecord{}
		e.userSpam[key] = record
	}

	record.lastMessages = append(record.lastMessages, now)

	var recent []time.Time
	for _, t := range record.lastMessages {
		if now.Sub(t) <= 3*time.Second {
			recent = append(recent, t)
		}
	}
	record.lastMessages = recent

	return len(recent) >= 5
}

func (e *Engine) isStaff(guild *discordgo.Guild, member *discordgo.Member) bool {
	if member.User.ID == guild.OwnerID {
		return true
	}
	for _, roleID := range member.Roles {
		for _, r := range guild.Roles {
			if r.ID == roleID {
				if r.Permissions&discordgo.PermissionAdministrator != 0 ||
					r.Permissions&discordgo.PermissionManageMessages != 0 ||
					r.Permissions&discordgo.PermissionKickMembers != 0 {
					return true
				}
			}
		}
	}
	return false
}

func (e *Engine) warnUser(s *discordgo.Session, channelID string, user *discordgo.User, reason string) {
	msg := fmt.Sprintf("⚠️ %s: *%s*", user.Mention(), reason)
	sent, err := discordutil.ChannelMessageSend(s, channelID, msg)
	if err == nil {
		go func(msgID string) {
			time.Sleep(6 * time.Second)
			_ = s.ChannelMessageDelete(channelID, msgID)
		}(sent.ID)
	}
}

func (e *Engine) notifyIncident(s *discordgo.Session, guildID, channelID, title, description string, color int) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Footer:      &discordgo.MessageEmbedFooter{Text: e.theme.FooterText, IconURL: e.theme.FooterIcon},
	}
	_, _ = discordutil.ChannelMessageSendEmbed(s, channelID, embed)
}

func (e *Engine) sendGuildAlert(s *discordgo.Session, guild *discordgo.Guild, embed *discordgo.MessageEmbed) {
	if guild.SystemChannelID != "" {
		_, err := discordutil.ChannelMessageSendEmbed(s, guild.SystemChannelID, embed)
		if err == nil {
			return
		}
	}

	channels, err := s.GuildChannels(guild.ID)
	if err != nil {
		return
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildText {
			_, err := discordutil.ChannelMessageSendEmbed(s, ch.ID, embed)
			if err == nil {
				break
			}
		}
	}
}

func (e *Engine) cleanupStaleRecords() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	for k, record := range e.userSpam {
		if len(record.lastMessages) == 0 || now.Sub(record.lastMessages[len(record.lastMessages)-1]) > 1*time.Minute {
			delete(e.userSpam, k)
		}
	}
}

func getSnowflakeTime(snowflake string) time.Time {
	var id int64
	fmt.Sscanf(snowflake, "%d", &id)
	timestamp := (id >> 22) + 1420070400000
	return time.Unix(0, timestamp*int64(time.Millisecond))
}
