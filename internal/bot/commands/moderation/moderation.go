package moderation

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(r *bot.Registry) error {
	return r.RegisterAll(
		// Ban
		&bot.Command{
			Name:        "banear",
			Description: "Banea a un miembro del servidor de forma permanente.",
			Category:    "moderacion",
			DefaultPerm: discordgo.PermissionBanMembers,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El miembro que será baneado",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "razon",
					Description: "Razón del castigo",
					Required:    false,
				},
			},
			Handler: c.handleBan,
		},

		// Kick
		&bot.Command{
			Name:        "expulsar",
			Description: "Lucifer expulsa a un usuario del club (Kick) 🔨",
			Category:    "moderacion",
			DefaultPerm: discordgo.PermissionKickMembers,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El usuario que será expulsado",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "razon",
					Description: "Razón de la expulsión",
					Required:    false,
				},
			},
			Handler: c.handleKick,
		},

		// Timeout
		&bot.Command{
			Name:        "silenciar",
			Description: "Lucifer aplica un silencio temporal autorizado (Timeout) 🤐",
			Category:    "moderacion",
			DefaultPerm: discordgo.PermissionModerateMembers,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El usuario a silenciar",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "minutos",
					Description: "Tiempo en minutos",
					Required:    true,
				},
			},
			Handler: c.handleTimeout,
		},

		&bot.Command{
			Name:        "advertir",
			Description: "Lucifer registra una advertencia formal para el usuario ⚠️",
			Category:    "moderacion",
			DefaultPerm: discordgo.PermissionManageMessages,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El usuario que recibirá la advertencia",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "razon",
					Description: "Motivo de la advertencia",
					Required:    true,
					MaxLength:   500,
				},
			},
			Handler: c.handleWarn,
		},

		// Purge
		&bot.Command{
			Name:        "limpiar",
			Description: "Lucifer limpia mensajes del chat (Borrar mensajes) 🧹",
			Category:    "moderacion",
			DefaultPerm: discordgo.PermissionManageMessages,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "cantidad",
					Description: "Número de mensajes a borrar (1-100)",
					Required:    true,
				},
			},
			Handler: c.handlePurge,
		},
	)
}

// ==========================================
// HANDLERS DE MODERACIÓN
// ==========================================

func (c *Cog) handleBan(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	user := i.ApplicationCommandData().Options[0].UserValue(s)
	reason := "Por orden del staff del Lux"
	if len(i.ApplicationCommandData().Options) > 1 {
		reason = i.ApplicationCommandData().Options[1].StringValue()
	}

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer está revisando la solicitud", "Aplicando la acción de moderación autorizada…")},
		},
	})

	time.Sleep(1 * time.Second)
	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer prepara la sanción", "Revisando los permisos y la jerarquía del servidor…")},
	})
	time.Sleep(1 * time.Second)

	dmChannel, dmErr := s.UserChannelCreate(user.ID)
	err := s.GuildBanCreateWithReason(i.GuildID, user.ID, reason, 0)
	if err != nil {
		_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(""),
			Embeds:  &[]*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Baneo no aplicado", "Discord no permitió completar el baneo. Revisa la jerarquía y los permisos del bot.")},
		})
		return nil
	}
	if err := recordStaffAction(ctx, env, s, i, user.ID, "ban", reason); err != nil {
		return fmt.Errorf("registrar baneo en auditoría: %w", err)
	}
	if dmErr == nil {
		dmErr = sendSanctionAppealDMToChannel(s, env, i.GuildID, dmChannel.ID, user, "ban", reason)
	}
	if dmErr != nil {
		logDeliveryFailure(env, "ban", i.GuildID, user.ID, dmErr)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⛔・MIEMBRO BANEADO",
		Description: fmt.Sprintf("**%s** ha sido enviado al infierno.\n\n*«No importa cuánto intentes huir de tus pecados, al final, siempre te alcanzan.»*", user.Mention()),
		Color:       env.Theme.ColorDanger,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Razón del castigo", Value: reason, Inline: false},
			{Name: "Ejecutor", Value: i.Member.User.Mention(), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    env.Theme.FooterText,
			IconURL: env.Theme.FooterIcon,
		},
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(""),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func (c *Cog) handleKick(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	user := i.ApplicationCommandData().Options[0].UserValue(s)
	reason := "Comportamiento inaceptable en el club"
	if len(i.ApplicationCommandData().Options) > 1 {
		reason = i.ApplicationCommandData().Options[1].StringValue()
	}

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer prepara la expulsión", "Revisando los permisos y la jerarquía del servidor…")},
		},
	})
	time.Sleep(1 * time.Second)

	dmChannel, dmErr := s.UserChannelCreate(user.ID)
	err := s.GuildMemberDeleteWithReason(i.GuildID, user.ID, reason)
	if err != nil {
		_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(""),
			Embeds:  &[]*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Expulsión no aplicada", "No tengo permisos suficientes para expulsar a esta persona.")},
		})
		return nil
	}
	if err := recordStaffAction(ctx, env, s, i, user.ID, "kick", reason); err != nil {
		return fmt.Errorf("registrar expulsión en auditoría: %w", err)
	}
	if dmErr == nil {
		dmErr = sendSanctionAppealDMToChannel(s, env, i.GuildID, dmChannel.ID, user, "kick", reason)
	}
	if dmErr != nil {
		logDeliveryFailure(env, "kick", i.GuildID, user.ID, dmErr)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔪 Alguien fue sacado a patadas",
		Description: fmt.Sprintf("**%s** fue expulsado del servidor.\n\n*«Vuelve cuando aprendas a comportarte en mi club.»*", user.Mention()),
		Color:       env.Theme.ColorWarning,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Razón", Value: reason, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: env.Theme.FooterText},
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(""),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func (c *Cog) handleTimeout(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	user := i.ApplicationCommandData().Options[0].UserValue(s)
	minutes := i.ApplicationCommandData().Options[1].IntValue()

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer prepara el silencio temporal", "Revisando los permisos y la jerarquía del servidor…")},
		},
	})
	time.Sleep(1 * time.Second)

	timeoutUntil := time.Now().Add(time.Duration(minutes) * time.Minute)
	err := s.GuildMemberTimeout(i.GuildID, user.ID, &timeoutUntil)
	if err != nil {
		_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(""),
			Embeds:  &[]*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Silencio no aplicado", "Discord no permitió silenciar a este usuario. Comprueba la jerarquía de roles del bot.")},
		})
		return nil
	}
	if err := recordStaffAction(ctx, env, s, i, user.ID, "timeout", fmt.Sprintf("%d minutos", minutes)); err != nil {
		return fmt.Errorf("registrar silencio en auditoría: %w", err)
	}
	if err := sendSanctionAppealDM(s, env, i.GuildID, user, "timeout", fmt.Sprintf("Silencio temporal de %d minutos.", minutes)); err != nil {
		logDeliveryFailure(env, "timeout", i.GuildID, user.ID, err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🤐 Silencio en la sala",
		Description: fmt.Sprintf("**%s** ha perdido el derecho a hablar por **%d minutos**.\n\n*«Shh... los adultos están hablando.»*", user.Mention(), minutes),
		Color:       env.Theme.ColorNeutral,
		Footer:      &discordgo.MessageEmbedFooter{Text: env.Theme.FooterText},
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(""),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func (c *Cog) handlePurge(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	amount := i.ApplicationCommandData().Options[0].IntValue()
	if amount < 1 || amount > 100 {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Límite de limpieza", "Solo se pueden limpiar entre 1 y 100 mensajes a la vez.")},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return nil
	}

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer limpia el canal", fmt.Sprintf("Preparando la eliminación de %d mensajes…", amount))},
		},
	})

	messages, err := s.ChannelMessages(i.ChannelID, int(amount), "", "", "")
	if err != nil {
		_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(""),
			Embeds:  &[]*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Limpieza no completada", "No se pudieron leer los mensajes del canal.")},
		})
		return nil
	}

	var messageIDs []string
	for _, m := range messages {
		messageIDs = append(messageIDs, m.ID)
	}

	err = s.ChannelMessagesBulkDelete(i.ChannelID, messageIDs)
	if err != nil {
		_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(""),
			Embeds:  &[]*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Limpieza no completada", "No se pudieron borrar los mensajes. Discord no permite eliminar mensajes con más de 14 días mediante esta operación.")},
		})
		return nil
	}
	_ = recordStaffAction(ctx, env, s, i, "", "purge", fmt.Sprintf("%d mensajes en el canal %s", len(messageIDs), i.ChannelID))

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(""),
	})
	return nil
}

