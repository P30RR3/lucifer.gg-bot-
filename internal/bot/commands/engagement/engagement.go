package engagement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/pkg/discordutil"
)

const (
	therapyHistory     = "history"
	therapyPrivacy     = "privacy"
	celestialMissions  = "missions"
	celestialLeaders   = "leaderboard"
	celestialBlessings = "blessings"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	if err := registry.Register(&bot.Command{
		Name:        "terapia",
		Description: "Abre tu centro de bienestar, historial y confidencialidad.",
		Category:    "comunidad",
		GuildOnly:   true,
		Handler:     c.handleTherapy,
	}); err != nil {
		return err
	}
	if err := registry.Register(&bot.Command{
		Name:        "celestial",
		Description: "Consulta misiones, clasificación y bendiciones del servidor.",
		Category:    "comunidad",
		GuildOnly:   true,
		Handler:     c.handleCelestial,
	}); err != nil {
		return err
	}
	return registry.Register(&bot.Command{
		Name:        "celestial-configurar",
		Description: "Configura roles que pueden canjearse con puntos celestiales.",
		Category:    "config",
		GuildOnly:   true,
		DefaultPerm: discordgo.PermissionAdministrator,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "rol",
				Description: "Añade o actualiza una bendición de rol.",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionRole, Name: "rol", Description: "Rol que se entregará al canjear", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "precio", Description: "Puntos celestiales necesarios", Required: true, MinValue: floatPtr(1), MaxValue: 100000},
					{Type: discordgo.ApplicationCommandOptionString, Name: "descripcion", Description: "Descripción visible en la tienda", Required: false, MaxLength: 100},
				},
			},
		},
		Handler: c.handleCelestialConfig,
	})
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("therapy", handleTherapyComponent)
	router.RegisterComponent("celestial", handleCelestialComponent)
}

func OnMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate, store *storage.Store, logger *slog.Logger) {
	if event == nil || event.Message == nil || event.Author == nil || event.Author.Bot || event.GuildID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := store.RecordMemberMessage(ctx, event.GuildID, event.Author.ID, event.ChannelID, event.Content); err != nil && logger != nil {
		logger.Warn("no se pudo registrar la actividad de texto", "guild_id", event.GuildID, "user_id", event.Author.ID, "err", err)
	}
}

func OnVoiceStateUpdate(_ *discordgo.Session, event *discordgo.VoiceStateUpdate, store *storage.Store, logger *slog.Logger) {
	if event == nil || event.GuildID == "" || event.UserID == "" || (event.Member != nil && event.Member.User != nil && event.Member.User.Bot) {
		return
	}
	oldChannel := ""
	if event.BeforeUpdate != nil {
		oldChannel = event.BeforeUpdate.ChannelID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := store.RecordMemberVoiceTransition(ctx, event.GuildID, event.UserID, oldChannel, event.ChannelID, time.Now()); err != nil && logger != nil {
		logger.Warn("no se pudo registrar actividad de voz", "guild_id", event.GuildID, "user_id", event.UserID, "err", err)
	}
}

func (c *Cog) handleTherapy(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if err := deferPrivatePanel(session, interaction); err != nil {
		return fmt.Errorf("iniciar panel /terapia: %w", err)
	}
	userID := interaction.Member.User.ID
	stats, err := env.Store.LoadMemberActivityStats(ctx, interaction.GuildID, userID)
	if err != nil {
		return fmt.Errorf("cargar historial personal: %w", err)
	}
	embed, components := renderTherapy(userID, therapyHistory, stats, env.Theme.Primary, "")
	env.Theme.BrandEmbed(embed)
	if err := editPanel(session, interaction, embed, components); err != nil {
		return fmt.Errorf("mostrar panel /terapia: %w", err)
	}
	return nil
}

func handleTherapyComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 4 || interaction.Member == nil || interaction.Member.User == nil {
		return replyPanelError(session, interaction, env, "No pude verificar este panel. Ejecuta `/terapia` para abrirlo de nuevo.")
	}
	userID := interaction.Member.User.ID
	if userID != parts[1] {
		return replyPanelError(session, interaction, env, "Este panel es privado; ejecuta `/terapia` para consultar tu propio historial.")
	}
	page := parts[2]
	action := parts[3]
	if !validTherapyPage(page) {
		return replyPanelError(session, interaction, env, "Esa sección del panel ya no está disponible.")
	}
	if err := deferPanelUpdate(session, interaction); err != nil {
		return fmt.Errorf("iniciar actualización de /terapia: %w", err)
	}
	status := ""
	switch action {
	case "select":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 || !validTherapyPage(values[0]) {
			return finishPanelError(session, interaction, env, "Esa sección del panel no existe.")
		}
		page = values[0]
	case "tracking", "stats_public", "response_mode":
		if err := env.Store.ToggleActivityPreference(ctx, interaction.GuildID, userID, action); err != nil {
			return fmt.Errorf("actualizar confidencialidad personal: %w", err)
		}
		status = "Preferencias de confidencialidad actualizadas."
		page = therapyPrivacy
	default:
		return finishPanelError(session, interaction, env, "La acción de este panel no es válida.")
	}
	stats, err := env.Store.LoadMemberActivityStats(ctx, interaction.GuildID, userID)
	if err != nil {
		return fmt.Errorf("actualizar historial personal: %w", err)
	}
	embed, components := renderTherapy(userID, page, stats, env.Theme.Primary, status)
	env.Theme.BrandEmbed(embed)
	if err := editPanel(session, interaction, embed, components); err != nil {
		return fmt.Errorf("actualizar panel /terapia: %w", err)
	}
	return nil
}

