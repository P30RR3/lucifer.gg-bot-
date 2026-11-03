package bot

import "krypton.dev/pkg/discordutil"
import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/storage"
)

type Auditor struct {
	store         *storage.Store
	log           *slog.Logger
	settingsMu    sync.RWMutex
	settingsCache map[string]cachedAuditSettings
	messageMu     sync.Mutex
	messages      map[string]*list.Element
	messageOrder  *list.List
}

type cachedAuditSettings struct {
	value     storage.AuditSettings
	expiresAt time.Time
}

type cachedAuditMessage struct {
	guildID   string
	messageID string
	authorID  string
	channelID string
	content   string
}

func NewAuditor(store *storage.Store, logger *slog.Logger) *Auditor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Auditor{
		store:         store,
		log:           logger.With("component", "audit"),
		settingsCache: make(map[string]cachedAuditSettings),
		messages:      make(map[string]*list.Element),
		messageOrder:  list.New(),
	}
}

func (a *Auditor) OnMessageCreate(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.GuildID == "" || event.Author == nil {
		return
	}
	settings, err := a.settings(event.GuildID)
	if err != nil {
		a.log.Error("no se pudo cargar la configuración de auditoría", "guild_id", event.GuildID, "err", err)
		return
	}
	message := cachedAuditMessage{
		guildID:   event.GuildID,
		messageID: event.ID,
		authorID:  event.Author.ID,
		channelID: event.ChannelID,
	}
	if settings.IncludeDeletedContent {
		message.content = event.Content
	}
	a.rememberMessage(message)
}

