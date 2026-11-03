package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/pkg/discordutil"
)

const (
	snapshotVersion = 1
	maxSnapshotSize = 2 << 20
	maxBackupRoles  = 250
	maxBackupRooms  = 500
)

type Cog struct{}

var guildOperations sync.Map

type guildSnapshot struct {
	Version        int                                `json:"version"`
	Roles          []roleSnapshot                     `json:"roles"`
	Channels       []channelSnapshot                  `json:"channels"`
	ServerChannels serverChannelSettings              `json:"server_channels"`
	BotChannels    storage.GuildBackupChannelSettings `json:"bot_channels"`
}

type serverChannelSettings struct {
	AFKChannelID           string `json:"afk_channel_id,omitempty"`
	SystemChannelID        string `json:"system_channel_id,omitempty"`
	RulesChannelID         string `json:"rules_channel_id,omitempty"`
	PublicUpdatesChannelID string `json:"public_updates_channel_id,omitempty"`
}

type roleSnapshot struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Color        int    `json:"color"`
	Position     int    `json:"position"`
	Permissions  int64  `json:"permissions"`
	Hoist        bool   `json:"hoist"`
	Mentionable  bool   `json:"mentionable"`
	UnicodeEmoji string `json:"unicode_emoji,omitempty"`
}

type channelSnapshot struct {
	ID                            string                           `json:"id"`
	Name                          string                           `json:"name"`
	Type                          discordgo.ChannelType            `json:"type"`
	Topic                         string                           `json:"topic,omitempty"`
	Position                      int                              `json:"position"`
	Bitrate                       int                              `json:"bitrate,omitempty"`
	UserLimit                     int                              `json:"user_limit,omitempty"`
	RateLimitPerUser              int                              `json:"rate_limit_per_user,omitempty"`
	ParentID                      string                           `json:"parent_id,omitempty"`
	NSFW                          bool                             `json:"nsfw"`
	PermissionOverwrites          []*discordgo.PermissionOverwrite `json:"permission_overwrites,omitempty"`
	AvailableTags                 []discordgo.ForumTag             `json:"available_tags,omitempty"`
	DefaultReactionEmoji          discordgo.ForumDefaultReaction   `json:"default_reaction_emoji,omitempty"`
	DefaultThreadRateLimitPerUser int                              `json:"default_thread_rate_limit_per_user,omitempty"`
	DefaultSortOrder              *discordgo.ForumSortOrderType    `json:"default_sort_order,omitempty"`
	DefaultForumLayout            discordgo.ForumLayout            `json:"default_forum_layout,omitempty"`
}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	return registry.Register(&bot.Command{
		Name:        "backup-panel",
		Description: "Administra el respaldo estructural de este servidor 🗝️",
		Category:    "utilidad",
		GuildOnly:   true,
		DefaultPerm: discordgo.PermissionManageGuild,
		Handler:     c.handlePanel,
	})
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("backup", handleComponent)
}

func (c *Cog) handlePanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.GuildID == "" {
		return errors.New("este panel solo funciona dentro de un servidor")
	}
	if env == nil || env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}
	return sendPanel(ctx, session, interaction, env, "Elige una acción. La restauración reemplaza canales y roles recreables.")
}

func sendPanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, status string) error {
	panelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	backup, err := env.Store.GetGuildBackup(panelCtx, interaction.GuildID)
	if err != nil && !errors.Is(err, storage.ErrGuildBackupNotFound) {
		return fmt.Errorf("consultar respaldo del servidor: %w", err)
	}
	embed := panelEmbed(env, backup, err == nil, status)
	components := panelComponents(interaction.GuildID)
	if interaction.Type == discordgo.InteractionApplicationCommand {
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
	_, err = discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	return err
}

func panelEmbed(env *bot.Env, backup storage.GuildBackup, exists bool, status string) *discordgo.MessageEmbed {
	state := "No hay un respaldo guardado."
	if exists {
		state = fmt.Sprintf("Creado <t:%d:R> por <@%s> · %d bytes.", backup.CreatedAt.Unix(), backup.CreatedBy, len(backup.Snapshot))
	}
	embed := &discordgo.MessageEmbed{
		Title:       "Archivo de la casa",
		Description: status,
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Respaldo actual", Value: state, Inline: false},
			{Name: "Incluye", Value: "Roles, canales, categorías, permisos, posiciones y ajustes compatibles.", Inline: false},
			{Name: "No incluye", Value: "Mensajes, miembros, asignaciones de roles, hilos, integraciones ni iconos de rol.", Inline: false},
			{Name: "Restauración", Value: "Solo el dueño del servidor puede confirmarla. Reemplaza la estructura recreable actual.", Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Un respaldo por servidor · la creación no sobrescribe el existente"},
	}
	env.Theme.BrandEmbed(embed)
	return embed
}

func panelComponents(guildID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Crear respaldo", Style: discordgo.PrimaryButton, CustomID: componentID(guildID, "create")},
			discordgo.Button{Label: "Cargar respaldo", Style: discordgo.DangerButton, CustomID: componentID(guildID, "restore")},
			discordgo.Button{Label: "Eliminar respaldo", Style: discordgo.SecondaryButton, CustomID: componentID(guildID, "delete")},
		}},
	}
}

func handleComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 3 || parts[0] != "backup" || parts[1] != interaction.GuildID {
		return respondEphemeral(session, interaction, env, "Este panel pertenece a otro servidor.")
	}
	if env == nil || env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}

	action := parts[2]
	switch action {
	case "create", "delete":
		if !hasManageGuild(interaction) {
			return respondEphemeral(session, interaction, env, "Necesitas Administrar servidor para realizar esa acción.")
		}
	case "restore":
		owner, err := isGuildOwner(session, interaction)
		if err != nil {
			return err
		}
		if !owner {
			return respondEphemeral(session, interaction, env, "Solo el dueño del servidor puede cargar un respaldo.")
		}
		return showConfirmation(session, interaction, env, "restore")
	case "restore-confirm":
		owner, err := isGuildOwner(session, interaction)
		if err != nil {
			return err
		}
		if !owner {
			return respondEphemeral(session, interaction, env, "Solo el dueño del servidor puede cargar un respaldo.")
		}
	case "delete-confirm":
		if !hasManageGuild(interaction) {
			return respondEphemeral(session, interaction, env, "Necesitas Administrar servidor para eliminar el respaldo.")
		}
	case "cancel":
		return respondUpdate(session, interaction, "Acción cancelada. La estructura del servidor permanece intacta.", nil)
	default:
		return respondEphemeral(session, interaction, env, "Esta acción ya no está disponible.")
	}
	if action == "create" || action == "restore-confirm" || action == "delete-confirm" {
		unlock, acquired := tryLockGuild(interaction.GuildID)
		if !acquired {
			return respondEphemeral(session, interaction, env, "Ya hay una operación de respaldo en curso para este servidor.")
		}
		defer unlock()
	}

	switch action {
	case "create":
		if err := deferComponentUpdate(session, interaction); err != nil {
			return err
		}
		status, err := createBackup(ctx, session, interaction, env)
		if err != nil {
			env.Log.Error("no se pudo crear el respaldo del servidor", "guild_id", interaction.GuildID, "err", err)
			status = "No se pudo crear el respaldo. Comprueba los permisos del bot y revisa los registros."
			if errors.Is(err, storage.ErrGuildBackupExists) {
				status = "Ya existe un respaldo vigente. Elimínalo antes de crear otro."
			}
		}
		return sendPanel(ctx, session, interaction, env, status)
	case "restore-confirm":
		if err := deferComponentUpdate(session, interaction); err != nil {
			return err
		}
		if err := restoreBackup(ctx, session, interaction, env); err != nil {
			env.Log.Error("no se pudo restaurar por completo el servidor", "guild_id", interaction.GuildID, "err", err)
			return sendPanel(ctx, session, interaction, env, "La restauración no terminó. La estructura puede haber quedado parcialmente modificada; revisa los permisos y registros.")
		}
		return sendPanel(ctx, session, interaction, env, "La estructura del servidor se reconstruyó desde el respaldo.")
	case "delete", "delete-confirm":
		if action == "delete" {
			return showConfirmation(session, interaction, env, "delete")
		}
		if err := deferComponentUpdate(session, interaction); err != nil {
			return err
		}
		if err := env.Store.DeleteGuildBackup(ctx, interaction.GuildID); err != nil {
			if errors.Is(err, storage.ErrGuildBackupNotFound) {
				return sendPanel(ctx, session, interaction, env, "No hay un respaldo guardado para eliminar.")
			}
			env.Log.Error("no se pudo eliminar el respaldo", "guild_id", interaction.GuildID, "err", err)
			return sendPanel(ctx, session, interaction, env, "No se pudo eliminar el respaldo. Revisa los registros del bot.")
		}
		return sendPanel(ctx, session, interaction, env, "Respaldo eliminado. La estructura del servidor no fue modificada.")
	}
	return nil
}

