package audit

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/ui"
)

const (
	auditPermission = discordgo.PermissionManageMessages
	staffPermission = discordgo.PermissionBanMembers | discordgo.PermissionKickMembers |
		discordgo.PermissionModerateMembers | discordgo.PermissionManageMessages | discordgo.PermissionManageGuild
)

type Cog struct {
	cooldowns sync.Map
}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	if err := registry.AddCommand(&bot.Command{
		Category:    "auditoria",
		DefaultPerm: auditPermission,
		AppCmd: &discordgo.ApplicationCommand{
			Name:                     "logs",
			Description:              "El diario del Lux registra los sucesos del servidor 📜",
			DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageMessages),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "configurar",
					Description: "Elige el canal y si se archivará el texto de mensajes borrados",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionChannel,
							Name:         "canal",
							Description:  "Canal de texto para el diario",
							Required:     true,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "contenido",
							Description: "Archivar contenido borrado (desactivado por defecto)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "estado",
					Description: "Consulta la configuración y los límites del diario",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "prueba",
					Description: "Envía una entrada de prueba al canal configurado",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "eliminados",
					Description: "Consulta hasta cinco mensajes borrados de las últimas horas",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "horas",
							Description: "Antigüedad máxima (1-24 horas, por defecto 6)",
							Required:    false,
							MinValue:    float64Ptr(1),
							MaxValue:    24,
						},
					},
				},
			},
		},
		LegacyHandler: c.handleLogs,
	}); err != nil {
		return err
	}

	if err := registry.AddCommand(&bot.Command{
		Category: "comunidad",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "soporte",
			Description: "Llama a la guardia del Lux cuando necesites ayuda 🛎️",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "llamar",
					Description: "Envía una petición al staff (máximo una cada cinco minutos)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "motivo",
							Description: "Explica brevemente qué necesitas",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
			},
		},
		LegacyHandler: c.handleSupport,
	}); err != nil {
		return err
	}

	if err := registry.Register(&bot.Command{
		Name:        "anuncio",
		Description: "Lucifer redacta un borrador de anuncio para que lo revises 🧠",
		Category:    "utilidad",
		DefaultPerm: discordgo.PermissionManageMessages,
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "Qué debe comunicar el anuncio (máximo 1200 caracteres)",
				Required:    true,
				MaxLength:   1200,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tono",
				Description: "Estilo del texto",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Chill", Value: "chill"},
					{Name: "Elegante", Value: "elegante"},
					{Name: "Urgente", Value: "urgente"},
				},
			},
		},
		Handler: c.handleAnnouncement,
	}); err != nil {
		return err
	}

	return registry.Register(&bot.Command{
		Name:        "staff",
		Description: "Consulta el ranking verificable de acciones de moderación 🏆",
		Category:    "moderacion",
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "periodo",
				Description: "Ventana del ranking",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Últimas 24 horas", Value: "24h"},
					{Name: "Últimos 7 días", Value: "7d"},
					{Name: "Últimos 30 días", Value: "30d"},
				},
			},
		},
		Handler: c.handleStaffRanking,
	})
}