func (c *Cog) handleCelestial(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if err := deferPrivatePanel(session, interaction); err != nil {
		return fmt.Errorf("iniciar panel /celestial: %w", err)
	}
	userID := interaction.Member.User.ID
	embed, components, err := renderCelestial(ctx, env, interaction.GuildID, userID, celestialMissions, "messages", "")
	if err != nil {
		return err
	}
	env.Theme.BrandEmbed(embed)
	if err := editPanel(session, interaction, embed, components); err != nil {
		return fmt.Errorf("mostrar panel /celestial: %w", err)
	}
	return nil
}

func handleCelestialComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 4 || interaction.Member == nil || interaction.Member.User == nil {
		return replyPanelError(session, interaction, env, "No pude verificar este panel. Ejecuta `/celestial` para abrirlo de nuevo.")
	}
	userID := interaction.Member.User.ID
	if userID != parts[1] {
		return replyPanelError(session, interaction, env, "Este panel es privado; abre el tuyo con `/celestial`.")
	}
	page, action := parts[2], parts[3]
	if !validCelestialPage(page) {
		return replyPanelError(session, interaction, env, "Esa sección del panel ya no está disponible.")
	}
	filter := "messages"
	if len(parts) > 4 {
		filter = parts[4]
	}
	if err := deferPanelUpdate(session, interaction); err != nil {
		return fmt.Errorf("iniciar actualización de /celestial: %w", err)
	}
	status := ""
	var err error
	switch action {
	case "select":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 || !validCelestialPage(values[0]) {
			return finishPanelError(session, interaction, env, "Esa sección del panel no existe.")
		}
		page = values[0]
	case "filter":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 || !validLeaderboardCategory(values[0]) {
			return finishPanelError(session, interaction, env, "Esa categoría no existe.")
		}
		page, filter = celestialLeaders, values[0]
	case "claim":
		if len(parts) < 5 {
			return finishPanelError(session, interaction, env, "No pude identificar la misión.")
		}
		balance, err := env.Store.ClaimCelestialMission(ctx, interaction.GuildID, userID, parts[4], time.Now())
		switch {
		case errors.Is(err, storage.ErrMissionNotReady):
			status = "Aún no has completado esta misión."
		case errors.Is(err, storage.ErrMissionAlreadyClaimed):
			status = "Ya reclamaste la recompensa de esta misión."
		case err != nil:
			return fmt.Errorf("reclamar misión celestial: %w", err)
		default:
			status = fmt.Sprintf("Recompensa reclamada. Tu saldo es **%d puntos**.", balance)
		}
		page = celestialMissions
	case "redeem":
		values := interaction.MessageComponentData().Values
		if len(values) != 1 {
			return finishPanelError(session, interaction, env, "Elige una bendición para canjear.")
		}
		status, err = redeemRole(ctx, session, env, interaction, userID, values[0])
		if err != nil {
			return fmt.Errorf("canjear bendición celestial: %w", err)
		}
		page = celestialBlessings
	default:
		return finishPanelError(session, interaction, env, "La acción de este panel no es válida.")
	}
	embed, components, err := renderCelestial(ctx, env, interaction.GuildID, userID, page, filter, status)
	if err != nil {
		return err
	}
	env.Theme.BrandEmbed(embed)
	if err := editPanel(session, interaction, embed, components); err != nil {
		return fmt.Errorf("actualizar panel /celestial: %w", err)
	}
	return nil
}