func showConfirmation(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, action string) error {
	title, description, confirmLabel, confirmStyle := "Eliminar respaldo", "Esta acción elimina el archivo guardado. No cambia canales ni roles del servidor.", "Sí, eliminar respaldo", discordgo.DangerButton
	confirmAction := "delete-confirm"
	if action == "restore" {
		title = "Confirmar restauración"
		description = "Se eliminarán los canales y roles recreables actuales y se crearán los del respaldo. No se restauran mensajes, miembros ni asignaciones de roles. Discord puede dejar cambios parciales si alguna operación falla."
		confirmLabel = "Sí, reemplazar estructura"
		confirmAction = "restore-confirm"
	}
	embed := &discordgo.MessageEmbed{Title: title, Description: description, Color: env.Theme.Primary}
	env.Theme.BrandEmbed(embed)
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: confirmLabel, Style: confirmStyle, CustomID: componentID(interaction.GuildID, confirmAction)},
			discordgo.Button{Label: "Cancelar", Style: discordgo.SecondaryButton, CustomID: componentID(interaction.GuildID, "cancel")},
		}},
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func createBackup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) (string, error) {
	snapshot, err := captureSnapshot(ctx, session, interaction.GuildID, env.Store)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("serializar estructura del servidor: %w", err)
	}
	if len(data) > maxSnapshotSize {
		return "", fmt.Errorf("el respaldo supera el límite de %d bytes", maxSnapshotSize)
	}
	err = env.Store.CreateGuildBackup(ctx, storage.GuildBackup{
		GuildID:   interaction.GuildID,
		CreatedBy: interactionUserID(interaction),
		Snapshot:  data,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Respaldo creado: **%d roles** y **%d canales**.", len(snapshot.Roles), len(snapshot.Channels)), nil
}

func captureSnapshot(ctx context.Context, session *discordgo.Session, guildID string, store *storage.Store) (guildSnapshot, error) {
	roles, err := session.GuildRoles(guildID)
	if err != nil {
		return guildSnapshot{}, fmt.Errorf("consultar roles del servidor: %w", err)
	}
	channels, err := session.GuildChannels(guildID)
	if err != nil {
		return guildSnapshot{}, fmt.Errorf("consultar canales del servidor: %w", err)
	}
	guild, err := session.Guild(guildID)
	if err != nil {
		return guildSnapshot{}, fmt.Errorf("consultar ajustes del servidor: %w", err)
	}
	if guild == nil {
		return guildSnapshot{}, errors.New("Discord no devolvió los ajustes del servidor")
	}
	botChannels, err := store.GetGuildBackupChannelSettings(ctx, guildID)
	if err != nil {
		return guildSnapshot{}, fmt.Errorf("consultar canales configurados del bot: %w", err)
	}
	snapshot := guildSnapshot{
		Version: snapshotVersion,
		ServerChannels: serverChannelSettings{
			AFKChannelID: guild.AfkChannelID, SystemChannelID: guild.SystemChannelID,
			RulesChannelID: guild.RulesChannelID, PublicUpdatesChannelID: guild.PublicUpdatesChannelID,
		},
		BotChannels: botChannels,
	}
	for _, role := range roles {
		if role == nil || role.ID == guildID || role.Managed {
			continue
		}
		snapshot.Roles = append(snapshot.Roles, roleSnapshot{
			ID: role.ID, Name: role.Name, Color: role.Color, Position: role.Position,
			Permissions: role.Permissions, Hoist: role.Hoist, Mentionable: role.Mentionable,
			UnicodeEmoji: role.UnicodeEmoji,
		})
	}
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		if !supportedChannelType(channel.Type) {
			return guildSnapshot{}, fmt.Errorf("el canal %q usa un tipo que la API no permite recrear (%d)", channel.Name, channel.Type)
		}
		snapshot.Channels = append(snapshot.Channels, channelSnapshot{
			ID: channel.ID, Name: channel.Name, Type: channel.Type, Topic: channel.Topic,
			Position: channel.Position, Bitrate: channel.Bitrate, UserLimit: channel.UserLimit,
			RateLimitPerUser: channel.RateLimitPerUser, ParentID: channel.ParentID, NSFW: channel.NSFW,
			PermissionOverwrites:          cloneOverwrites(channel.PermissionOverwrites),
			AvailableTags:                 cloneForumTags(channel.AvailableTags),
			DefaultReactionEmoji:          channel.DefaultReactionEmoji,
			DefaultThreadRateLimitPerUser: channel.DefaultThreadRateLimitPerUser,
			DefaultSortOrder:              channel.DefaultSortOrder,
			DefaultForumLayout:            channel.DefaultForumLayout,
		})
	}
	if err := validateSnapshot(snapshot); err != nil {
		return guildSnapshot{}, err
	}
	return snapshot, nil
}