func (c *Cog) handleLogs(ctx *bot.Context) {
	if !hasPermission(ctx.Interaction, auditPermission) {
		respond(ctx, "⛔ Solo el staff del servidor puede consultar o cambiar el diario.", true)
		return
	}
	data := ctx.Interaction.ApplicationCommandData()
	if len(data.Options) == 0 {
		respond(ctx, "Elige una acción para el diario del Lux.", true)
		return
	}
	if err := ctx.Defer(true); err != nil {
		ctx.Env.Log.Error("no se pudo diferir el comando de auditoría", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}

	switch option := data.Options[0]; option.Name {
	case "configurar":
		c.configureLogs(ctx, option)
	case "estado":
		c.showLogStatus(ctx)
	case "prueba":
		c.testLogs(ctx)
	case "eliminados":
		c.showDeletedMessages(ctx, option)
	default:
		respond(ctx, "No reconozco esa acción del diario.", true)
	}
}

func (c *Cog) configureLogs(ctx *bot.Context, option *discordgo.ApplicationCommandInteractionDataOption) {
	channelOption := option.GetOption("canal")
	if channelOption == nil {
		respond(ctx, "Debes elegir un canal de texto.", true)
		return
	}
	channel := channelOption.ChannelValue(ctx.Session)
	if channel == nil || channel.GuildID != ctx.Interaction.GuildID || channel.Type != discordgo.ChannelTypeGuildText {
		respond(ctx, "Ese canal no pertenece a este servidor o no es un canal de texto.", true)
		return
	}
	settings, err := ctx.Env.Store.GetAuditSettings(ctx.Context, ctx.Interaction.GuildID)
	if err != nil {
		respond(ctx, "No pude leer la configuración del diario. Inténtalo más tarde.", true)
		ctx.Env.Log.Error("falló la lectura de configuración del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	settings.GuildID = ctx.Interaction.GuildID
	settings.LogChannelID = channel.ID
	if contentOption := option.GetOption("contenido"); contentOption != nil {
		settings.IncludeDeletedContent = contentOption.BoolValue()
	}
	if err := ctx.Env.Store.SaveAuditSettings(ctx.Context, settings); err != nil {
		respond(ctx, "No pude guardar la configuración del diario.", true)
		ctx.Env.Log.Error("falló el guardado de configuración del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	if ctx.Env.Auditor != nil {
		ctx.Env.Auditor.UpdateSettings(settings)
	}
	visibility := "desactivado; los mensajes borrados conservarán solo autor, canal e ID"
	if settings.IncludeDeletedContent {
		visibility = "activado; el texto de mensajes borrados se conservará durante siete días"
	}
	respond(ctx, fmt.Sprintf("📜 El Diario del Inframundo escribirá en <#%s>.\nArchivo de contenido borrado: **%s**.", channel.ID, visibility), true)
}

func (c *Cog) showLogStatus(ctx *bot.Context) {
	settings, err := ctx.Env.Store.GetAuditSettings(ctx.Context, ctx.Interaction.GuildID)
	if err != nil {
		respond(ctx, "No pude consultar el estado del diario.", true)
		ctx.Env.Log.Error("falló la consulta de estado del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	channel := "*(sin configurar)*"
	if settings.LogChannelID != "" {
		channel = "<#" + settings.LogChannelID + ">"
	}
	content := "Desactivado (solo metadatos de mensajes eliminados)"
	if settings.IncludeDeletedContent {
		content = "Activado; retención máxima de siete días"
	}
	respond(ctx, fmt.Sprintf("📜 **Expediente del diario**\nCanal: %s\nTexto de mensajes borrados: %s\nEventos: mensajes editados/eliminados, voz, miembros, bans, canales, hilos y roles.", channel, content), true)
}

func (c *Cog) testLogs(ctx *bot.Context) {
	settings, err := ctx.Env.Store.GetAuditSettings(ctx.Context, ctx.Interaction.GuildID)
	if err != nil {
		respond(ctx, "No pude consultar el canal de auditoría.", true)
		ctx.Env.Log.Error("falló la consulta del canal del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	if settings.LogChannelID == "" {
		respond(ctx, "Primero configura el canal con `/logs configurar`.", true)
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🕯️ Prueba del Diario del Inframundo",
		Description: "El Lux está conectado y puede publicar aquí.",
		Color:       0x7B1FA2,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	ctx.Env.Theme.BrandEmbed(embed)
	_, err = discordutil.ChannelMessageSendEmbed(ctx.Session, settings.LogChannelID, embed)
	if err != nil {
		respond(ctx, "No pude escribir en ese canal. Revisa que tenga permiso para ver el canal, enviar mensajes y adjuntar enlaces.", true)
		ctx.Env.Log.Error("falló la prueba del canal del diario", "guild_id", ctx.Interaction.GuildID, "channel_id", settings.LogChannelID, "err", err)
		return
	}
	respond(ctx, "✅ Lucifer dejó una entrada de prueba en el diario.", true)
}

func (c *Cog) showDeletedMessages(ctx *bot.Context, option *discordgo.ApplicationCommandInteractionDataOption) {
	hours := int64(6)
	if hoursOption := option.GetOption("horas"); hoursOption != nil {
		hours = hoursOption.IntValue()
	}
	if hours < 1 || hours > 24 {
		respond(ctx, "El periodo debe estar entre una y 24 horas.", true)
		return
	}
	messages, err := ctx.Env.Store.RecentDeletedMessages(ctx.Context, ctx.Interaction.GuildID, time.Now().Add(-time.Duration(hours)*time.Hour), 5)
	if err != nil {
		respond(ctx, "No pude consultar los mensajes eliminados.", true)
		ctx.Env.Log.Error("falló la consulta de mensajes eliminados", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	if len(messages) == 0 {
		respond(ctx, fmt.Sprintf("🗑️ No hay mensajes eliminados registrados en las últimas %d horas.", hours), true)
		return
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(messages))
	for index, message := range messages {
		author := "Autor desconocido"
		if message.AuthorID != "" {
			author = "<@" + message.AuthorID + ">"
		}
		content := "_El contenido no se archivó. Un administrador puede activarlo con `/logs configurar`._"
		if message.Content != "" {
			content = truncate(message.Content, 800)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%d. %s · <#%s>", index+1, message.DeletedAt.Local().Format("15:04:05"), message.ChannelID),
			Value:  fmt.Sprintf("%s\n%s", author, content),
			Inline: false,
		})
	}
	if err := ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       "🗑️ Últimos mensajes borrados",
			Description: fmt.Sprintf("Hasta cinco registros de las últimas %d horas. Solo visible para ti y personal autorizado.", hours),
			Color:       0x7B1FA2,
			Fields:      fields,
		}},
		Flags: discordgo.MessageFlagsEphemeral,
	}); err != nil {
		ctx.Env.Log.Error("no se pudo responder con los mensajes eliminados", "guild_id", ctx.Interaction.GuildID, "err", err)
	}
}

func (c *Cog) handleSupport(ctx *bot.Context) {
	data := ctx.Interaction.ApplicationCommandData()
	if len(data.Options) == 0 {
		respond(ctx, "Usa `llamar` para avisar a la guardia del Lux.", true)
		return
	}
	if err := ctx.Defer(true); err != nil {
		ctx.Env.Log.Error("no se pudo diferir el comando de soporte", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	option := data.Options[0]
	switch option.Name {
	case "llamar":
		c.callSupport(ctx, option)
	default:
		respond(ctx, "No reconozco esa acción de soporte.", true)
	}
}

func (c *Cog) callSupport(ctx *bot.Context, option *discordgo.ApplicationCommandInteractionDataOption) {
	guildID := ctx.Interaction.GuildID
	if ctx.Interaction.Member == nil || ctx.Interaction.Member.User == nil {
		respond(ctx, "Este comando solo está disponible dentro de un servidor.", true)
		return
	}
	userID := ctx.Interaction.Member.User.ID
	key := guildID + ":" + userID
	now := time.Now()
	if lastValue, ok := c.cooldowns.Load(key); ok && now.Sub(lastValue.(time.Time)) < 5*time.Minute {
		remaining := 5*time.Minute - now.Sub(lastValue.(time.Time))
		respond(ctx, fmt.Sprintf("Lucifer ya recibió tu aviso. Espera %s antes de volver a llamar.", remaining.Round(time.Second)), true)
		return
	}

	settings, err := ctx.Env.Store.GetAuditSettings(ctx.Context, guildID)
	if err != nil {
		respond(ctx, "No pude consultar el canal de soporte.", true)
		ctx.Env.Log.Error("falló la consulta de configuración de soporte", "guild_id", guildID, "err", err)
		return
	}
	if settings.SupportChannelID == "" {
		channel, err := c.createSupportChannel(ctx, guildID)
		if err != nil {
			respond(ctx, "No pude crear el canal privado de soporte. Comprueba los permisos del bot.", true)
			ctx.Env.Log.Error("falló la creación automática del canal de soporte", "guild_id", guildID, "err", err)
			return
		}
		settings.GuildID = guildID
		settings.SupportChannelID = channel.ID
		if err := ctx.Env.Store.SaveAuditSettings(ctx.Context, settings); err != nil {
			_, deleteErr := ctx.Session.ChannelDelete(channel.ID)
			ctx.Env.Log.Error("falló el guardado del canal automático de soporte", "guild_id", guildID, "err", errors.Join(err, deleteErr))
			respond(ctx, "No pude registrar el canal de soporte. Avísale al administrador del bot.", true)
			return
		}
		if ctx.Env.Auditor != nil {
			ctx.Env.Auditor.UpdateSettings(settings)
		}
	}
	reason := "No indicó un motivo."
	if reasonOption := option.GetOption("motivo"); reasonOption != nil && strings.TrimSpace(reasonOption.StringValue()) != "" {
		reason = sanitizeText(reasonOption.StringValue())
	}
	guild, err := ctx.Session.Guild(guildID)
	if err != nil {
		respond(ctx, "No pude identificar al staff de este servidor.", true)
		ctx.Env.Log.Error("falló la lectura de roles para soporte", "guild_id", guildID, "err", err)
		return
	}
	roles := eligibleStaffRoles(guild.Roles, guildID)
	channelPerms, err := ctx.Session.UserChannelPermissions(ctx.Session.State.User.ID, settings.SupportChannelID)
	if err != nil {
		respond(ctx, "No pude comprobar los permisos del canal de soporte.", true)
		ctx.Env.Log.Error("falló la comprobación de permisos del canal de soporte", "guild_id", guildID, "channel_id", settings.SupportChannelID, "err", err)
		return
	}
	allowedRoleIDs := make([]string, 0, len(roles))
	roleMentions := make([]string, 0, len(roles))
	for _, role := range roles {
		if role.Mentionable || channelPerms&discordgo.PermissionMentionEveryone != 0 {
			allowedRoleIDs = append(allowedRoleIDs, role.ID)
			roleMentions = append(roleMentions, "<@&"+role.ID+">")
		}
	}
	mentionLine := "No se encontró un rol de staff mencionable; el aviso queda publicado sin ping."
	if len(roleMentions) > 0 {
		mentionLine = strings.Join(roleMentions, " ")
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🛎️ Una petición llegó a la recepción del Lux",
		Description: fmt.Sprintf("**Solicitante:** <@%s>\n**Canal de origen:** <#%s>\n**Motivo:** %s", userID, ctx.Interaction.ChannelID, truncate(reason, 1000)),
		Color:       0x7B1FA2,
		Timestamp:   now.UTC().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "La guardia debe responder en el canal de origen; las solicitudes no crean tickets."},
	}
	ctx.Env.Theme.BrandEmbed(embed)
	_, err = discordutil.ChannelMessageSendComplex(ctx.Session, settings.SupportChannelID, &discordgo.MessageSend{
		Content: mentionLine,
		Embeds:  []*discordgo.MessageEmbed{embed},
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Roles: allowedRoleIDs,
		},
	})
	if err != nil {
		respond(ctx, "No pude avisar al staff. Comprueba los permisos del canal y vuelve a intentarlo.", true)
		ctx.Env.Log.Error("falló el envío de petición de soporte", "guild_id", guildID, "channel_id", settings.SupportChannelID, "err", err)
		return
	}
	c.cooldowns.Store(key, now)
	c.cooldowns.Range(func(key, value any) bool {
		if now.Sub(value.(time.Time)) > time.Hour {
			c.cooldowns.Delete(key)
		}
		return true
	})
	respond(ctx, fmt.Sprintf("✅ Lucifer avisó al staff. Tu petición se publicó en <#%s>; espera allí su respuesta.", settings.SupportChannelID), true)
}

func (c *Cog) createSupportChannel(ctx *bot.Context, guildID string) (*discordgo.Channel, error) {
	const channelName = "🛎️・guardia-del-lux"
	guild, err := ctx.Session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("consultar roles para soporte: %w", err)
	}
	channels, err := ctx.Session.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("buscar canal privado de soporte: %w", err)
	}
	for _, channel := range channels {
		if channel.Name == channelName && channel.Type == discordgo.ChannelTypeGuildText &&
			isPrivateToEveryone(channel, guildID) {
			return channel, nil
		}
	}
	botID := ctx.Session.State.User.ID
	perms, err := ctx.Session.UserChannelPermissions(botID, ctx.Interaction.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("comprobar permisos para canal privado de soporte: %w", err)
	}
	required := int64(discordgo.PermissionManageChannels | discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks)
	if perms&required != required {
		return nil, errors.New("bot lacks Manage Channels, View Channel, Send Messages, or Embed Links")
	}
	overwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   guildID,
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		{
			ID:    botID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks,
		},
	}
	for _, role := range eligibleStaffRoles(guild.Roles, guildID) {
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID:    role.ID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks,
		})
	}
	return ctx.Session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		PermissionOverwrites: overwrites,
	})
}