func (c *Cog) handleCelestialConfig(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		return respondEmbed(session, interaction, env.Theme.ErrorEmbed("Acceso restringido", "Solo administradores pueden configurar bendiciones."), true)
	}
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 || options[0].Name != "rol" {
		return respondEmbed(session, interaction, env.Theme.ErrorEmbed("Configuración inválida", "Selecciona la opción `rol`."), true)
	}
	role := options[0].Options[0].RoleValue(session, interaction.GuildID)
	if role == nil || role.ID == interaction.GuildID || role.Managed {
		return respondEmbed(session, interaction, env.Theme.ErrorEmbed("Rol no válido", "Elige un rol normal del servidor, no `@everyone` ni un rol administrado por una integración."), true)
	}
	botMember, err := session.GuildMember(interaction.GuildID, session.State.User.ID)
	if err != nil {
		return fmt.Errorf("consultar jerarquía de roles del bot: %w", err)
	}
	roles, err := session.GuildRoles(interaction.GuildID)
	if err != nil {
		return fmt.Errorf("consultar roles del servidor: %w", err)
	}
	highestPosition := 0
	for _, assignedID := range botMember.Roles {
		for _, guildRole := range roles {
			if guildRole.ID == assignedID && guildRole.Position > highestPosition {
				highestPosition = guildRole.Position
			}
		}
	}
	if role.Position >= highestPosition {
		return respondEmbed(session, interaction, env.Theme.ErrorEmbed("Jerarquía de roles", "Mueve el rol del bot por encima del rol que quieres ofrecer en la tienda."), true)
	}
	name := role.Name
	description := ""
	price := int64(0)
	for _, option := range options[0].Options {
		switch option.Name {
		case "precio":
			price = option.IntValue()
		case "descripcion":
			description = option.StringValue()
		}
	}
	if err := env.Store.ConfigureCelestialRole(ctx, interaction.GuildID, role.ID, name, description, price); err != nil {
		return fmt.Errorf("guardar bendición de rol: %w", err)
	}
	embed := env.Theme.SuccessEmbed(
		"Bendición configurada",
		fmt.Sprintf("**%s** está disponible por **%d puntos celestiales**. El bot necesita permiso para gestionar roles.", role.Name, price),
	)
	return respondEmbed(session, interaction, embed, true)
}

func renderTherapy(userID, page string, stats *storage.MemberActivityStats, color int, status string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{Color: color}
	switch page {
	case therapyHistory:
		embed.Title = "🕯️ Mi historial"
		embed.Description = "Un resumen personal de actividad. Solo se guardan contadores agregados; el bot no conserva el texto de tus mensajes. La visibilidad de este panel sigue el ajuste de `/lux`."
		channel := "Aún no hay un canal destacado."
		if stats.FavoriteChannelID != "" {
			channel = "<#" + stats.FavoriteChannelID + ">"
		}
		words := "Aún no hay suficientes datos."
		if len(stats.TopWords) > 0 {
			parts := make([]string, 0, len(stats.TopWords))
			for _, word := range stats.TopWords {
				parts = append(parts, fmt.Sprintf("`%s` · %d", word.Word, word.Occurrences))
			}
			words = strings.Join(parts, "\n")
		}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Mensajes registrados", Value: fmt.Sprint(stats.Messages), Inline: true},
			{Name: "Hora más activa (UTC)", Value: fmt.Sprintf("%02d:00–%02d:00", stats.ActiveHourUTC, (stats.ActiveHourUTC+1)%24), Inline: true},
			{Name: "Canal favorito", Value: channel, Inline: true},
			{Name: "Tiempo en voz", Value: formatDuration(stats.VoiceSeconds), Inline: true},
			{Name: "Palabras más usadas", Value: words, Inline: false},
		}
		if !stats.Preferences.TrackingEnabled {
			embed.Description += "\n\nEl seguimiento está desactivado. Actívalo en **Confidencialidad** para empezar a registrar tu actividad."
		}
	case therapyPrivacy:
		embed.Title = "🔐 Confidencialidad"
		embed.Description = "El ajuste de estadísticas controla tu aparición en los rankings. La visibilidad del historial se configura en `/lux`."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Seguimiento de actividad", Value: enabledText(stats.Preferences.TrackingEnabled), Inline: true},
			{Name: "Aparecer en rankings", Value: enabledText(stats.Preferences.StatsPublic), Inline: true},
			{Name: "Respuestas de IA", Value: responseModeText(stats.Preferences.ResponseMode), Inline: true},
		}
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "El seguimiento está desactivado de forma predeterminada. Al apagarlo se conservan las estadísticas anteriores."}
	}
	if status != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "✦ Preferencias", Value: status})
	}
	return embed, therapyComponents(userID, page, stats.Preferences)
}

