package moderation

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/ui"
)

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("sanction_appeal", handleAppealButton)
	router.RegisterModal("sanction_appeal_submit", handleAppealSubmit)
}

func sendSanctionAppealDM(session *discordgo.Session, env *bot.Env, guildID string, user *discordgo.User, sanction, reason string) error {
	if user == nil {
		return fmt.Errorf("no se pudo identificar al usuario sancionado")
	}
	channel, err := session.UserChannelCreate(user.ID)
	if err != nil {
		return fmt.Errorf("abrir DM de sanción: %w", err)
	}
	return sendSanctionAppealDMToChannel(session, env, guildID, channel.ID, user, sanction, reason)
}

func sendSanctionAppealDMToChannel(session *discordgo.Session, env *bot.Env, guildID, channelID string, user *discordgo.User, sanction, reason string) error {
	if user == nil || channelID == "" {
		return fmt.Errorf("no se pudo identificar el DM del usuario sancionado")
	}
	guildName := "este servidor"
	if guild, err := session.Guild(guildID); err == nil && guild != nil && guild.Name != "" {
		guildName = guild.Name
	}
	embed := &discordgo.MessageEmbed{
		Title:       "⚖️・AVISO DEL CLUB",
		Description: fmt.Sprintf("La administración de **%s** registró una sanción para tu cuenta. Si crees que hubo un error, puedes enviar una apelación al equipo del servidor.", guildName),
		Color:       env.Theme.ColorWarning,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Medida", Value: sanctionLabel(sanction), Inline: true},
			{Name: "Motivo", Value: ui.FrameMessage(reason), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "La apelación será revisada por moderadores; no revela información privada."},
	}
	env.Theme.BrandEmbed(embed)
	_, err := discordutil.ChannelMessageSendComplex(session, channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Apelar sanción",
					Emoji:    &discordgo.ComponentEmoji{Name: "⚖️"},
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("sanction_appeal:%s:%s:%s", guildID, user.ID, sanction),
				},
			}},
		},
	})
	return err
}

func handleAppealButton(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, _ *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 4 || interaction.User == nil || interaction.User.ID != parts[2] {
		return respondAppeal(session, interaction, "⚖️ Este botón de apelación no corresponde a tu cuenta.", true)
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("sanction_appeal_submit:%s:%s:%s", parts[1], parts[2], parts[3]),
			Title:    "⚖️ Apelar una sanción",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "reason",
						Label:       "¿Por qué consideras injusta la sanción?",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						MinLength:   10,
						MaxLength:   1000,
						Placeholder: "Explica los hechos con respeto y detalle...",
					},
				}},
			},
		},
	})
}

func handleAppealSubmit(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.User == nil {
		return respondAppeal(session, interaction, "⚖️ No pude verificar la cuenta que envía la apelación.", true)
	}
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 4 || parts[0] != "sanction_appeal_submit" || parts[2] != interaction.User.ID {
		return respondAppeal(session, interaction, "⚖️ Este formulario ya no es válido para tu cuenta.", true)
	}
	guildID, targetID, sanction := parts[1], parts[2], parts[3]
	if guildID == "" || targetID == "" || sanction == "" {
		return respondAppeal(session, interaction, "⚖️ Faltan datos de la sanción; no se guardó la apelación.", true)
	}
	reason := strings.TrimSpace(appealModalText(interaction.ModalSubmitData().Components, "reason"))
	if len([]rune(reason)) < 10 {
		return respondAppeal(session, interaction, "⚖️ Añade un poco más de contexto para que el equipo pueda revisar tu caso.", true)
	}
	appealID, err := env.Store.CreateSanctionAppeal(ctx, guildID, targetID, sanction, reason)
	if err != nil {
		return fmt.Errorf("guardar apelación de sanción: %w", err)
	}
	settings, err := env.Store.GetAuditSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("consultar canal del staff para apelación: %w", err)
	}
	if settings.LogChannelID != "" {
		embed := &discordgo.MessageEmbed{
			Title:       "⚖️・NUEVA APELACIÓN",
			Description: fmt.Sprintf("**Expediente:** `#%d`\n**Usuario:** <@%s>\n**Medida:** %s\n\n**Declaración:**\n%s", appealID, targetID, sanctionLabel(sanction), reason),
			Color:       env.Theme.ColorWarning,
			Footer:      &discordgo.MessageEmbedFooter{Text: "Revisa este expediente desde /puertas-infernales → Apelaciones."},
		}
		env.Theme.BrandEmbed(embed)
		if _, err := discordutil.ChannelMessageSendEmbed(session, settings.LogChannelID, embed); err != nil {
			logger := env.Log
			if logger == nil {
				logger = slog.Default()
			}
			logger.Error("apelación guardada, pero no se pudo avisar al canal de auditoría", "guild_id", guildID, "appeal_id", appealID, "err", err)
			return fmt.Errorf("apelación guardada, pero falló el aviso al staff: %w", err)
		}
	}
	return respondAppeal(session, interaction, fmt.Sprintf("⚖️ Tu apelación **#%s** quedó registrada. El equipo del servidor la revisará desde su consola privada.", strconv.FormatInt(appealID, 10)), true)
}

func appealModalText(components []discordgo.MessageComponent, customID string) string {
	for _, row := range components {
		actionsRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, component := range actionsRow.Components {
			input, ok := component.(*discordgo.TextInput)
			if ok && input.CustomID == customID {
				return input.Value
			}
		}
	}
	return ""
}

func respondAppeal(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: ui.FrameMessage(content), Flags: flags},
	})
}

func logDeliveryFailure(env *bot.Env, sanction, guildID, userID string, err error) {
	logger := env.Log
	if logger == nil {
		logger = slog.Default()
	}
	logger.Warn("sanción aplicada; no se pudo enviar la notificación de apelación por DM", "sanction", sanction, "guild_id", guildID, "target_id", userID, "err", err)
}

func sanctionLabel(sanction string) string {
	switch sanction {
	case "ban":
		return "Baneo"
	case "timeout":
		return "Silencio temporal"
	case "warn":
		return "Advertencia"
	case "kick":
		return "Expulsión"
	default:
		return sanction
	}
}
