package luxpanel

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
)

const (
	pageProfile   = "profile"
	pageInventory = "inventory"
	pageSocial    = "social"
	pageSettings  = "settings"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	return registry.Register(&bot.Command{
		Name:        "lux",
		Description: "Abre tu panel personal de perfil, inventario, comunidad y ajustes.",
		Category:    "comunidad",
		GuildOnly:   true,
		Handler:     c.handlePanel,
	})
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("lux", handleComponent)
}

func (c *Cog) handlePanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return fmt.Errorf("iniciar panel /lux: %w", err)
	}
	panel, err := env.Store.LoadLuxPanel(ctx, interaction.GuildID, interaction.Member.User.ID)
	if err != nil {
		return fmt.Errorf("cargar panel /lux: %w", err)
	}

	embed, components := renderPanel(interaction.Member.User.ID, pageProfile, panel, env.Theme.Primary)
	env.Theme.BrandEmbed(embed)
	if _, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		return fmt.Errorf("mostrar panel /lux: %w", err)
	}
	return nil
}

func handleComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 3 || interaction.Member == nil || interaction.Member.User == nil {
		return respondPanelError(session, interaction, "No pude verificar este panel. Vuelve a abrir `/lux`.")
	}
	if interaction.Member.User.ID != parts[1] {
		return respondPanelError(session, interaction, "Este panel es privado; abre `/lux` para consultar el tuyo.")
	}
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		return fmt.Errorf("iniciar actualización del panel /lux: %w", err)
	}

	page := pageProfile
	action := "select"
	if len(parts) > 2 {
		page = parts[2]
	}
	if len(parts) > 3 {
		action = parts[3]
	}

	status := ""
	switch action {
	case "select":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 || !validPage(values[0]) {
			return followupPanelError(session, interaction, "Esa sección del panel no existe.")
		}
		page = values[0]
	case "daily":
		balance, err := env.Store.ClaimDailyChips(ctx, interaction.GuildID, interaction.Member.User.ID)
		switch {
		case err == nil:
			status = fmt.Sprintf("🪙 Reclamaste **250 fichas**. Saldo actual: **%d**.", balance)
		case err == storage.ErrDailyCooldown:
			status = "⏳ Ya reclamaste la recompensa de hoy. Vuelve dentro de 24 horas."
		default:
			return fmt.Errorf("reclamar recompensa desde /lux: %w", err)
		}
		page = pageProfile
	case "notifications", "hide_inventory", "looking_for_group", "language", "responses_public":
		if err := env.Store.ToggleLuxPreference(ctx, interaction.GuildID, interaction.Member.User.ID, action); err != nil {
			return fmt.Errorf("actualizar ajuste de /lux: %w", err)
		}
		if action == "responses_public" {
			visible, err := env.Store.LuxResponsesPublic(ctx, interaction.GuildID, interaction.Member.User.ID)
			if err != nil {
				return fmt.Errorf("consultar visibilidad actualizada: %w", err)
			}
			discordutil.SetResponseVisibility(interaction.GuildID, interaction.Member.User.ID, visible)
		}
		if action == "looking_for_group" {
			page = pageSocial
		} else {
			page = pageSettings
		}
	case "":
		return followupPanelError(session, interaction, "La acción del panel no es válida.")
	default:
		return followupPanelError(session, interaction, "La acción del panel ya no está disponible.")
	}

	panel, err := env.Store.LoadLuxPanel(ctx, interaction.GuildID, interaction.Member.User.ID)
	if err != nil {
		return fmt.Errorf("actualizar panel /lux: %w", err)
	}
	embed, components := renderPanel(interaction.Member.User.ID, page, panel, env.Theme.Primary)
	env.Theme.BrandEmbed(embed)
	if action == "responses_public" {
		if panel.Preferences.ResponsesPublic {
			_, err := discordutil.ChannelMessageSendComplex(session, interaction.ChannelID, &discordgo.MessageSend{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			})
			if err != nil {
				return fmt.Errorf("publicar panel con respuestas visibles: %w", err)
			}
			empty := ""
			if _, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
				Content:    &empty,
				Embeds:     &[]*discordgo.MessageEmbed{},
				Components: &[]discordgo.MessageComponent{},
			}); err != nil {
				return fmt.Errorf("retirar panel privado tras activar visibilidad: %w", err)
			}
			return nil
		}
		if interaction.Message != nil && interaction.Message.Flags&discordgo.MessageFlagsEphemeral == 0 {
			if err := session.ChannelMessageDelete(interaction.ChannelID, interaction.Message.ID); err != nil {
				return fmt.Errorf("retirar panel público tras desactivar visibilidad: %w", err)
			}
			_, err := discordutil.FollowupMessageCreate(session, interaction.Interaction, false, &discordgo.WebhookParams{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				return fmt.Errorf("mostrar panel privado de ajustes: %w", err)
			}
			return nil
		}
	}
	edit := &discordgo.WebhookEdit{
		Content:    &status,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}
	if _, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, edit); err != nil {
		return fmt.Errorf("actualizar mensaje del panel /lux: %w", err)
	}
	return nil
}