func (c *Cog) handleWarn(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	options := i.ApplicationCommandData().Options
	if len(options) < 2 {
		return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Indica el usuario y el motivo de la advertencia.", Flags: discordgo.MessageFlagsEphemeral},
		})
	}
	target := options[0].UserValue(s)
	reason := strings.TrimSpace(options[1].StringValue())
	if target == nil || reason == "" {
		return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "El usuario y el motivo son obligatorios.", Flags: discordgo.MessageFlagsEphemeral},
		})
	}
	if i.Member != nil && i.Member.User != nil && target.ID == i.Member.User.ID {
		return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "No puedes advertirte a ti mismo.", Flags: discordgo.MessageFlagsEphemeral},
		})
	}
	if err := discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return err
	}
	if err := recordStaffAction(ctx, env, s, i, target.ID, "warn", reason); err != nil {
		logWarningEditFailure(env, i, editWarningResponse(s, i, "No pude registrar la advertencia. No se envió al usuario."))
		return nil
	}

	err := sendSanctionAppealDM(s, env, i.GuildID, target, "warn", reason)
	if err != nil {
		logDeliveryFailure(env, "warn", i.GuildID, target.ID, err)
	}
	delivery := "No pude enviar el aviso por DM (posiblemente están cerrados)."
	if err == nil {
		delivery = "Se notificó al usuario por DM y puede apelar desde ese mensaje."
	}
	logWarningEditFailure(env, i, editWarningResponse(s, i, fmt.Sprintf("⚠️ Advertencia registrada para **%s**. %s", target.Username, delivery)))
	return nil
}

func recordStaffAction(ctx context.Context, env *bot.Env, s *discordgo.Session, interaction *discordgo.InteractionCreate, targetID, action, details string) error {
	if env == nil || env.Store == nil || interaction == nil || interaction.Member == nil || interaction.Member.User == nil {
		return fmt.Errorf("no se pudo identificar al miembro del staff")
	}
	var err error
	if env.Auditor != nil {
		err = env.Auditor.RecordStaffAction(ctx, s, interaction.GuildID, interaction.Member.User.ID, targetID, action, details)
	} else {
		err = env.Store.RecordStaffAction(ctx, interaction.GuildID, interaction.Member.User.ID, targetID, action, details)
	}
	if err != nil {
		logger := env.Log
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("falló el registro de una acción de moderación", "guild_id", interaction.GuildID, "actor_id", interaction.Member.User.ID, "action", action, "err", err)
		return err
	}
	return nil
}

func editWarningResponse(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string) error {
	_, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{Content: &content})
	return err
}

func logWarningEditFailure(env *bot.Env, interaction *discordgo.InteractionCreate, err error) {
	if err == nil {
		return
	}
	logger := env.Log
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("no se pudo actualizar la respuesta de advertencia", "guild_id", interaction.GuildID, "err", err)
}

func stringPtr(s string) *string {
	return &s
}
