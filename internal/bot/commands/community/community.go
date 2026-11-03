package community

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

const (
	maxIdeaTitle       = 80
	maxIdeaDescription = 1200
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	return registry.Register(&bot.Command{
		Name:        "ideas",
		Description: "Abre el panel de ideas o envía una propuesta nueva.",
		Category:    "comunidad",
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "panel",
				Description: "Explora, consulta, vota y resuelve propuestas con botones.",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "enviar",
				Description: "Publica una propuesta para mejorar este servidor.",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "titulo", Description: "Título breve de la propuesta", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "detalle", Description: "Explica la propuesta (10-1200 caracteres)", Required: true},
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: "anonima", Description: "Oculta tu nombre en la publicación", Required: false},
				},
			},
		},
		Handler: c.handleIdeas,
	})
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("ideas", handleIdeasComponent)
}

func (c *Cog) handleIdeas(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 {
		return respond(session, interaction, "Elige una acción para las ideas del club.", true)
	}

	subcommand := options[0]
	switch subcommand.Name {
	case "panel":
		return showIdeaPanel(ctx, session, interaction, env, 0)
	case "enviar":
		return c.submitIdea(ctx, session, interaction, env, subcommand.Options)
	default:
		return respond(session, interaction, "Esa acción no existe en el tablero del Lux.", true)
	}
}

func showIdeaPanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, selectedID int64) error {
	ideas, err := env.Store.ListCommunityIdeas(ctx, interaction.GuildID, 20)
	if err != nil {
		return fmt.Errorf("cargar panel de ideas: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "💡・EL TABLERO DE IDEAS DEL LUX",
		Description: "Elige una propuesta en el menú para leerla y apoyarla. El staff verá controles para aceptarla o descartarla.",
		Color:       env.Theme.Primary,
	}
	options := make([]discordgo.SelectMenuOption, 0, len(ideas))
	selectedOpen := false
	for _, idea := range ideas {
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncateOption(fmt.Sprintf("#%d・%s", idea.ID, idea.Title), 100),
			Value:       strconv.FormatInt(idea.ID, 10),
			Description: truncateOption(fmt.Sprintf("%d votos・%s", idea.Votes, ideaStatusLabel(idea.Status)), 100),
			Default:     idea.ID == selectedID,
		})
	}
	if len(ideas) == 0 {
		embed.Description = "Todavia no hay propuestas. Usa `/ideas enviar` para abrir la primera."
	}
	if selectedID > 0 {
		selected, err := env.Store.GetCommunityIdea(ctx, interaction.GuildID, selectedID)
		switch {
		case errors.Is(err, storage.ErrCommunityIdeaNotFound):
			selectedID = 0
		case err != nil:
			return fmt.Errorf("cargar propuesta seleccionada: %w", err)
		default:
			selectedOpen = selected.Status == "open"
			author := fmt.Sprintf("<@%s>", selected.AuthorID)
			if selected.Anonymous {
				author = "Identidad reservada"
			}
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: fmt.Sprintf("📌 #%d・%s", selected.ID, safeText(selected.Title)), Value: safeText(selected.Description)},
				{Name: "🗳️ Apoyos", Value: fmt.Sprintf("%d", selected.Votes), Inline: true},
				{Name: "📍 Estado", Value: ideaStatusLabel(selected.Status), Inline: true},
				{Name: "👤 Autoría", Value: author, Inline: true},
			}
		}
	}
	components := []discordgo.MessageComponent{}
	if len(options) > 0 {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.StringSelectMenu,
				CustomID:    fmt.Sprintf("ideas:%s:select", interaction.Member.User.ID),
				Placeholder: "✦ Elige una propuesta ✦",
				Options:     options,
				MinValues:   intPointer(1),
				MaxValues:   1,
			},
		}})
		if selectedID > 0 && selectedOpen {
			components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{CustomID: fmt.Sprintf("ideas:%s:vote:%d", interaction.Member.User.ID, selectedID), Label: "🗳️ Apoyar / retirar voto", Style: discordgo.PrimaryButton},
			}})
			if memberHasPermission(interaction.Member, discordgo.PermissionManageMessages) {
				components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{CustomID: fmt.Sprintf("ideas:%s:accepted:%d", interaction.Member.User.ID, selectedID), Label: "✅ Aceptar", Style: discordgo.SuccessButton},
					discordgo.Button{CustomID: fmt.Sprintf("ideas:%s:declined:%d", interaction.Member.User.ID, selectedID), Label: "🗑️ Descartar", Style: discordgo.DangerButton},
				}})
			}
		}
	}
	env.Theme.BrandEmbed(embed)
	if interaction.Type == discordgo.InteractionMessageComponent {
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
		})
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func handleIdeasComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "Este panel solo funciona dentro del servidor.", true)
	}
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 3 {
		return respond(session, interaction, "Este control ya no está disponible. Abre `/ideas panel` para actualizarlo.", true)
	}
	action := parts[2]
	selectedID := int64(0)
	if action == "select" {
		values := interaction.MessageComponentData().Values
		if len(values) != 1 {
			return respond(session, interaction, "Elige una sola propuesta.", true)
		}
		id, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || id < 1 {
			return respond(session, interaction, "La propuesta seleccionada no es valida.", true)
		}
		return showIdeaPanel(ctx, session, interaction, env, id)
	}
	if len(parts) != 4 {
		return respond(session, interaction, "Este control ya no es valido.", true)
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || id < 1 {
		return respond(session, interaction, "La propuesta seleccionada no es valida.", true)
	}
	selectedID = id
	switch action {
	case "vote":
		if _, _, err := env.Store.ToggleCommunityIdeaVote(ctx, interaction.GuildID, id, interaction.Member.User.ID); err != nil {
			if errors.Is(err, storage.ErrCommunityIdeaNotFound) || errors.Is(err, storage.ErrCommunityIdeaClosed) {
				return respond(session, interaction, "La propuesta ya no esta disponible para votar.", true)
			}
			return fmt.Errorf("actualizar voto desde panel: %w", err)
		}
	case "accepted", "declined":
		if !memberHasPermission(interaction.Member, discordgo.PermissionManageMessages) {
			return respond(session, interaction, "Esta decision requiere Gestionar mensajes.", true)
		}
		status := action
		if err := env.Store.SetCommunityIdeaStatus(ctx, interaction.GuildID, id, status); err != nil {
			if errors.Is(err, storage.ErrCommunityIdeaNotFound) || errors.Is(err, storage.ErrCommunityIdeaClosed) {
				return respond(session, interaction, "La propuesta ya no se puede resolver.", true)
			}
			return fmt.Errorf("resolver propuesta desde panel: %w", err)
		}
	default:
		return respond(session, interaction, "Accion de ideas desconocida.", true)
	}
	return showIdeaPanel(ctx, session, interaction, env, selectedID)
}