func renderPanel(userID, page string, panel *storage.LuxPanelData, color int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	english := panel.Preferences.Language == "en"
	title := "⚜️・PROTOCOLO SOBERANO DE MEMBRESÍA"
	description := "Su consola personal de honor. Seleccione una sección protocolar para consultar su rango, insignias y ajustes."
	if english {
		title = "⚜️・SOVEREIGN MEMBERSHIP PROTOCOL"
		description = "Your personal honor console. Select a section below to review your rank, badges and settings."
	}

	embed := &discordgo.MessageEmbed{Title: title, Description: description, Color: color}
	level := panel.Experience/100 + 1
	progress := panel.Experience % 100
	switch page {
	case pageProfile:
		rank := rankForLevel(level, english)
		if panel.Rank != "" {
			rank = panel.Rank
		}
		if english {
			embed.Title = "👑・Your Sovereign Profile"
			embed.Description = "Official registry of distinction, treasury balance and daily stipend."
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Rank", Value: rank, Inline: true},
				{Name: "Tier", Value: fmt.Sprintf("Nivel %d", level), Inline: true},
				{Name: "Sovereign Coins", Value: fmt.Sprintf("🪙 %d", panel.Balance), Inline: true},
				{Name: "Protocol XP", Value: fmt.Sprintf("%d / 100 XP", progress), Inline: true},
			}
		} else {
			embed.Title = "👑・Su Expediente Soberano"
			embed.Description = "Registro oficial de distinción, balance de tesorería y estipendio diario."
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Rango Soberano", Value: rank, Inline: true},
				{Name: "Nivel de Honor", Value: fmt.Sprintf("Nivel %d", level), Inline: true},
				{Name: "Monedas Soberanas", Value: fmt.Sprintf("🪙 %d", panel.Balance), Inline: true},
				{Name: "Progreso Protocolar", Value: fmt.Sprintf("%d / 100 XP", progress), Inline: true},
			}
		}
	case pageInventory:
		if english {
			embed.Title = "🎒・Your inventory"
			embed.Description = "Your personal items are visible only to you in this panel."
			if len(panel.Inventory) == 0 {
				embed.Fields = []*discordgo.MessageEmbedField{{Name: "Empty", Value: "You don't own any special items yet. Visit `/tienda` to browse the Lux collection."}}
			} else {
				for _, item := range panel.Inventory {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name: item.Name, Value: fmt.Sprintf("Quantity: **%d**", item.Quantity), Inline: true,
					})
				}
			}
		} else {
			embed.Title = "🎒・Tu inventario"
			embed.Description = "Tus objetos son privados y solo tú puedes verlos en este panel."
			if len(panel.Inventory) == 0 {
				embed.Fields = []*discordgo.MessageEmbedField{{Name: "Vacío", Value: "Aún no tienes objetos especiales. Usa `/tienda` para explorar la colección del Lux."}}
			} else {
				for _, item := range panel.Inventory {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name: item.Name, Value: fmt.Sprintf("Cantidad: **%d**", item.Quantity), Inline: true,
					})
				}
			}
		}
	case pageSocial:
		embed.Title = "🎭・Club social"
		embed.Description = "Activa la búsqueda de grupo para que otras personas sepan que quieres jugar."
		if english {
			embed.Title = "🎭・Social club"
			embed.Description = "Turn on group search so other members know you want to play."
		}
		members := "Nadie está buscando grupo ahora mismo."
		if english {
			members = "Nobody is looking for a group right now."
		}
		if len(panel.LookingForGroup) > 0 {
			mentions := make([]string, 0, len(panel.LookingForGroup))
			for _, id := range panel.LookingForGroup {
				mentions = append(mentions, "<@"+id+">")
			}
			members = strings.Join(mentions, "\n")
		}
		fieldName := "Buscando grupo"
		if english {
			fieldName = "Looking for a group"
		}
		embed.Fields = []*discordgo.MessageEmbedField{{Name: fieldName, Value: members}}
	case pageSettings:
		embed.Title = "⚙️・Ajustes personales"
		if english {
			embed.Title = "⚙️・Personal settings"
		}
		embed.Description = fmt.Sprintf(
			"Avisos: **%s** · Ocultar inventario: **%s** · Respuestas del bot: **%s** · Idioma del panel: **%s**",
			onOff(panel.Preferences.NotificationsEnabled, false),
			onOff(panel.Preferences.HideInventory, false),
			responseVisibilityName(panel.Preferences.ResponsesPublic, false),
			languageName(panel.Preferences.Language),
		)
		if english {
			embed.Description = fmt.Sprintf(
				"Notifications: **%s** · Hide inventory: **%s** · Bot replies: **%s** · Dashboard language: **%s**",
				onOff(panel.Preferences.NotificationsEnabled, true),
				onOff(panel.Preferences.HideInventory, true),
				responseVisibilityName(panel.Preferences.ResponsesPublic, true),
				languageName(panel.Preferences.Language),
			)
		}
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Los avisos se guardan para futuras funciones compatibles. Tu inventario sigue visible solo para ti en este panel."}
		if english {
			embed.Footer.Text = "Notifications are saved for future compatible features. Your inventory remains visible only to you in this panel."
		}
	}

	return embed, panelComponents(userID, page, panel, english)
}

