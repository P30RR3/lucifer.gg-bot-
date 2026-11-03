package audit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/pkg/discordutil"
)

const announcementDraftTTL = 30 * time.Minute

type announcementDraft struct {
	guildID   string
	channelID string
	ownerID   string
	text      string
	asEmbed   bool
	createdAt time.Time
}

var announcementDrafts sync.Map

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("announcement", handleAnnouncementComponent)
	router.RegisterModal("announcement_edit", handleAnnouncementEditModal)
}

func announcementPreview(ownerID, text string, asEmbed bool) (string, []*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	content := "### 📢・Borrador Oficial de Comunicado\n\n" + truncate(text, 1600) +
		"\n\n*⚜️ Verifique la redacción antes de su emisión formal a los canales.*"
	var embeds []*discordgo.MessageEmbed
	if asEmbed {
		content = ""
		embeds = []*discordgo.MessageEmbed{{
			Title:       "⚜️・COMUNICADO OFICIAL",
			Description: truncate(text, 4000),
			Color:       0x4A000A,
			Footer:      &discordgo.MessageEmbedFooter{Text: "Vista previa · Protocolo Lucifer"},
		}}
	}
	embedLabel := "Publicar con Embed"
	if asEmbed {
		embedLabel = "Publicar como Texto Normal"
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "announcement:delete:" + ownerID, Label: "Descartar", Emoji: &discordgo.ComponentEmoji{Name: "🗑️"}, Style: discordgo.DangerButton},
		discordgo.Button{CustomID: "announcement:edit:" + ownerID, Label: "Editar Texto", Emoji: &discordgo.ComponentEmoji{Name: "✍️"}, Style: discordgo.SecondaryButton},
		discordgo.Button{CustomID: "announcement:publish:" + ownerID, Label: "Emitir Comunicado", Emoji: &discordgo.ComponentEmoji{Name: "📢"}, Style: discordgo.SuccessButton},
		discordgo.Button{CustomID: "announcement:embed:" + ownerID, Label: embedLabel, Emoji: &discordgo.ComponentEmoji{Name: "📜"}, Style: discordgo.PrimaryButton},
	}}}
	return content, embeds, components
}

func handleAnnouncementComponent(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 3 || interaction.Message == nil || interaction.Member == nil || interaction.Member.User == nil {
		return respondError(session, interaction, "⚠️ El panel del comunicado ya no se encuentra vigente.")
	}
	ownerID := parts[2]
	draft, ok := loadAnnouncementDraft(interaction.Message.ID)
	if !ok || draft.guildID != interaction.GuildID || draft.ownerID != ownerID {
		return respondError(session, interaction, "⚠️ La sesión del borrador ha caducado. Vuelva a ejecutar `/anuncio`.")
	}
	if interaction.Member.User.ID != ownerID && !hasPermission(interaction, discordgo.PermissionManageMessages) {
		return respondError(session, interaction, "⚠️ Acción restringida a cancilleres autorizados o al autor del borrador.")
	}

	switch parts[1] {
	case "delete":
		if err := updateAnnouncementMessage(session, interaction, &discordgo.InteractionResponseData{
			Content:    "🗑️ **Borrador descartado.** No se emitió ningún comunicado oficial.",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		}); err != nil {
			return err
		}
		announcementDrafts.Delete(interaction.Message.ID)
		return nil
	case "edit":
		return openAnnouncementModal(session, interaction, "announcement_edit", interaction.Message.ID, ownerID, "✍️ Redactar Comunicado Oficial", "text", "Texto del anuncio", discordgo.TextInputParagraph, draft.text, 4000)
	case "embed":
		draft.asEmbed = !draft.asEmbed
		content, embeds, components := announcementPreview(ownerID, draft.text, draft.asEmbed)
		if err := updateAnnouncementMessage(session, interaction, &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     embeds,
			Components: components,
		}); err != nil {
			return err
		}
		announcementDrafts.Store(interaction.Message.ID, draft)
		return nil
	case "publish":
		var err error
		if draft.asEmbed {
			embed := &discordgo.MessageEmbed{
				Title:       "⚜️・COMUNICADO OFICIAL",
				Description: truncate(draft.text, 4000),
				Color:       env.Theme.Primary,
				Footer:      &discordgo.MessageEmbedFooter{Text: "Emitido bajo sello oficial de Lucifer ⚜️"},
			}
			env.Theme.BrandEmbed(embed)
			_, err = discordutil.ChannelMessageSendComplex(session, draft.channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
		} else {
			// Plain text announcement without embed as requested
			_, err = discordutil.ChannelMessageSendComplex(session, draft.channelID, &discordgo.MessageSend{
				Content: "📢 **COMUNICADO OFICIAL**\n\n" + truncate(draft.text, 1900) + "\n\n*⚜️ Emitido con distinción y vigencia protocolar.*",
			})
		}
		if err != nil {
			return fmt.Errorf("publicar anuncio en el canal: %w", err)
		}
		announcementDrafts.Delete(interaction.Message.ID)
		return updateAnnouncementMessage(session, interaction, &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✨ **Comunicado transmitido exitosamente** a <#%s>.", draft.channelID),
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		})
	default:
		return respondError(session, interaction, "Esa acción del anuncio ya no está disponible.")
	}
}