func therapyComponents(userID, page string, preferences storage.ActivityPreferences) []discordgo.MessageComponent {
	options := []discordgo.SelectMenuOption{
		{Label: "Mi historial", Value: therapyHistory, Default: page == therapyHistory},
		{Label: "Confidencialidad", Value: therapyPrivacy, Default: page == therapyPrivacy},
	}
	rows := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "therapy:" + userID + ":" + page + ":select", Placeholder: "Elige una sección", Options: options, MinValues: intPtr(1), MaxValues: 1},
	}}}
	if page == therapyPrivacy {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{CustomID: "therapy:" + userID + ":" + page + ":tracking", Label: toggleLabel("Seguimiento", preferences.TrackingEnabled), Style: discordgo.SecondaryButton},
			discordgo.Button{CustomID: "therapy:" + userID + ":" + page + ":stats_public", Label: toggleLabel("Aparición en rankings", preferences.StatsPublic), Style: discordgo.SecondaryButton},
			discordgo.Button{CustomID: "therapy:" + userID + ":" + page + ":response_mode", Label: "Respuestas: " + responseModeShort(preferences.ResponseMode), Style: discordgo.PrimaryButton},
		}})
	}
	return rows
}

func renderCelestial(ctx context.Context, env *bot.Env, guildID, userID, page, filter, status string) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	balance, err := env.Store.CelestialBalance(ctx, guildID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("consultar puntos celestiales: %w", err)
	}
	embed := &discordgo.MessageEmbed{Color: env.Theme.Primary, Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Saldo: %d puntos celestiales", balance)}}
	switch page {
	case celestialMissions:
		embed.Title = "✨ Tablón de misiones"
		embed.Description = "Completa objetivos diarios y semanales para reclamar puntos."
		preferences, err := env.Store.GetActivityPreferences(ctx, guildID, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("consultar seguimiento para misiones: %w", err)
		}
		if !preferences.TrackingEnabled {
			embed.Description += "\nActiva el seguimiento desde `/terapia` → **Confidencialidad** para empezar a medir progreso."
		}
		missions, err := env.Store.CelestialMissions(ctx, guildID, userID, time.Now())
		if err != nil {
			return nil, nil, fmt.Errorf("consultar misiones celestiales: %w", err)
		}
		components := celestialComponents(userID, page, filter, nil)
		claimButtons := make([]discordgo.MessageComponent, 0, len(missions))
		for _, mission := range missions {
			state := fmt.Sprintf("Progreso: **%d/%d** · Recompensa: **%d puntos**", mission.Current, mission.Target, mission.Points)
			if mission.Claimed {
				state = "✅ Recompensa reclamada en este periodo."
			} else if mission.Current >= mission.Target {
				state += "\nPulsa el botón para reclamarla."
				claimButtons = append(claimButtons, discordgo.Button{
					CustomID: "celestial:" + userID + ":" + page + ":claim:" + mission.ID,
					Label:    "Reclamar " + mission.Name,
					Style:    discordgo.SuccessButton,
				})
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name: mission.Name, Value: mission.Description + "\n" + state,
			})
		}
		if len(claimButtons) > 0 {
			components = append(components, discordgo.ActionsRow{Components: claimButtons})
		}
		if status != "" {
			embed.Description += "\n\n" + status
		}
		return embed, components, nil
	case celestialLeaders:
		embed.Title = "🏆 Salón de fama"
		embed.Description = "La clasificación muestra únicamente a miembros que hicieron públicas sus estadísticas en `/terapia`."
		leaders, err := env.Store.ListEngagementLeaders(ctx, guildID, filter)
		if err != nil {
			return nil, nil, fmt.Errorf("consultar salón de fama: %w", err)
		}
		if len(leaders) == 0 {
			embed.Fields = []*discordgo.MessageEmbedField{{Name: "Sin resultados", Value: "Aún no hay estadísticas públicas en esta categoría."}}
		} else {
			lines := make([]string, 0, len(leaders))
			for idx, leader := range leaders {
				score := fmt.Sprintf("%d mensajes", leader.Score)
				switch filter {
				case "voice":
					score = formatDuration(leader.Score)
				case "help":
					score = fmt.Sprintf("%d respuestas de ayuda", leader.Score)
				}
				lines = append(lines, fmt.Sprintf("**%d.** <@%s> · %s", idx+1, leader.UserID, score))
			}
			embed.Fields = []*discordgo.MessageEmbedField{{Name: leaderboardTitle(filter), Value: strings.Join(lines, "\n")}}
		}
		if status != "" {
			embed.Description += "\n\n" + status
		}
		return embed, celestialComponents(userID, page, filter, nil), nil
	case celestialBlessings:
		embed.Title = "🪽 Bendiciones"
		embed.Description = "Canjea puntos por roles configurados por la administración. Los permisos de cada rol los controla el servidor."
		rewards, err := env.Store.ListCelestialRewards(ctx, guildID)
		if err != nil {
			return nil, nil, fmt.Errorf("consultar bendiciones: %w", err)
		}
		if len(rewards) == 0 {
			embed.Fields = []*discordgo.MessageEmbedField{{Name: "La tienda está vacía", Value: "Un administrador puede añadir roles con `/celestial-configurar rol`."}}
		}
		for _, reward := range rewards {
			value := fmt.Sprintf("<@&%s> · **%d puntos**", reward.RoleID, reward.Price)
			if reward.Description != "" {
				value = reward.Description + "\n" + value
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: reward.Name, Value: value, Inline: true})
		}
		if status != "" {
			embed.Description += "\n\n" + status
		}
		return embed, celestialComponents(userID, page, filter, rewards), nil
	default:
		return nil, nil, errors.New("sección de /celestial no válida")
	}
}

