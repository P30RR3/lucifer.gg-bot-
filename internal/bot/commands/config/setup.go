package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

const (
	setupPermission = discordgo.PermissionViewChannel |
		discordgo.PermissionSendMessages |
		discordgo.PermissionEmbedLinks
	maxWelcomeTemplateLength = 1500
	defaultWelcomeTemplate   = "Bienvenido a **{servidor}**, {usuario}.\n\nLa noche es joven, los deseos están sobre la mesa y el club tiene una sola regla: trata a los demás con respeto. Pasa por los canales fijados para conocer la casa. — *Lucifer*"
)

func (c *Cog) handleSetup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.GuildID == "" {
		return errors.New("este asistente solo funciona dentro de un servidor")
	}

	if env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}

	return showSetupPanel(ctx, session, interaction, env, "Elige los canales que quieres configurar. Cada selección se guarda de inmediato.")
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("configuracion", handleSetupComponent)
	router.RegisterModal("configuracion_mensaje", handleWelcomeMessageModal)
	router.RegisterModal("configuracion_imagen", handleWelcomeImageModal)
}

func showSetupPanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, status string) error {
	guildSettings, err := env.Store.GetOrCreateGuild(ctx, interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar configuración del servidor: %w", err)
	}
	auditSettings, err := env.Store.GetAuditSettings(ctx, interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar configuración de auditoría: %w", err)
	}

	guild, err := session.Guild(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar servidor de Discord: %w", err)
	}

	welcomeChannel := setupChannelLabel(guildSettings.WelcomeChannel)
	logChannel := setupChannelLabel(auditSettings.LogChannelID)
	embed := &discordgo.MessageEmbed{
		Title:       "Configuración del servidor",
		Description: status,
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Canal de bienvenida", Value: welcomeChannel, Inline: true},
			{Name: "Mensaje de bienvenida", Value: welcomeTemplateSummary(guildSettings.WelcomeMessage), Inline: true},
			{Name: "Imagen de bienvenida", Value: welcomeImageSummary(configuredWelcomeImage(guildSettings, env)), Inline: true},
			{Name: "Canal de auditoría", Value: logChannel, Inline: true},
			{Name: "Comprobación", Value: "Pulsa **Probar configuración** para revisar canales y permisos del bot.", Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%s · Los cambios se guardan automáticamente", guild.Name),
		},
	}
	env.Theme.BrandEmbed(embed)

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.ChannelSelectMenu,
				CustomID:    setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "welcome"),
				Placeholder: "Selecciona el canal de bienvenida",
				MinValues:   setupIntPtr(1),
				MaxValues:   1,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
				},
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.ChannelSelectMenu,
				CustomID:    setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "logs"),
				Placeholder: "Selecciona el canal de auditoría",
				MinValues:   setupIntPtr(1),
				MaxValues:   1,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
				},
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Editar bienvenida",
				Style:    discordgo.PrimaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "edit-message"),
			},
			discordgo.Button{
				Label:    "Vista previa",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "preview"),
			},
			discordgo.Button{
				Label:    "Enviar prueba",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "test-message"),
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Personalizar imagen",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "edit-image"),
			},
			discordgo.Button{
				Label:    "Probar configuración",
				Style:    discordgo.PrimaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "check"),
			},
			discordgo.Button{
				Label:    "Desactivar bienvenida",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "clear-welcome"),
			},
			discordgo.Button{
				Label:    "Quitar auditoría",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "clear-logs"),
			},
			discordgo.Button{
				Label:    "Restaurar imagen",
				Style:    discordgo.SecondaryButton,
				CustomID: setupCustomID(interaction.GuildID, setupInteractionUserID(interaction), "clear-image"),
			},
		}},
	}

	if interaction.Type == discordgo.InteractionApplicationCommand {
		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func handleSetupComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 4 || parts[1] != interaction.GuildID || parts[2] != setupInteractionUserID(interaction) {
		return respondSetupEphemeral(session, interaction, env, "Este panel pertenece a otra persona o servidor.")
	}
	if interaction.Member == nil || interaction.Member.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageGuild) == 0 {
		return respondSetupEphemeral(session, interaction, env, "Necesitas el permiso Administrar servidor para usar este panel.")
	}
	if env == nil || env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}

	action := parts[3]
	switch action {
	case "edit-message":
		settings, err := env.Store.GetOrCreateGuild(ctx, interaction.GuildID)
		if err != nil {
			return fmt.Errorf("cargar mensaje de bienvenida: %w", err)
		}
		template := settings.WelcomeMessage
		if strings.TrimSpace(template) == "" {
			template = defaultWelcomeTemplate
		}
		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("configuracion_mensaje:%s:%s", interaction.GuildID, setupInteractionUserID(interaction)),
				Title:    "Editar bienvenida",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "message",
							Label:       "Mensaje · usa {usuario} y {servidor}",
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							MinLength:   1,
							MaxLength:   maxWelcomeTemplateLength,
							Value:       template,
							Placeholder: "Bienvenido a {servidor}, {usuario}.",
						},
					}},
				},
			},
		})
	case "edit-image":
		settings, err := env.Store.GetOrCreateGuild(ctx, interaction.GuildID)
		if err != nil {
			return fmt.Errorf("cargar imagen de bienvenida: %w", err)
		}
		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("configuracion_imagen:%s:%s", interaction.GuildID, setupInteractionUserID(interaction)),
				Title:    "Imagen de bienvenida",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "image_url",
							Label:       "URL HTTPS de una imagen directa",
							Style:       discordgo.TextInputShort,
							Required:    false,
							MaxLength:   1000,
							Value:       settings.WelcomeImageURL,
							Placeholder: "https://ejemplo.com/imagen.png",
						},
					}},
				},
			},
		})
	case "preview":
		return sendWelcomePreview(ctx, session, interaction, env, false)
	case "test-message":
		return sendWelcomePreview(ctx, session, interaction, env, true)
	case "welcome", "logs":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 {
			return respondSetupEphemeral(session, interaction, env, "Selecciona exactamente un canal.")
		}
		channel, err := session.Channel(values[0])
		if err != nil || channel == nil {
			return respondSetupEphemeral(session, interaction, env, "No pude consultar ese canal. Comprueba que siga disponible.")
		}
		if channel.GuildID != interaction.GuildID || channel.Type != discordgo.ChannelTypeGuildText {
			return respondSetupEphemeral(session, interaction, env, "El canal debe ser de texto y pertenecer a este servidor.")
		}
		if action == "welcome" {
			_, err = env.Store.PatchGuild(ctx, interaction.GuildID, func(settings *storage.GuildSettings) {
				settings.WelcomeChannel = channel.ID
			})
			if err != nil {
				return fmt.Errorf("guardar canal de bienvenida: %w", err)
			}
			return showSetupPanel(ctx, session, interaction, env, fmt.Sprintf("Canal de bienvenida guardado: <#%s>.", channel.ID))
		}

		settings, err := env.Store.GetAuditSettings(ctx, interaction.GuildID)
		if err != nil {
			return fmt.Errorf("cargar configuración de auditoría: %w", err)
		}
		settings.GuildID = interaction.GuildID
		settings.LogChannelID = channel.ID
		if err := env.Store.SaveAuditSettings(ctx, settings); err != nil {
			return fmt.Errorf("guardar canal de auditoría: %w", err)
		}
		if env.Auditor != nil {
			env.Auditor.UpdateSettings(settings)
		}
		return showSetupPanel(ctx, session, interaction, env, fmt.Sprintf("Canal de auditoría guardado: <#%s>.", channel.ID))
	case "clear-welcome":
		_, err := env.Store.PatchGuild(ctx, interaction.GuildID, func(settings *storage.GuildSettings) {
			settings.WelcomeChannel = ""
		})
		if err != nil {
			return fmt.Errorf("desactivar canal de bienvenida: %w", err)
		}
		return showSetupPanel(ctx, session, interaction, env, "La bienvenida pública quedó desactivada. Se mantiene la bienvenida por mensaje directo.")
	case "clear-logs":
		settings, err := env.Store.GetAuditSettings(ctx, interaction.GuildID)
		if err != nil {
			return fmt.Errorf("cargar configuración de auditoría: %w", err)
		}
		settings.GuildID = interaction.GuildID
		settings.LogChannelID = ""
		if err := env.Store.SaveAuditSettings(ctx, settings); err != nil {
			return fmt.Errorf("desvincular canal de auditoría: %w", err)
		}
		if env.Auditor != nil {
			env.Auditor.UpdateSettings(settings)
		}
		return showSetupPanel(ctx, session, interaction, env, "El canal de auditoría quedó desvinculado.")
	case "clear-image":
		_, err := env.Store.PatchGuild(ctx, interaction.GuildID, func(settings *storage.GuildSettings) {
			settings.WelcomeImageURL = ""
		})
		if err != nil {
			return fmt.Errorf("restaurar imagen de bienvenida: %w", err)
		}
		return showSetupPanel(ctx, session, interaction, env, "Imagen restaurada. Se usará la imagen predeterminada; si no existe, el fondo infernal con el avatar redondo.")
	case "check":
		result, err := checkSetup(ctx, session, interaction.GuildID, env)
		if err != nil {
			return fmt.Errorf("comprobar configuración: %w", err)
		}
		return showSetupPanel(ctx, session, interaction, env, result)
	default:
		return respondSetupEphemeral(session, interaction, env, "Esta acción de configuración ya no está disponible.")
	}
}

func handleWelcomeImageModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || interaction.Member == nil || interaction.Member.User == nil ||
		interaction.GuildID != parts[1] || interaction.Member.User.ID != parts[2] ||
		interaction.Member.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageGuild) == 0 {
		return respondSetupEphemeral(session, interaction, env, "No tienes autorización para editar la imagen de bienvenida.")
	}
	if env == nil || env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}

	imageURL := strings.TrimSpace(setupModalText(interaction.ModalSubmitData().Components, "image_url"))
	if imageURL != "" {
		if err := validateWelcomeImageURL(imageURL); err != nil {
			return respondSetupEphemeral(session, interaction, env, err.Error())
		}
	}
	_, err := env.Store.PatchGuild(ctx, interaction.GuildID, func(settings *storage.GuildSettings) {
		settings.WelcomeImageURL = imageURL
	})
	if err != nil {
		return fmt.Errorf("guardar imagen de bienvenida: %w", err)
	}
	status := "Imagen de bienvenida guardada."
	if imageURL == "" {
		status = "Imagen personalizada eliminada. Se usará el fondo infernal con el avatar redondo."
	}
	return showSetupPanel(ctx, session, interaction, env, status)
}

func validateWelcomeImageURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return errors.New("La imagen debe ser una URL HTTPS válida y directa.")
	}
	return nil
}

func handleWelcomeMessageModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || interaction.Member == nil || interaction.Member.User == nil ||
		interaction.GuildID != parts[1] || interaction.Member.User.ID != parts[2] ||
		interaction.Member.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageGuild) == 0 {
		return respondSetupEphemeral(session, interaction, env, "No tienes autorización para editar la bienvenida de este servidor.")
	}
	if env == nil || env.Store == nil || env.Theme == nil {
		return errors.New("el almacenamiento o el tema del bot no están disponibles")
	}

	message := strings.TrimSpace(setupModalText(interaction.ModalSubmitData().Components, "message"))
	if len([]rune(message)) == 0 || len([]rune(message)) > maxWelcomeTemplateLength {
		return respondSetupEphemeral(session, interaction, env, "El mensaje debe tener entre 1 y 1.500 caracteres.")
	}

	_, err := env.Store.PatchGuild(ctx, interaction.GuildID, func(settings *storage.GuildSettings) {
		settings.WelcomeMessage = message
	})
	if err != nil {
		return fmt.Errorf("guardar mensaje de bienvenida: %w", err)
	}
	return showSetupPanel(ctx, session, interaction, env, "Mensaje de bienvenida guardado. Usa **Vista previa** para revisarlo antes de que llegue alguien.")
}