func restoreBackup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	backup, err := env.Store.GetGuildBackup(ctx, interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar el respaldo guardado: %w", err)
	}
	if len(backup.Snapshot) > maxSnapshotSize {
		return errors.New("el respaldo guardado excede el tamaño permitido")
	}
	var snapshot guildSnapshot
	if err := json.Unmarshal(backup.Snapshot, &snapshot); err != nil {
		return fmt.Errorf("leer el formato del respaldo: %w", err)
	}
	if err := validateSnapshot(snapshot); err != nil {
		return fmt.Errorf("validar el respaldo: %w", err)
	}
	guild, err := session.Guild(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("consultar servidor: %w", err)
	}
	roles, err := session.GuildRoles(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("consultar jerarquía de roles: %w", err)
	}
	channels, err := session.GuildChannels(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("consultar canales actuales: %w", err)
	}
	if session.State == nil || session.State.User == nil {
		return errors.New("no se pudo identificar al bot en el servidor")
	}
	botMember, err := session.GuildMember(interaction.GuildID, session.State.User.ID)
	if err != nil {
		return fmt.Errorf("consultar permisos del bot: %w", err)
	}
	if err := checkRestorePermissions(guild, roles, botMember, snapshot); err != nil {
		return err
	}
	channelIDs := make(map[string]string, len(snapshot.Channels))
	if err := replaceGuildStructure(interaction.GuildID, session, roles, channels, snapshot, channelIDs); err != nil {
		return err
	}
	if err := restoreServerChannelSettings(interaction.GuildID, session, snapshot.ServerChannels, channelIDs); err != nil {
		return err
	}
	botChannels := remapBotChannelSettings(snapshot.BotChannels, channelIDs)
	botChannels.GuildID = interaction.GuildID
	if err := env.Store.RestoreGuildBackupChannelSettings(ctx, botChannels); err != nil {
		return err
	}
	return nil
}

func checkRestorePermissions(guild *discordgo.Guild, roles []*discordgo.Role, member *discordgo.Member, snapshot guildSnapshot) error {
	if guild == nil || member == nil {
		return errors.New("no se pudo verificar la jerarquía y los permisos del bot")
	}
	roleByID := make(map[string]*discordgo.Role, len(roles))
	for _, role := range roles {
		if role != nil {
			roleByID[role.ID] = role
		}
	}
	everyoneRole := roleByID[guild.ID]
	if everyoneRole == nil {
		return errors.New("no se pudo verificar la jerarquía y los permisos del bot")
	}
	permissions := everyoneRole.Permissions
	highestPosition := 0
	for _, roleID := range member.Roles {
		role := roleByID[roleID]
		if role == nil {
			continue
		}
		permissions |= role.Permissions
		if role.Position > highestPosition {
			highestPosition = role.Position
		}
	}
	if permissions&discordgo.PermissionAdministrator != discordgo.PermissionAdministrator &&
		permissions&(discordgo.PermissionManageRoles|discordgo.PermissionManageChannels) != (discordgo.PermissionManageRoles|discordgo.PermissionManageChannels) {
		return errors.New("el bot necesita Administrar roles y Administrar canales para restaurar")
	}
	for _, role := range roles {
		if role != nil && role.ID != guild.ID && !role.Managed && role.Position >= highestPosition {
			return fmt.Errorf("el rol %q está en la misma altura o por encima del rol más alto del bot", role.Name)
		}
	}
	for _, role := range snapshot.Roles {
		if role.Position >= highestPosition {
			return fmt.Errorf("el respaldo contiene el rol %q por encima de la jerarquía del bot", role.Name)
		}
	}
	if hasServerChannelSettings(snapshot.ServerChannels) &&
		permissions&discordgo.PermissionAdministrator != discordgo.PermissionAdministrator &&
		permissions&discordgo.PermissionManageGuild == 0 {
		return errors.New("el bot necesita Administrar servidor para restaurar los canales especiales")
	}
	return nil
}