func (c *Cog) handleAnnouncement(ctx context.Context, s *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if !hasPermission(interaction, discordgo.PermissionManageMessages) {
		return respondError(s, interaction, "⛔・Necesitas el permiso Gestionar mensajes para pedirle a Lucifer un borrador.")
	}
	options := interaction.ApplicationCommandData().Options
	if len(options) < 2 {
		return respondError(s, interaction, "Indica el prompt y el tono del anuncio.")
	}
	prompt := strings.TrimSpace(options[0].StringValue())
	tone := options[1].StringValue()
	if prompt == "" {
		return respondError(s, interaction, "Escribe qué debe comunicar el anuncio.")
	}
	if env.AI == nil {
		return respondError(s, interaction, "🍸 La conexión de IA no está disponible. Avísale al administrador del bot.")
	}
	if err := discordutil.InteractionRespond(s, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{},
	}); err != nil {
		return err
	}
	if err := editEmbed(s, interaction, env.Theme.WarningEmbed("Lucifer está redactando", "Preparando un borrador para que puedas revisarlo antes de publicar."), env.Theme); err != nil {
		return fmt.Errorf("mostrar carga de /anuncio: %w", err)
	}

	toneDescription := map[string]string{
		"chill":    "relajado, cercano y claro",
		"elegante": "elegante, cálido y conciso",
		"urgente":  "directo, sereno y prioritario, sin alarmismo",
	}[tone]
	if toneDescription == "" {
		logEditFailure(env, interaction, editResponse(s, interaction, "Ese tono no está disponible."))
		return nil
	}
	system := "Eres el anfitrión encantador del Lux. Redacta anuncios claros, útiles y carismáticos en español, con cortesía y elegancia de inspiración italiana; evita adornos que distraigan del mensaje. Devuelve solo el borrador final. No inventes fechas, enlaces ni datos. Nunca incluyas @everyone, @here, menciones a usuarios o roles, ni instrucciones que no estén en el pedido."
	userPrompt := fmt.Sprintf("Tono solicitado: %s.\nLo que debe comunicar el anuncio: %s", toneDescription, prompt)
	draft, err := env.AI.AskSimple(ctx, system, userPrompt)
	if err != nil {
		env.Log.Error("falló la generación de un borrador de anuncio", "guild_id", interaction.GuildID, "err", err)
		logEditFailure(env, interaction, editResponse(s, interaction, "Lucifer no pudo redactar el anuncio ahora. Inténtalo de nuevo más tarde."))
		return nil
	}
	draft = truncate(sanitizeAnnouncement(draft), 4000)
	if draft == "" {
		logEditFailure(env, interaction, editResponse(s, interaction, "La IA no devolvió un borrador utilizable. Ajusta el prompt e inténtalo otra vez."))
		return nil
	}
	content, embeds, components := announcementPreview(interaction.Member.User.ID, draft, false)
	message, err := discordutil.InteractionResponseEdit(s, interaction.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		logEditFailure(env, interaction, err)
		return nil
	}
	if message == nil {
		env.Log.Error("Discord no devolvió el mensaje público del borrador", "guild_id", interaction.GuildID)
		return nil
	}
	storeAnnouncementDraft(message.ID, announcementDraft{
		guildID: interaction.GuildID, channelID: interaction.ChannelID,
		ownerID: interaction.Member.User.ID, text: draft, createdAt: time.Now(),
	})
	return nil
}