func panelComponents(userID, page string, panel *storage.LuxPanelData, english bool) []discordgo.MessageComponent {
	selectPlaceholder := "✦ Elige una sección ✦"
	labels := []string{"👤 Perfil", "🎒 Inventario", "🎭 Club social", "⚙️ Ajustes"}
	if english {
		selectPlaceholder = "✦ Choose a section ✦"
		labels = []string{"👤 Profile", "🎒 Inventory", "🎭 Social club", "⚙️ Settings"}
	}
	options := make([]discordgo.SelectMenuOption, 0, 4)
	for idx, value := range []string{pageProfile, pageInventory, pageSocial, pageSettings} {
		options = append(options, discordgo.SelectMenuOption{Label: labels[idx], Value: value, Default: page == value})
	}
	rows := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.StringSelectMenu,
				CustomID:    "lux:" + userID + ":" + page + ":select",
				Placeholder: selectPlaceholder,
				Options:     options,
				MinValues:   intPtr(1),
				MaxValues:   1,
			},
		}},
	}
	switch page {
	case pageProfile:
		label := "🎁 Reclamar recompensa diaria"
		if english {
			label = "🎁 Claim daily reward"
		}
		rows = append(rows, buttonRow(userID, page, "daily", label, discordgo.SuccessButton))
	case pageSocial:
		label := "🔎 Dejar de buscar grupo"
		if !panel.Preferences.LookingForGroup {
			label = "🔎 Buscar grupo"
		}
		if english {
			label = "🔎 Stop looking for a group"
			if !panel.Preferences.LookingForGroup {
				label = "🔎 Look for a group"
			}
		}
		rows = append(rows, buttonRow(userID, page, "looking_for_group", label, discordgo.PrimaryButton))
		dailyLabel := "🎁 Recompensa diaria"
		if english {
			dailyLabel = "🎁 Daily reward"
		}
		rows = append(rows, buttonRow(userID, page, "daily", dailyLabel, discordgo.SuccessButton))
	case pageSettings:
		notificationsLabel := "🔔 Avisos: activados"
		if !panel.Preferences.NotificationsEnabled {
			notificationsLabel = "🔕 Avisos: desactivados"
		}
		hideLabel := "🎒 Mostrar en perfiles públicos"
		if !panel.Preferences.HideInventory {
			hideLabel = "🎒 Ocultar en perfiles públicos"
		}
		languageLabel := "🌐 Cambiar a English"
		if panel.Preferences.Language == "en" {
			languageLabel = "🌐 Cambiar a español"
		}
		visibilityLabel := "👁️ Visibilidad: pública"
		if !panel.Preferences.ResponsesPublic {
			visibilityLabel = "🔒 Visibilidad: solo tú"
		}
		if english {
			notificationsLabel = "🔔 Notifications: on"
			if !panel.Preferences.NotificationsEnabled {
				notificationsLabel = "🔕 Notifications: off"
			}
			hideLabel = "🎒 Show in public profiles"
			if !panel.Preferences.HideInventory {
				hideLabel = "🎒 Hide from public profiles"
			}
			languageLabel = "🌐 Switch to Spanish"
			if panel.Preferences.Language == "en" {
				languageLabel = "🌐 Switch to English"
			}
			visibilityLabel = "👁️ Visibility: public"
			if !panel.Preferences.ResponsesPublic {
				visibilityLabel = "🔒 Visibility: only you"
			}
		}
		rows = append(rows, buttonRow(userID, page, "notifications", notificationsLabel, discordgo.SecondaryButton))
		rows = append(rows, buttonRow(userID, page, "hide_inventory", hideLabel, discordgo.SecondaryButton))
		rows = append(rows, buttonRow(userID, page, "language", languageLabel, discordgo.SecondaryButton))
		rows = append(rows, buttonRow(userID, page, "responses_public", visibilityLabel, discordgo.PrimaryButton))
	}
	return rows
}