func replaceGuildStructure(guildID string, session *discordgo.Session, currentRoles []*discordgo.Role, currentChannels []*discordgo.Channel, snapshot guildSnapshot, channelIDs map[string]string) error {
	for _, channel := range currentChannels {
		if channel == nil {
			continue
		}
		if _, err := session.ChannelDelete(channel.ID); err != nil {
			return fmt.Errorf("eliminar canal %q: %w", channel.Name, err)
		}
	}
	for _, role := range currentRoles {
		if role == nil || role.ID == guildID || role.Managed {
			continue
		}
		if err := session.GuildRoleDelete(guildID, role.ID); err != nil {
			return fmt.Errorf("eliminar rol %q: %w", role.Name, err)
		}
	}

	roleIDs := make(map[string]string, len(snapshot.Roles))
	sortedRoles := append([]roleSnapshot(nil), snapshot.Roles...)
	sort.Slice(sortedRoles, func(i, j int) bool { return sortedRoles[i].Position < sortedRoles[j].Position })
	reordered := make([]*discordgo.Role, 0, len(sortedRoles))
	for _, role := range sortedRoles {
		color, hoist, mentionable := role.Color, role.Hoist, role.Mentionable
		permissions, unicodeEmoji := role.Permissions, role.UnicodeEmoji
		created, err := session.GuildRoleCreate(guildID, &discordgo.RoleParams{
			Name: role.Name, Color: &color, Hoist: &hoist, Permissions: &permissions,
			Mentionable: &mentionable, UnicodeEmoji: optionalString(unicodeEmoji),
		})
		if err != nil {
			return fmt.Errorf("crear rol %q: %w", role.Name, err)
		}
		roleIDs[role.ID] = created.ID
		reordered = append(reordered, &discordgo.Role{ID: created.ID, Position: role.Position})
	}
	if len(reordered) > 0 {
		if _, err := session.GuildRoleReorder(guildID, reordered); err != nil {
			return fmt.Errorf("restaurar posiciones de roles: %w", err)
		}
	}

	for _, channel := range snapshot.Channels {
		if channel.Type != discordgo.ChannelTypeGuildCategory {
			continue
		}
		created, err := createChannel(session, guildID, channel, "", roleIDs)
		if err != nil {
			return fmt.Errorf("crear categoría %q: %w", channel.Name, err)
		}
		channelIDs[channel.ID] = created.ID
	}
	for _, channel := range snapshot.Channels {
		if channel.Type == discordgo.ChannelTypeGuildCategory {
			continue
		}
		parentID := ""
		if channel.ParentID != "" {
			parentID = channelIDs[channel.ParentID]
			if parentID == "" {
				return fmt.Errorf("no se encontró la categoría del canal %q", channel.Name)
			}
		}
		created, err := createChannel(session, guildID, channel, parentID, roleIDs)
		if err != nil {
			return fmt.Errorf("crear canal %q: %w", channel.Name, err)
		}
		channelIDs[channel.ID] = created.ID
	}
	return nil
}

func restoreServerChannelSettings(guildID string, session *discordgo.Session, settings serverChannelSettings, channelIDs map[string]string) error {
	params := &discordgo.GuildParams{}
	changed := false
	if id := channelIDs[settings.AFKChannelID]; settings.AFKChannelID != "" && id != "" {
		params.AfkChannelID = id
		changed = true
	}
	if id := channelIDs[settings.SystemChannelID]; settings.SystemChannelID != "" && id != "" {
		params.SystemChannelID = id
		changed = true
	}
	if id := channelIDs[settings.RulesChannelID]; settings.RulesChannelID != "" && id != "" {
		params.RulesChannelID = id
		changed = true
	}
	if id := channelIDs[settings.PublicUpdatesChannelID]; settings.PublicUpdatesChannelID != "" && id != "" {
		params.PublicUpdatesChannelID = id
		changed = true
	}
	if !changed {
		return nil
	}
	if _, err := session.GuildEdit(guildID, params); err != nil {
		return fmt.Errorf("restaurar canales especiales del servidor: %w", err)
	}
	return nil
}