func sendWelcomePreview(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, sendToChannel bool) error {
	settings, err := env.Store.GetOrCreateGuild(ctx, interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar configuración de bienvenida: %w", err)
	}
	guild, err := session.Guild(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("cargar servidor para la vista previa: %w", err)
	}

	var user *discordgo.User
	if interaction.Member != nil {
		user = interaction.Member.User
	}
	if user == nil {
		return respondSetupEphemeral(session, interaction, env, "No pude identificarte para generar la vista previa.")
	}

	imageURL := configuredWelcomeImage(settings, env)
	embed := BuildWelcomeEmbed(env.Theme, guild.Name, user, settings.WelcomeMessage, imageURL)
	files, err := WelcomeImageFiles(imageURL)
	if err != nil {
		return fmt.Errorf("generar fondo de bienvenida: %w", err)
	}
	if !sendToChannel {
		embed.Title = "Vista previa de bienvenida"
		return respondSetupPrivateEmbed(session, interaction, embed, "Así se verá el mensaje de bienvenida:", files)
	}

	if settings.WelcomeChannel == "" {
		return respondSetupEphemeral(session, interaction, env, "Primero selecciona un canal de bienvenida.")
	}
	permissions, err := session.UserChannelPermissions(session.State.User.ID, settings.WelcomeChannel)
	if err != nil {
		return fmt.Errorf("comprobar permisos del canal de bienvenida: %w", err)
	}
	if missing := missingSetupPermissions(permissions); len(missing) > 0 {
		return respondSetupEphemeral(session, interaction, env, fmt.Sprintf("No puedo enviar la prueba; faltan estos permisos en <#%s>: %s.", settings.WelcomeChannel, strings.Join(missing, ", ")))
	}
	embed.Title = "Prueba de bienvenida · " + guild.Name
	_, err = session.ChannelMessageSendComplex(settings.WelcomeChannel, &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{embed},
		Files:           files,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	})
	if err != nil {
		return fmt.Errorf("enviar prueba de bienvenida: %w", err)
	}
	return respondSetupEphemeral(session, interaction, env, fmt.Sprintf("Envié la prueba al canal <#%s>.", settings.WelcomeChannel))
}

func BuildWelcomeEmbed(theme *ui.Theme, guildName string, user *discordgo.User, template, imageURL string) *discordgo.MessageEmbed {
	userMention := "una nueva alma"
	if user != nil {
		userMention = user.Mention()
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🍸・LAS PUERTAS DEL LUX SE ABREN",
		Description: RenderWelcomeMessage(template, userMention, guildName),
		Color:       theme.Primary,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Una entrada digna de Los Ángeles. Disfruta tu estancia."},
	}
	if user != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("256")}
	}
	if imageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: imageURL}
	} else {
		embed.Image = &discordgo.MessageEmbedImage{URL: "attachment://welcome-inferno.png"}
	}
	theme.BrandEmbed(embed)
	return embed
}

func WelcomeImageFiles(imageURL string) ([]*discordgo.File, error) {
	if imageURL != "" {
		return nil, nil
	}
	background, err := infernoBackgroundPNG()
	if err != nil {
		return nil, err
	}
	return []*discordgo.File{{
		Name:   "welcome-inferno.png",
		Reader: bytes.NewReader(background),
	}}, nil
}

func configuredWelcomeImage(settings *storage.GuildSettings, env *bot.Env) string {
	if settings != nil && settings.WelcomeImageURL != "" {
		return settings.WelcomeImageURL
	}
	if env != nil && env.Config != nil {
		return env.Config.UI.WelcomeImageURL
	}
	return ""
}

func ResolveWelcomeImageURL(serverImageURL, defaultImageURL string) string {
	if serverImageURL != "" {
		return serverImageURL
	}
	return defaultImageURL
}

func welcomeImageSummary(imageURL string) string {
	if imageURL == "" {
		return "Fondo infernal integrado · avatar redondo"
	}
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Host == "" {
		return "Imagen personalizada configurada"
	}
	return truncateWelcomeSummary(parsed.Host, 100)
}

func RenderWelcomeMessage(template, userMention, guildName string) string {
	if strings.TrimSpace(template) == "" {
		template = defaultWelcomeTemplate
	}
	message := strings.ReplaceAll(template, "{usuario}", userMention)
	message = strings.ReplaceAll(message, "{servidor}", guildName)
	return message
}