func buttonRow(userID, page, action, label string, style discordgo.ButtonStyle) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			CustomID: "lux:" + userID + ":" + page + ":" + action,
			Label:    label,
			Style:    style,
		},
	}}
}

func respondPanelError(session *discordgo.Session, interaction *discordgo.InteractionCreate, message string) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: message, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func followupPanelError(session *discordgo.Session, interaction *discordgo.InteractionCreate, message string) error {
	_, err := discordutil.FollowupMessageCreate(session, interaction.Interaction, true, &discordgo.WebhookParams{
		Content: message,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return err
}

func validPage(page string) bool {
	return page == pageProfile || page == pageInventory || page == pageSocial || page == pageSettings
}

func rankForLevel(level int64, english bool) string {
	switch {
	case level >= 20:
		if english {
			return "👑 Imperial Dignitary"
		}
		return "👑 Dignatario Imperial"
	case level >= 10:
		if english {
			return "💎 Grand Sovereign"
		}
		return "💎 Gran Soberano"
	case level >= 5:
		if english {
			return "⚜️ Distinguished Member"
		}
		return "⚜️ Miembro Distinguido"
	default:
		if english {
			return "✨ Honorable Citizen"
		}
		return "✨ Ciudadano de Honor"
	}
}

func onOff(enabled, english bool) string {
	if english {
		if enabled {
			return "on"
		}
		return "off"
	}
	if enabled {
		return "activados"
	}
	return "desactivados"
}

func languageName(language string) string {
	if language == "en" {
		return "English"
	}
	return "Español"
}

func responseVisibilityName(public, english bool) string {
	if english {
		if public {
			return "public"
		}
		return "private"
	}
	if public {
		return "pública"
	}
	return "privada"
}

func intPtr(value int) *int {
	return &value
}