func hasServerChannelSettings(settings serverChannelSettings) bool {
	return settings.AFKChannelID != "" || settings.SystemChannelID != "" ||
		settings.RulesChannelID != "" || settings.PublicUpdatesChannelID != ""
}

func remapBotChannelSettings(settings storage.GuildBackupChannelSettings, channelIDs map[string]string) storage.GuildBackupChannelSettings {
	settings.LogChannelID = channelIDs[settings.LogChannelID]
	settings.WelcomeChannel = channelIDs[settings.WelcomeChannel]
	settings.AuditLogChannelID = channelIDs[settings.AuditLogChannelID]
	settings.SupportChannelID = channelIDs[settings.SupportChannelID]
	settings.HelpChannelID = channelIDs[settings.HelpChannelID]
	settings.ConfessionChannelID = channelIDs[settings.ConfessionChannelID]
	return settings
}

func createChannel(session *discordgo.Session, guildID string, channel channelSnapshot, parentID string, roleIDs map[string]string) (*discordgo.Channel, error) {
	created, err := session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: channel.Name, Type: channel.Type, Topic: channel.Topic, Bitrate: channel.Bitrate,
		UserLimit: channel.UserLimit, RateLimitPerUser: channel.RateLimitPerUser,
		Position: channel.Position, ParentID: parentID, NSFW: channel.NSFW,
		PermissionOverwrites: remapOverwrites(channel.PermissionOverwrites, roleIDs),
	})
	if err != nil {
		return nil, err
	}
	if len(channel.AvailableTags) == 0 && channel.DefaultSortOrder == nil &&
		channel.DefaultForumLayout == discordgo.ForumLayoutNotSet &&
		channel.DefaultThreadRateLimitPerUser == 0 &&
		channel.DefaultReactionEmoji.EmojiID == "" && channel.DefaultReactionEmoji.EmojiName == "" {
		return created, nil
	}
	edit := &discordgo.ChannelEdit{}
	if len(channel.AvailableTags) > 0 {
		tags := cloneForumTags(channel.AvailableTags)
		for i := range tags {
			tags[i].ID = ""
		}
		edit.AvailableTags = &tags
	}
	if channel.DefaultSortOrder != nil {
		sortOrder := *channel.DefaultSortOrder
		edit.DefaultSortOrder = &sortOrder
	}
	if channel.DefaultForumLayout != discordgo.ForumLayoutNotSet {
		layout := channel.DefaultForumLayout
		edit.DefaultForumLayout = &layout
	}
	if channel.DefaultThreadRateLimitPerUser != 0 {
		rateLimit := channel.DefaultThreadRateLimitPerUser
		edit.DefaultThreadRateLimitPerUser = &rateLimit
	}
	if channel.DefaultReactionEmoji.EmojiID != "" || channel.DefaultReactionEmoji.EmojiName != "" {
		reaction := channel.DefaultReactionEmoji
		edit.DefaultReactionEmoji = &reaction
	}
	if _, err := session.ChannelEdit(created.ID, edit); err != nil {
		return nil, fmt.Errorf("restaurar ajustes avanzados: %w", err)
	}
	return created, nil
}