func (c *Cog) submitIdea(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	title := strings.TrimSpace(option(options, "titulo").StringValue())
	description := strings.TrimSpace(option(options, "detalle").StringValue())
	if len([]rune(title)) < 5 || len([]rune(title)) > maxIdeaTitle {
		return respond(session, interaction, "El título debe tener entre 5 y 80 caracteres.", true)
	}
	if len([]rune(description)) < 10 || len([]rune(description)) > maxIdeaDescription {
		return respond(session, interaction, "El detalle debe tener entre 10 y 1.200 caracteres.", true)
	}

	anonymous := false
	if anonymousOption := option(options, "anonima"); anonymousOption != nil {
		anonymous = anonymousOption.BoolValue()
	}
	idea, err := env.Store.CreateCommunityIdea(
		ctx, interaction.GuildID, interaction.Member.User.ID,
		title, description, anonymous,
	)
	if errors.Is(err, storage.ErrCommunityIdeaCooldown) {
		return respond(session, interaction, "⏳ Ya publicaste una propuesta hace poco. Espera 10 minutos antes de enviar otra.", true)
	}
	if err != nil {
		return fmt.Errorf("guardar propuesta: %w", err)
	}

	author := interaction.Member.User.Username
	if anonymous {
		author = "Miembro del club (identidad reservada)"
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("💡 Propuesta #%d · %s", idea.ID, safeText(idea.Title)),
		Description: safeText(idea.Description),
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Propuesta por", Value: safeText(author), Inline: true},
			{Name: "Estado", Value: "🟡 Abierta a votos", Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Abre /ideas panel para explorar y votar propuestas.",
		},
	}
	env.Theme.BrandEmbed(embed)
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🍸 La propuesta ya está sobre la mesa del Lux.",
			Embeds:  []*discordgo.MessageEmbed{embed},
		},
	}); err != nil {
		return fmt.Errorf("publicar propuesta: %w", err)
	}
	return nil
}