func celestialComponents(userID, page, filter string, rewards []storage.CelestialReward) []discordgo.MessageComponent {
	pages := []struct{ label, value string }{
		{"Misiones", celestialMissions}, {"Salón de fama", celestialLeaders}, {"Bendiciones", celestialBlessings},
	}
	options := make([]discordgo.SelectMenuOption, 0, len(pages))
	for _, item := range pages {
		options = append(options, discordgo.SelectMenuOption{Label: item.label, Value: item.value, Default: page == item.value})
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "celestial:" + userID + ":" + page + ":select", Placeholder: "Elige una sección", Options: options, MinValues: intPtr(1), MaxValues: 1},
	}}}
	if page == celestialLeaders {
		rankOptions := []discordgo.SelectMenuOption{
			{Label: "Actividad en texto", Value: "messages", Default: filter == "messages"},
			{Label: "Tiempo en voz", Value: "voice", Default: filter == "voice"},
			{Label: "Ayuda a la comunidad", Value: "help", Default: filter == "help"},
		}
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "celestial:" + userID + ":" + page + ":filter", Placeholder: "Filtrar clasificación", Options: rankOptions, MinValues: intPtr(1), MaxValues: 1},
		}})
	}
	if page == celestialBlessings && len(rewards) > 0 {
		rewardOptions := make([]discordgo.SelectMenuOption, 0, len(rewards))
		for _, reward := range rewards {
			label := fmt.Sprintf("%s · %d puntos", reward.Name, reward.Price)
			if len([]rune(label)) > 100 {
				label = string([]rune(label)[:97]) + "..."
			}
			rewardOptions = append(rewardOptions, discordgo.SelectMenuOption{Label: label, Value: reward.Key})
		}
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "celestial:" + userID + ":" + page + ":redeem", Placeholder: "Elige una bendición para canjear", Options: rewardOptions, MinValues: intPtr(1), MaxValues: 1},
		}})
	}
	return components
}