func validateSnapshot(snapshot guildSnapshot) error {
	if snapshot.Version != snapshotVersion {
		return fmt.Errorf("versión del respaldo no compatible: %d", snapshot.Version)
	}
	if len(snapshot.Roles) > maxBackupRoles || len(snapshot.Channels) > maxBackupRooms {
		return errors.New("el respaldo excede el número máximo de roles o canales")
	}
	roleIDs := make(map[string]struct{}, len(snapshot.Roles))
	for _, role := range snapshot.Roles {
		if role.ID == "" || strings.TrimSpace(role.Name) == "" {
			return errors.New("el respaldo contiene un rol sin ID o nombre")
		}
		if _, exists := roleIDs[role.ID]; exists {
			return errors.New("el respaldo contiene IDs de rol duplicados")
		}
		roleIDs[role.ID] = struct{}{}
	}
	channelIDs := make(map[string]channelSnapshot, len(snapshot.Channels))
	for _, channel := range snapshot.Channels {
		if channel.ID == "" || strings.TrimSpace(channel.Name) == "" || !supportedChannelType(channel.Type) {
			return fmt.Errorf("el respaldo contiene un canal inválido (%q)", channel.Name)
		}
		if _, exists := channelIDs[channel.ID]; exists {
			return errors.New("el respaldo contiene IDs de canal duplicados")
		}
		channelIDs[channel.ID] = channel
	}
	for _, channel := range snapshot.Channels {
		if channel.ParentID == "" {
			continue
		}
		parent, exists := channelIDs[channel.ParentID]
		if !exists || parent.Type != discordgo.ChannelTypeGuildCategory {
			return fmt.Errorf("el canal %q referencia una categoría ausente", channel.Name)
		}
	}
	for _, id := range []string{
		snapshot.ServerChannels.AFKChannelID, snapshot.ServerChannels.SystemChannelID,
		snapshot.ServerChannels.RulesChannelID, snapshot.ServerChannels.PublicUpdatesChannelID,
	} {
		if id != "" {
			if _, exists := channelIDs[id]; !exists {
				return fmt.Errorf("el respaldo referencia un canal especial ausente: %s", id)
			}
		}
	}
	return nil
}

func supportedChannelType(channelType discordgo.ChannelType) bool {
	switch channelType {
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildVoice,
		discordgo.ChannelTypeGuildCategory, discordgo.ChannelTypeGuildNews,
		discordgo.ChannelTypeGuildStageVoice, discordgo.ChannelTypeGuildForum,
		discordgo.ChannelTypeGuildMedia:
		return true
	default:
		return false
	}
}

func remapOverwrites(overwrites []*discordgo.PermissionOverwrite, roleIDs map[string]string) []*discordgo.PermissionOverwrite {
	if len(overwrites) == 0 {
		return nil
	}
	result := cloneOverwrites(overwrites)
	for _, overwrite := range result {
		if overwrite.Type == discordgo.PermissionOverwriteTypeRole {
			if replacement, ok := roleIDs[overwrite.ID]; ok {
				overwrite.ID = replacement
			}
		}
	}
	return result
}

func cloneOverwrites(overwrites []*discordgo.PermissionOverwrite) []*discordgo.PermissionOverwrite {
	if len(overwrites) == 0 {
		return nil
	}
	result := make([]*discordgo.PermissionOverwrite, 0, len(overwrites))
	for _, overwrite := range overwrites {
		if overwrite == nil {
			continue
		}
		copy := *overwrite
		result = append(result, &copy)
	}
	return result
}

func cloneForumTags(tags []discordgo.ForumTag) []discordgo.ForumTag {
	if len(tags) == 0 {
		return nil
	}
	result := append([]discordgo.ForumTag(nil), tags...)
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func hasManageGuild(interaction *discordgo.InteractionCreate) bool {
	if interaction.Member == nil {
		return false
	}
	permissions := interaction.Member.Permissions
	return permissions&discordgo.PermissionAdministrator != 0 || permissions&discordgo.PermissionManageGuild != 0
}

func isGuildOwner(session *discordgo.Session, interaction *discordgo.InteractionCreate) (bool, error) {
	guild, err := session.Guild(interaction.GuildID)
	if err != nil {
		return false, fmt.Errorf("consultar dueño del servidor: %w", err)
	}
	return guild.OwnerID != "" && interactionUserID(interaction) == guild.OwnerID, nil
}

func interactionUserID(interaction *discordgo.InteractionCreate) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return interaction.Member.User.ID
	}
	if interaction.User != nil {
		return interaction.User.ID
	}
	return ""
}

func componentID(guildID, action string) string {
	return "backup:" + guildID + ":" + action
}

func tryLockGuild(guildID string) (func(), bool) {
	value, _ := guildOperations.LoadOrStore(guildID, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	if !lock.TryLock() {
		return nil, false
	}
	return lock.Unlock, true
}

func deferComponentUpdate(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

func respondEphemeral(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, content string) error {
	data := &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral}
	if env != nil && env.Theme != nil {
		data.Content = ""
		data.Embeds = []*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Una pausa, por favor", content)}
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func respondUpdate(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: content, Components: components},
	})
}