func (c *Cog) voteIdea(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de propuesta válido.", true)
	}
	voted, votes, err := env.Store.ToggleCommunityIdeaVote(
		ctx, interaction.GuildID, id, interaction.Member.User.ID,
	)
	if errors.Is(err, storage.ErrCommunityIdeaNotFound) {
		return respond(session, interaction, "No encontré esa propuesta en este servidor.", true)
	}
	if errors.Is(err, storage.ErrCommunityIdeaClosed) {
		return respond(session, interaction, "Esa propuesta ya fue resuelta y no admite más votos.", true)
	}
	if err != nil {
		return fmt.Errorf("votar propuesta: %w", err)
	}

	action := "retiraste tu apoyo"
	if voted {
		action = "apoyaste la propuesta"
	}
	return respond(session, interaction, fmt.Sprintf("🗳️ %s. Ahora tiene **%d votos**.", action, votes), true)
}

func (c *Cog) listIdeas(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	ideas, err := env.Store.ListCommunityIdeas(ctx, interaction.GuildID, 10)
	if err != nil {
		return fmt.Errorf("listar propuestas: %w", err)
	}
	if len(ideas) == 0 {
		return respond(session, interaction, "Todavía no hay propuestas. Sé la primera persona en usar `/ideas enviar`.", true)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💡 El tablero de ideas del Lux",
		Description: "Las propuestas abiertas aparecen primero; los votos ayudan al equipo a priorizar.",
		Color:       env.Theme.Primary,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Abre /ideas panel · Cada persona tiene un voto por propuesta"},
	}
	for _, idea := range ideas {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("#%d · %s", idea.ID, safeText(idea.Title)),
			Value:  fmt.Sprintf("**%s** · %d votos\n%s", ideaStatusLabel(idea.Status), idea.Votes, shorten(safeText(idea.Description), 180)),
			Inline: false,
		})
	}
	env.Theme.BrandEmbed(embed)
	return respondEmbed(session, interaction, embed, true)
}

func (c *Cog) showIdea(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de propuesta válido.", true)
	}
	idea, err := env.Store.GetCommunityIdea(ctx, interaction.GuildID, id)
	if errors.Is(err, storage.ErrCommunityIdeaNotFound) {
		return respond(session, interaction, "No encontré esa propuesta en este servidor.", true)
	}
	if err != nil {
		return fmt.Errorf("consultar propuesta: %w", err)
	}

	author := fmt.Sprintf("<@%s>", idea.AuthorID)
	if idea.Anonymous {
		author = "Identidad reservada"
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("💡 Propuesta #%d · %s", idea.ID, safeText(idea.Title)),
		Description: safeText(idea.Description),
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Estado", Value: ideaStatusLabel(idea.Status), Inline: true},
			{Name: "Apoyos", Value: fmt.Sprintf("%d", idea.Votes), Inline: true},
			{Name: "Autoría", Value: author, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Usa /ideas panel para apoyar una propuesta abierta."},
	}
	env.Theme.BrandEmbed(embed)
	return respondEmbed(session, interaction, embed, true)
}

func (c *Cog) resolveIdea(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	if interaction.Member == nil || interaction.Member.Permissions&(discordgo.PermissionManageMessages|discordgo.PermissionManageGuild|discordgo.PermissionAdministrator) == 0 {
		return respond(session, interaction, "Solo el equipo de moderación puede resolver propuestas.", true)
	}
	id := option(options, "id").IntValue()
	status := option(options, "decision").StringValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de propuesta válido.", true)
	}
	err := env.Store.SetCommunityIdeaStatus(ctx, interaction.GuildID, id, status)
	if errors.Is(err, storage.ErrCommunityIdeaNotFound) {
		return respond(session, interaction, "No encontré esa propuesta en este servidor.", true)
	}
	if errors.Is(err, storage.ErrCommunityIdeaClosed) {
		return respond(session, interaction, "Esa propuesta ya fue resuelta.", true)
	}
	if err != nil {
		return fmt.Errorf("resolver propuesta: %w", err)
	}
	return respond(session, interaction, fmt.Sprintf("✅ Propuesta **#%d** marcada como **%s**.", id, ideaStatusLabel(status)), true)
}

func option(options []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, candidate := range options {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

func respond(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: ui.FrameMessage(content), Flags: flags},
	})
}

func respondEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  flags,
		},
	})
}

func ideaStatusLabel(status string) string {
	switch status {
	case "accepted":
		return "✅ Aceptada"
	case "declined":
		return "⚪ Descartada"
	default:
		return "🟡 Abierta"
	}
}

func safeText(value string) string {
	value = strings.ReplaceAll(value, "@", "@\u200b")
	return strings.TrimSpace(value)
}

func shorten(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func truncateOption(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func intPointer(value int) *int {
	return &value
}

func memberHasPermission(member *discordgo.Member, permission int64) bool {
	return member != nil && member.Permissions&(permission|discordgo.PermissionAdministrator) != 0
}