func (c *Cog) handleStaffRanking(ctx context.Context, s *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if err := discordutil.InteractionRespond(s, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return err
	}
	period := 7 * 24 * time.Hour
	periodLabel := "7 días"
	options := interaction.ApplicationCommandData().Options
	if len(options) > 0 {
		switch options[0].StringValue() {
		case "24h":
			period, periodLabel = 24*time.Hour, "24 horas"
		case "30d":
			period, periodLabel = 30*24*time.Hour, "30 días"
		case "7d":
			period, periodLabel = 7*24*time.Hour, "7 días"
		}
	}
	entries, err := env.Store.StaffRanking(ctx, interaction.GuildID, time.Now().Add(-period), 10)
	if err != nil {
		env.Log.Error("falló la consulta del ranking del staff", "guild_id", interaction.GuildID, "err", err)
		logEditFailure(env, interaction, editResponse(s, interaction, "No pude consultar el ranking del staff."))
		return nil
	}
	if len(entries) == 0 {
		logEditFailure(env, interaction, editEmbed(s, interaction, &discordgo.MessageEmbed{
			Title:       "🏆 Méritos de la guardia del Lux",
			Description: fmt.Sprintf("No hay acciones registradas en los últimos %s.\n\nEl ranking cuenta bans, expulsiones, silencios y limpiezas realizadas con los comandos de Krypton. No atribuye acciones manuales ni tickets todavía.", periodLabel),
			Color:       0x7B1FA2,
		}, env.Theme))
		return nil
	}
	var table strings.Builder
	table.WriteString("Puesto  Staff                 Ban  Avisos  Kick  Silencio  Limpieza  Total\n")
	for index, entry := range entries {
		table.WriteString(fmt.Sprintf("%-7d <@%s>  %3d  %6d  %4d  %8d  %8d  %5d\n",
			index+1, entry.ActorID, entry.Bans, entry.Warnings, entry.Kicks, entry.Timeouts, entry.Cleans, entry.Total))
	}
	logEditFailure(env, interaction, editEmbed(s, interaction, &discordgo.MessageEmbed{
		Title:       "🏆 Tabla de méritos del Lux",
		Description: fmt.Sprintf("Actividad registrada durante los últimos %s:\n```text\n%s```\n\nCuenta solo acciones exitosas ejecutadas por Krypton; no incluye tickets ni moderación manual no atribuible.", periodLabel, truncate(table.String(), 3800)),
		Color:       0x7B1FA2,
		Footer:      &discordgo.MessageEmbedFooter{Text: "El diario conserva acciones durante 365 días."},
	}, env.Theme))
	return nil
}