func openAnnouncementModal(session *discordgo.Session, interaction *discordgo.InteractionCreate, prefix, messageID, ownerID, title, inputID, label string, style discordgo.TextInputStyle, value string, maxLength int) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s:%s", prefix, messageID, ownerID),
			Title:    title,
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    inputID,
					Label:       label,
					Placeholder: "rojo oscuro, borgoña, granate, verde oscuro o negro",
					Style:       style,
					Value:       value,
					Required:    true,
					MinLength:   1,
					MaxLength:   maxLength,
				},
			}}},
		},
	})
}

func handleAnnouncementEditModal(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 {
		return respondError(session, interaction, "El formulario de edición ya no está disponible.")
	}
	draft, ok := loadAnnouncementDraft(parts[1])
	if !ok || draft.guildID != interaction.GuildID || draft.ownerID != parts[2] {
		return respondError(session, interaction, "La previsualización expiró. Vuelve a ejecutar `/anuncio`.")
	}
	if !canManageAnnouncement(interaction, draft.ownerID) {
		return respondError(session, interaction, "No tienes autorización para modificar este anuncio.")
	}
	text := strings.TrimSpace(modalText(interaction.ModalSubmitData().Components, "text"))
	if text == "" {
		return respondError(session, interaction, "El texto del anuncio no puede quedar vacío.")
	}
	draft.text = sanitizeAnnouncement(text)
	content, embeds, components := announcementPreview(draft.ownerID, draft.text, draft.asEmbed)
	if _, err := discordutil.ChannelMessageEditComplex(session, &discordgo.MessageEdit{
		ID:         parts[1],
		Channel:    draft.channelID,
		Content:    &content,
		Embeds:     &embeds,
		Components: &components,
	}); err != nil {
		env.Log.Error("no se pudo actualizar el borrador público del anuncio", "message_id", parts[1], "err", err)
		return respondError(session, interaction, "No se pudo actualizar la previsualización. Inténtalo otra vez.")
	}
	announcementDrafts.Store(parts[1], draft)
	return announcementAck(session, interaction, "✦ El borrador fue actualizado; el panel público ya muestra la nueva versión.")
}

func updateAnnouncementMessage(session *discordgo.Session, interaction *discordgo.InteractionCreate, data *discordgo.InteractionResponseData) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: data,
	})
}

func announcementAck(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func canManageAnnouncement(interaction *discordgo.InteractionCreate, ownerID string) bool {
	if interaction == nil || interaction.Member == nil || interaction.Member.User == nil {
		return false
	}
	return interaction.Member.User.ID == ownerID || hasPermission(interaction, discordgo.PermissionManageMessages)
}

func loadAnnouncementDraft(messageID string) (announcementDraft, bool) {
	value, ok := announcementDrafts.Load(messageID)
	if !ok {
		return announcementDraft{}, false
	}
	draft, ok := value.(announcementDraft)
	if !ok || time.Since(draft.createdAt) > announcementDraftTTL {
		announcementDrafts.Delete(messageID)
		return announcementDraft{}, false
	}
	return draft, true
}

func storeAnnouncementDraft(messageID string, draft announcementDraft) {
	now := time.Now()
	announcementDrafts.Range(func(key, value any) bool {
		current, ok := value.(announcementDraft)
		if !ok || now.Sub(current.createdAt) > announcementDraftTTL {
			announcementDrafts.Delete(key)
		}
		return true
	})
	announcementDrafts.Store(messageID, draft)
}

func announcementColor(name string) (int, bool) {
	colors := map[string]int{
		"rojo oscuro":  0x4A0000,
		"rojo":         0x4A0000,
		"borgoña":      0x3B0008,
		"burdeos":      0x3B0008,
		"granate":      0x510A14,
		"verde oscuro": 0x15301F,
		"verde":        0x15301F,
		"negro":        0x171014,
	}
	color, ok := colors[name]
	return color, ok
}

func modalText(components []discordgo.MessageComponent, id string) string {
	for _, component := range components {
		row, ok := component.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, child := range row.Components {
			if input, ok := child.(*discordgo.TextInput); ok && input.CustomID == id {
				return input.Value
			}
		}
	}
	return ""
}