func (a *Auditor) OnMessageDelete(s *discordgo.Session, event *discordgo.MessageDelete) {
	if event == nil {
		return
	}
	message := event.Message
	if message == nil {
		message = event.BeforeDelete
	}
	if message == nil {
		return
	}
	guildID := message.GuildID
	if guildID == "" {
		guildID = a.guildForChannel(s, message.ChannelID)
	}
	if guildID == "" {
		return
	}
	cachedMessage, found := a.takeMessage(message.ID)

	settings, err := a.settings(guildID)
	if err != nil {
		a.log.Error("no se pudo cargar la configuración de auditoría", "guild_id", guildID, "err", err)
		return
	}

	authorID, content := "", ""
	if message.Author != nil {
		authorID = message.Author.ID
	}
	if authorID == "" && found {
		authorID = cachedMessage.authorID
	}
	if message.ChannelID == "" && found {
		message.ChannelID = cachedMessage.channelID
	}
	if settings.IncludeDeletedContent {
		content = message.Content
		if content == "" && found {
			content = cachedMessage.content
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = a.store.RecordDeletedMessage(ctx, guildID, storage.DeletedMessage{
		MessageID: message.ID,
		AuthorID:  authorID,
		ChannelID: message.ChannelID,
		Content:   content,
	})
	cancel()
	if err != nil {
		a.log.Error("no se pudo registrar el mensaje eliminado", "guild_id", guildID, "message_id", message.ID, "err", err)
	}

	summary := fmt.Sprintf("Se eliminó un mensaje en <#%s>.", message.ChannelID)
	if authorID != "" {
		summary += fmt.Sprintf("\nAutor: <@%s> · ID: `%s`", authorID, authorID)
	}
	if len(message.Attachments) > 0 {
		summary += fmt.Sprintf("\nAdjuntos: %d", len(message.Attachments))
	}
	details := ""
	if content != "" {
		details = "Contenido (archivado por configuración):\n" + truncateDiscord(content, 1000)
	}
	a.emit(s, settings, "🗑️ Mensaje eliminado", summary, details)
}

func (a *Auditor) OnMessageDeleteBulk(s *discordgo.Session, event *discordgo.MessageDeleteBulk) {
	if event == nil || event.GuildID == "" {
		return
	}
	settings, err := a.settings(event.GuildID)
	if err != nil {
		a.log.Error("no se pudo cargar la configuración de auditoría", "guild_id", event.GuildID, "err", err)
		return
	}
	deletedMessages := make([]storage.DeletedMessage, 0, len(event.Messages))
	for _, id := range event.Messages {
		cachedMessage, found := a.takeMessage(id)
		deleted := storage.DeletedMessage{MessageID: id, ChannelID: event.ChannelID}
		if found {
			deleted.AuthorID = cachedMessage.authorID
			if settings.IncludeDeletedContent {
				deleted.Content = cachedMessage.content
			}
		}
		deletedMessages = append(deletedMessages, deleted)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := a.store.RecordDeletedMessages(ctx, event.GuildID, deletedMessages); err != nil {
		a.log.Error("no se pudo registrar la limpieza de mensajes", "guild_id", event.GuildID, "count", len(deletedMessages), "err", err)
	}
	cancel()
	a.emit(s, settings, "🧹 Limpieza de mensajes", fmt.Sprintf("Se eliminaron **%d mensajes** en <#%s>.", len(event.Messages), event.ChannelID), "")
}

func (a *Auditor) OnMessageUpdate(s *discordgo.Session, event *discordgo.MessageUpdate) {
	if event == nil || event.Message == nil || event.GuildID == "" {
		return
	}
	if event.Content == "" {
		return
	}
	cachedMessage, found := a.peekMessage(event.ID)
	beforeContent := ""
	if event.BeforeUpdate != nil {
		beforeContent = event.BeforeUpdate.Content
	} else if found {
		beforeContent = cachedMessage.content
	}
	if beforeContent == event.Content {
		return
	}
	settings, err := a.settings(event.GuildID)
	if err != nil {
		a.log.Error("no se pudo cargar la configuración de auditoría", "guild_id", event.GuildID, "err", err)
		return
	}
	updatedMessage := cachedAuditMessage{
		guildID:   event.GuildID,
		messageID: event.ID,
		channelID: event.ChannelID,
	}
	authorID := ""
	if event.Author != nil {
		authorID = event.Author.ID
	}
	if authorID == "" && found {
		authorID = cachedMessage.authorID
	}
	updatedMessage.authorID = authorID
	if settings.IncludeDeletedContent {
		updatedMessage.content = event.Content
	}
	a.rememberMessage(updatedMessage)
	summary := fmt.Sprintf("Se editó un mensaje en <#%s> · ID `%s`.", event.ChannelID, event.ID)
	if authorID != "" {
		summary += fmt.Sprintf("\nAutor: <@%s>", authorID)
	}
	summary += "\nEl texto no se copia al diario."
	a.logEvent(s, event.GuildID, "✏️ Mensaje editado", summary)
}

func (a *Auditor) OnVoiceStateUpdate(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	if event == nil || event.VoiceState == nil || event.GuildID == "" {
		return
	}
	before := event.BeforeUpdate
	oldChannel := ""
	if before != nil {
		oldChannel = before.ChannelID
	}
	newChannel := event.ChannelID
	if oldChannel == newChannel && before != nil &&
		before.Mute == event.Mute && before.Deaf == event.Deaf &&
		before.SelfMute == event.SelfMute && before.SelfDeaf == event.SelfDeaf {
		return
	}

	summary := ""
	switch {
	case oldChannel == "" && newChannel != "":
		summary = fmt.Sprintf("<@%s> entró al canal de voz <#%s>.", event.UserID, newChannel)
	case oldChannel != "" && newChannel == "":
		summary = fmt.Sprintf("<@%s> salió del canal de voz <#%s>.", event.UserID, oldChannel)
	case oldChannel != newChannel:
		summary = fmt.Sprintf("<@%s> se movió de <#%s> a <#%s>.", event.UserID, oldChannel, newChannel)
	default:
		summary = fmt.Sprintf("Cambió el estado de voz de <@%s> en <#%s>.", event.UserID, newChannel)
	}
	a.logEvent(s, event.GuildID, "🎙️ Movimiento en el club de voz", summary)
}

func (a *Auditor) OnGuildMemberAdd(s *discordgo.Session, event *discordgo.GuildMemberAdd) {
	if event == nil || event.GuildID == "" || event.Member == nil || event.Member.User == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🚪 Nueva alma en el Lux", fmt.Sprintf("<@%s> se unió al servidor.", event.Member.User.ID))
}

func (a *Auditor) OnGuildMemberRemove(s *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event == nil || event.GuildID == "" || event.Member == nil || event.Member.User == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🚶 Salida del club", fmt.Sprintf("<@%s> salió del servidor.", event.Member.User.ID))
}

func (a *Auditor) OnGuildMemberUpdate(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	if event == nil || event.GuildID == "" || event.Member == nil || event.Member.User == nil || event.BeforeUpdate == nil {
		return
	}
	before := event.BeforeUpdate
	changes := make([]string, 0, 2)
	if before.Nick != event.Nick {
		changes = append(changes, "cambió su apodo")
	}
	if !sameStrings(before.Roles, event.Roles) {
		changes = append(changes, "cambió sus roles")
	}
	if len(changes) == 0 {
		return
	}
	a.logEvent(s, event.GuildID, "🪪 Expediente de miembro actualizado",
		fmt.Sprintf("<@%s> %s.", event.User.ID, strings.Join(changes, " y ")))
}

func (a *Auditor) OnGuildBanAdd(s *discordgo.Session, event *discordgo.GuildBanAdd) {
	if event == nil || event.User == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🔥 Ban aplicado", fmt.Sprintf("La cuenta `%s` fue baneada.", event.User.ID))
}

func (a *Auditor) OnGuildBanRemove(s *discordgo.Session, event *discordgo.GuildBanRemove) {
	if event == nil || event.User == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🕊️ Ban retirado", fmt.Sprintf("Se retiró el ban de `%s`.", event.User.ID))
}

func (a *Auditor) OnChannelCreate(s *discordgo.Session, event *discordgo.ChannelCreate) {
	if event == nil || event.Channel == nil || event.GuildID == "" {
		return
	}
	a.logEvent(s, event.GuildID, "➕ Canal creado", fmt.Sprintf("Se creó **%s** (`%s`).", event.Name, event.ID))
}

func (a *Auditor) OnChannelUpdate(s *discordgo.Session, event *discordgo.ChannelUpdate) {
	if event == nil || event.Channel == nil || event.GuildID == "" {
		return
	}
	if event.BeforeUpdate != nil && event.BeforeUpdate.Name == event.Name &&
		event.BeforeUpdate.Topic == event.Topic && event.BeforeUpdate.ParentID == event.ParentID {
		return
	}
	a.logEvent(s, event.GuildID, "📝 Canal actualizado", fmt.Sprintf("El canal **%s** (`%s`) fue modificado.", event.Name, event.ID))
}

func (a *Auditor) OnChannelDelete(s *discordgo.Session, event *discordgo.ChannelDelete) {
	if event == nil || event.Channel == nil || event.GuildID == "" {
		return
	}
	a.logEvent(s, event.GuildID, "➖ Canal eliminado", fmt.Sprintf("Se eliminó **%s** (`%s`).", event.Name, event.ID))
}

func (a *Auditor) OnThreadCreate(s *discordgo.Session, event *discordgo.ThreadCreate) {
	if event == nil || event.Channel == nil || event.GuildID == "" {
		return
	}
	a.logEvent(s, event.GuildID, "🧵 Hilo abierto", fmt.Sprintf("Se abrió **%s** (`%s`).", event.Name, event.ID))
}

func (a *Auditor) OnThreadDelete(s *discordgo.Session, event *discordgo.ThreadDelete) {
	if event == nil || event.Channel == nil || event.GuildID == "" {
		return
	}
	a.logEvent(s, event.GuildID, "🧵 Hilo cerrado", fmt.Sprintf("Se cerró **%s** (`%s`).", event.Name, event.ID))
}

func (a *Auditor) OnGuildRoleCreate(s *discordgo.Session, event *discordgo.GuildRoleCreate) {
	if event == nil || event.GuildRole == nil || event.Role == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🎭 Rol creado", fmt.Sprintf("Se creó el rol **%s** (`%s`).", event.Role.Name, event.Role.ID))
}

func (a *Auditor) OnGuildRoleUpdate(s *discordgo.Session, event *discordgo.GuildRoleUpdate) {
	if event == nil || event.GuildRole == nil || event.Role == nil {
		return
	}
	a.logEvent(s, event.GuildID, "🎭 Rol actualizado", fmt.Sprintf("Se actualizó el rol **%s** (`%s`).", event.Role.Name, event.Role.ID))
}

func (a *Auditor) OnGuildRoleDelete(s *discordgo.Session, event *discordgo.GuildRoleDelete) {
	if event == nil || event.GuildID == "" {
		return
	}
	a.logEvent(s, event.GuildID, "🎭 Rol eliminado", fmt.Sprintf("Se eliminó el rol `%s`.", event.RoleID))
}

func (a *Auditor) RecordStaffAction(ctx context.Context, s *discordgo.Session, guildID, actorID, targetID, action, details string) error {
	if err := a.store.RecordStaffAction(ctx, guildID, actorID, targetID, action, details); err != nil {
		return err
	}
	title := map[string]string{
		"ban":     "🔥 Ban ejecutado por la guardia",
		"warn":    "⚠️ Advertencia registrada",
		"kick":    "🔪 Expulsión ejecutada por la guardia",
		"timeout": "🤐 Silencio aplicado por la guardia",
		"purge":   "🧹 Limpieza ejecutada por la guardia",
	}[action]
	if title == "" {
		title = "⚖️ Acción de moderación registrada"
	}
	actionLabel := map[string]string{
		"ban":     "un ban",
		"warn":    "una advertencia",
		"kick":    "una expulsión",
		"timeout": "un silencio temporal",
		"purge":   "una limpieza",
	}[action]
	if actionLabel == "" {
		actionLabel = action
	}
	summary := fmt.Sprintf("<@%s> registró **%s**.", actorID, actionLabel)
	if targetID != "" {
		summary += fmt.Sprintf("\nUsuario: <@%s>", targetID)
	}
	if details != "" {
		summary += "\nDetalle: " + truncateDiscord(details, 1000)
	}
	a.logEvent(s, guildID, title, summary)
	return nil
}

func (a *Auditor) logEvent(s *discordgo.Session, guildID, title, summary string) {
	settings, err := a.settings(guildID)
	if err != nil {
		a.log.Error("no se pudo cargar la configuración de auditoría", "guild_id", guildID, "err", err)
		return
	}
	a.emit(s, settings, title, summary, "")
}

func (a *Auditor) settings(guildID string) (storage.AuditSettings, error) {
	a.settingsMu.RLock()
	cached, ok := a.settingsCache[guildID]
	a.settingsMu.RUnlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	settings, err := a.store.GetAuditSettings(ctx, guildID)
	if err != nil {
		return storage.AuditSettings{}, err
	}
	a.settingsMu.Lock()
	a.settingsCache[guildID] = cachedAuditSettings{value: settings, expiresAt: time.Now().Add(15 * time.Second)}
	a.settingsMu.Unlock()
	return settings, nil
}

func (a *Auditor) UpdateSettings(settings storage.AuditSettings) {
	a.settingsMu.Lock()
	a.settingsCache[settings.GuildID] = cachedAuditSettings{value: settings, expiresAt: time.Now().Add(15 * time.Second)}
	a.settingsMu.Unlock()
	if settings.IncludeDeletedContent {
		return
	}
	a.messageMu.Lock()
	for element := a.messageOrder.Front(); element != nil; element = element.Next() {
		message := element.Value.(cachedAuditMessage)
		if message.guildID == settings.GuildID {
			message.content = ""
			element.Value = message
		}
	}
	a.messageMu.Unlock()
}

func (a *Auditor) rememberMessage(message cachedAuditMessage) {
	if message.messageID == "" {
		return
	}
	a.messageMu.Lock()
	defer a.messageMu.Unlock()
	if current, ok := a.messages[message.messageID]; ok {
		current.Value = message
		a.messageOrder.MoveToBack(current)
		return
	}
	a.messages[message.messageID] = a.messageOrder.PushBack(message)
	if a.messageOrder.Len() > 5000 {
		oldest := a.messageOrder.Front()
		if oldest != nil {
			evicted := oldest.Value.(cachedAuditMessage)
			delete(a.messages, evicted.messageID)
			a.messageOrder.Remove(oldest)
		}
	}
}

func (a *Auditor) takeMessage(messageID string) (cachedAuditMessage, bool) {
	a.messageMu.Lock()
	defer a.messageMu.Unlock()
	element, ok := a.messages[messageID]
	if !ok {
		return cachedAuditMessage{}, false
	}
	message := element.Value.(cachedAuditMessage)
	delete(a.messages, messageID)
	a.messageOrder.Remove(element)
	return message, true
}

func (a *Auditor) peekMessage(messageID string) (cachedAuditMessage, bool) {
	a.messageMu.Lock()
	defer a.messageMu.Unlock()
	element, ok := a.messages[messageID]
	if !ok {
		return cachedAuditMessage{}, false
	}
	return element.Value.(cachedAuditMessage), true
}

func (a *Auditor) emit(s *discordgo.Session, settings storage.AuditSettings, title, summary, details string) {
	if settings.LogChannelID == "" {
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: summary,
		Color:       0x7B1FA2,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Diario del Inframundo · eventos disponibles para este servidor"},
	}
	if details != "" {
		embed.Fields = []*discordgo.MessageEmbedField{{Name: "Detalle", Value: details, Inline: false}}
	}
	_, err := discordutil.ChannelMessageSendComplex(s, settings.LogChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	})
	if err != nil {
		a.log.Error("no se pudo publicar un evento en el diario del servidor",
			"guild_id", settings.GuildID, "channel_id", settings.LogChannelID, "event", title, "err", err)
	}
}

func (a *Auditor) guildForChannel(s *discordgo.Session, channelID string) string {
	channel, err := s.State.Channel(channelID)
	if err != nil || channel == nil {
		return ""
	}
	return channel.GuildID
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func truncateDiscord(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-3]) + "..."
}