func welcomeTemplateSummary(template string) string {
	if strings.TrimSpace(template) == "" {
		template = defaultWelcomeTemplate
	}
	template = strings.ReplaceAll(template, "{usuario}", "@usuario")
	template = strings.ReplaceAll(template, "{servidor}", "servidor")
	return truncateWelcomeSummary(template, 180)
}

func truncateWelcomeSummary(message string, maxRunes int) string {
	runes := []rune(message)
	if len(runes) <= maxRunes {
		return message
	}
	return string(runes[:maxRunes-1]) + "…"
}

func setupModalText(components []discordgo.MessageComponent, id string) string {
	for _, row := range components {
		actionRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, component := range actionRow.Components {
			input, ok := component.(*discordgo.TextInput)
			if ok && input.CustomID == id {
				return input.Value
			}
		}
	}
	return ""
}

func checkSetup(ctx context.Context, session *discordgo.Session, guildID string, env *bot.Env) (string, error) {
	settings, err := env.Store.GetOrCreateGuild(ctx, guildID)
	if err != nil {
		return "", fmt.Errorf("cargar canal de bienvenida: %w", err)
	}
	audit, err := env.Store.GetAuditSettings(ctx, guildID)
	if err != nil {
		return "", fmt.Errorf("cargar canal de auditoría: %w", err)
	}

	var results []string
	check := func(label, channelID string) {
		if channelID == "" {
			results = append(results, "⚠️ **"+label+":** falta seleccionar un canal.")
			return
		}
		channel, err := session.Channel(channelID)
		if err != nil || channel == nil || channel.GuildID != guildID || channel.Type != discordgo.ChannelTypeGuildText {
			results = append(results, "❌ **"+label+":** el canal configurado ya no existe o no es válido.")
			return
		}
		permissions, err := session.UserChannelPermissions(session.State.User.ID, channelID)
		if err != nil {
			results = append(results, "❌ **"+label+":** no pude consultar los permisos del bot.")
			return
		}
		missing := missingSetupPermissions(permissions)
		if len(missing) > 0 {
			results = append(results, fmt.Sprintf("❌ **%s** en <#%s>: faltan %s.", label, channelID, strings.Join(missing, ", ")))
			return
		}
		results = append(results, fmt.Sprintf("✅ **%s** en <#%s>: canal y permisos listos.", label, channelID))
	}
	check("Bienvenida", settings.WelcomeChannel)
	check("Auditoría", audit.LogChannelID)
	return strings.Join(results, "\n"), nil
}

func missingSetupPermissions(permissions int64) []string {
	if permissions&setupPermission == setupPermission {
		return nil
	}
	var missing []string
	if permissions&discordgo.PermissionViewChannel == 0 {
		missing = append(missing, "Ver canal")
	}
	if permissions&discordgo.PermissionSendMessages == 0 {
		missing = append(missing, "Enviar mensajes")
	}
	if permissions&discordgo.PermissionEmbedLinks == 0 {
		missing = append(missing, "Insertar enlaces")
	}
	return missing
}

func respondSetupEphemeral(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, message string) error {
	data := &discordgo.InteractionResponseData{
		Content: message,
		Flags:   discordgo.MessageFlagsEphemeral,
	}
	if env != nil && env.Theme != nil {
		data.Content = ""
		data.Embeds = []*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Configuración", message)}
	}
	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func respondSetupPrivateEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, message string, files []*discordgo.File) error {
	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Embeds:  []*discordgo.MessageEmbed{embed},
			Files:   files,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func setupCustomID(guildID, userID, action string) string {
	return fmt.Sprintf("configuracion:%s:%s:%s", guildID, userID, action)
}

func setupInteractionUserID(interaction *discordgo.InteractionCreate) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return interaction.Member.User.ID
	}
	if interaction.User != nil {
		return interaction.User.ID
	}
	return ""
}

func setupChannelLabel(channelID string) string {
	if channelID == "" {
		return "No configurado"
	}
	return "<#" + channelID + ">"
}

func setupIntPtr(value int) *int {
	return &value
}