func redeemRole(ctx context.Context, session *discordgo.Session, env *bot.Env, interaction *discordgo.InteractionCreate, userID, rewardKey string) (string, error) {
	member, err := session.GuildMember(interaction.GuildID, userID)
	if err != nil {
		return "", fmt.Errorf("consultar miembro antes del canje: %w", err)
	}
	redemptionID, reward, err := env.Store.BeginCelestialRedemption(ctx, interaction.GuildID, userID, rewardKey)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrInsufficientCelestialPoints):
			return "No tienes puntos suficientes para esta bendición.", nil
		case errors.Is(err, storage.ErrRewardAlreadyOwned):
			return "Ya canjeaste este rol.", nil
		default:
			return "", err
		}
	}
	for _, roleID := range member.Roles {
		if roleID == reward.RoleID {
			if err := env.Store.CancelCelestialRedemption(ctx, redemptionID); err != nil {
				return "", fmt.Errorf("reembolsar rol que ya poseía: %w", err)
			}
			return "Ya tienes ese rol; no se descontaron puntos.", nil
		}
	}
	if err := session.GuildMemberRoleAdd(interaction.GuildID, userID, reward.RoleID); err != nil {
		if refundErr := env.Store.CancelCelestialRedemption(ctx, redemptionID); refundErr != nil {
			return "", errors.Join(fmt.Errorf("entregar rol de recompensa: %w", err), fmt.Errorf("reembolsar puntos del canje: %w", refundErr))
		}
		return "Discord no permitió asignar el rol. Tus puntos fueron reembolsados; revisa los permisos y la jerarquía del bot.", nil
	}
	if err := env.Store.CompleteCelestialRedemption(ctx, redemptionID); err != nil {
		removeErr := session.GuildMemberRoleRemove(interaction.GuildID, userID, reward.RoleID)
		refundErr := env.Store.CancelCelestialRedemption(ctx, redemptionID)
		return "", errors.Join(fmt.Errorf("confirmar canje de rol: %w", err), removeErr, refundErr)
	}
	return fmt.Sprintf("Bendición canjeada: <@&%s>. Se descontaron **%d puntos**.", reward.RoleID, reward.Price), nil
}

func deferPrivatePanel(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
}

func deferPanelUpdate(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

func editPanel(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	_, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Content:    stringPointer(""),
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	return err
}

func replyPanelError(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, message string) error {
	return respondEmbed(session, interaction, env.Theme.ErrorEmbed("Panel privado", message), true)
}

func finishPanelError(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, message string) error {
	_, err := discordutil.FollowupMessageCreate(session, interaction.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Acción no disponible", message)},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
	return err
}

func respondEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: flags},
	})
}

func validTherapyPage(page string) bool {
	return page == therapyHistory || page == therapyPrivacy
}

func validCelestialPage(page string) bool {
	return page == celestialMissions || page == celestialLeaders || page == celestialBlessings
}

func validLeaderboardCategory(category string) bool {
	return category == "messages" || category == "voice" || category == "help"
}

func toggleLabel(name string, enabled bool) string {
	if enabled {
		return "Desactivar " + name
	}
	return "Activar " + name
}

func enabledText(enabled bool) string {
	if enabled {
		return "✅ Activado"
	}
	return "⛔ Desactivado"
}

func responseModeText(mode string) string {
	switch mode {
	case "public":
		return "Público en el canal"
	case "dm":
		return "Mensaje directo"
	default:
		return "Privado (solo tú)"
	}
}

func responseModeShort(mode string) string {
	switch mode {
	case "public":
		return "públicas"
	case "dm":
		return "DM"
	default:
		return "privadas"
	}
}

func leaderboardTitle(category string) string {
	switch category {
	case "voice":
		return "Más tiempo en voz"
	case "help":
		return "Reputación por ayudar"
	default:
		return "Más actividad en texto"
	}
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	duration := time.Duration(seconds) * time.Second
	if duration < time.Minute {
		return "<1 min"
	}
	hours := int64(duration / time.Hour)
	minutes := int64((duration % time.Hour) / time.Minute)
	if hours > 0 {
		return fmt.Sprintf("%d h %d min", hours, minutes)
	}
	return fmt.Sprintf("%d min", minutes)
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