func hasPermission(interaction *discordgo.InteractionCreate, required int64) bool {
	if interaction == nil || interaction.Member == nil {
		return false
	}
	permissions := interaction.Member.Permissions
	return permissions&discordgo.PermissionAdministrator != 0 || permissions&required != 0
}

func eligibleStaffRoles(roles []*discordgo.Role, guildID string) []*discordgo.Role {
	eligible := make([]*discordgo.Role, 0, len(roles))
	for _, role := range roles {
		if role == nil || role.ID == guildID || role.Managed || role.Permissions&staffPermission == 0 {
			continue
		}
		eligible = append(eligible, role)
		if len(eligible) == 20 {
			break
		}
	}
	return eligible
}

func respond(ctx *bot.Context, content string, ephemeral bool) {
	data := &discordgo.InteractionResponseData{Content: content}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := ctx.Respond(data); err != nil {
		ctx.Env.Log.Error("no se pudo responder al comando", "command", ctx.Interaction.ApplicationCommandData().Name, "err", err)
	}
}

func respondError(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string) error {
	embed := &discordgo.MessageEmbed{Title: "❌ No se pudo completar la solicitud", Description: content, Color: 0x6A0712}
	ui.BrandEmbed(embed)
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func editResponse(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string) error {
	embed := &discordgo.MessageEmbed{Title: "❌ No se pudo completar la solicitud", Description: content, Color: 0x6A0712}
	ui.BrandEmbed(embed)
	empty := ""
	_, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Content: &empty,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return err
}

func logEditFailure(env *bot.Env, interaction *discordgo.InteractionCreate, err error) {
	if err == nil {
		return
	}
	logger := env.Log
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("no se pudo actualizar la respuesta del comando", "guild_id", interaction.GuildID, "err", err)
}

func isPrivateToEveryone(channel *discordgo.Channel, guildID string) bool {
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == discordgo.PermissionOverwriteTypeRole &&
			overwrite.ID == guildID &&
			overwrite.Deny&discordgo.PermissionViewChannel != 0 {
			return true
		}
	}
	return false
}

func editEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, theme *ui.Theme) error {
	theme.BrandEmbed(embed)
	_, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
	return err
}

func sanitizeText(value string) string {
	value = strings.ReplaceAll(value, "@", "@\u200b")
	return strings.TrimSpace(value)
}

func sanitizeAnnouncement(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "@everyone", "@\u200beveryone")
	value = strings.ReplaceAll(value, "@here", "@\u200bhere")
	value = strings.ReplaceAll(value, "<@", "<\u200b@")
	return value
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-3]) + "..."
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
