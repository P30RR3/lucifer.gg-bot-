package sanctions

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
)

func securityCenterCommand() *bot.Command {
	return &bot.Command{
		Name:        "seguridad",
		Description: "🛡️ Abre la consola oficial de alta seguridad: sanciones, apelaciones y contingencia protocolar.",
		Category:    "moderacion",
		GuildOnly:   true,
		Handler:     handleSecurityCenter,
	}
}

func handleSecurityCenter(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "⚠️ **Acceso restringido:** Esta consola se encuentra reservada exclusivamente para autoridades con permisos de moderación.", true)
	}
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return fmt.Errorf("iniciar consola de seguridad: %w", err)
	}
	return showSecurityPanel(session, interaction, env)
}

func RegisterSecurityComponents(router *bot.Router) {
	router.RegisterComponent("security", handleSecurityComponent)
	router.RegisterModal("security_search", handleSecuritySearch)
	router.RegisterModal("security_extend", handleTimeoutExtension)
}

func logSecurityReadFailure(env *bot.Env, subject, guildID string, err error) {
	logger := env.Log
	if logger == nil {
		logger = slog.Default()
	}
	logger.Warn("no se pudo cargar un resumen de seguridad", "subject", subject, "guild_id", guildID, "err", err)
}

func showSecurityPanel(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	banCount := "Sin permiso"
	if hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionBanMembers) {
		bans, err := session.GuildBans(interaction.GuildID, 1000, "", "")
		if err != nil {
			logSecurityReadFailure(env, "baneos", interaction.GuildID, err)
			banCount = "No disponible"
		} else {
			banCount = strconv.Itoa(len(bans))
		}
	}
	timeoutCount := "Sin permiso"
	if hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
		_, timedOut, err := listActiveTimeouts(session, interaction.GuildID, 0)
		if err != nil {
			logSecurityReadFailure(env, "silencios activos", interaction.GuildID, err)
			timeoutCount = "No disponible"
		} else {
			timeoutCount = strconv.Itoa(timedOut)
		}
	}

	lockdownText := "Desactivado"
	if env.Lockdown != nil {
		info := env.Lockdown.Info(interaction.GuildID)
		switch {
		case info.Active:
			lockdownText = fmt.Sprintf("ACTIVO · termina <t:%d:R>", info.Until.Unix())
		case len(info.Channels) > 0:
			lockdownText = fmt.Sprintf("Restauración pendiente · %d canal(es) de texto", len(info.Channels))
		}
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🛡️・CONSOLA DE ALTA SEGURIDAD SOBERANA",
		Description: "Consola protocolar oficial: supervisión de sanciones vigentes, apelaciones formales, auditoría y contingencia de defensa perimetral.",
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⛓️ Baneos Vigentes", Value: banCount, Inline: true},
			{Name: "🤐 Silencios Activos", Value: timeoutCount, Inline: true},
			{Name: "🚨 Bloqueo Perimetral", Value: lockdownText, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Lucifer Protocol • Soberanía y protección perimetral ⚜️"},
	}

	if env.Engine != nil {
		status := env.Engine.SecurityStatus(interaction.GuildID)
		raidStatus := "Sin alerta activa"
		if status.AutoLockdownActive {
			raidStatus = fmt.Sprintf("⚠️ Protocolo antiraid · termina <t:%d:R>", status.AutoLockdownUntil.Unix())
		}
		embed.Fields = append(embed.Fields,
			&discordgo.MessageEmbedField{Name: "👁️ Vigilancia en tiempo real", Value: fmt.Sprintf("Ingresos en 10 s: **%d**\nCanales eliminados (1 min): **%d**\nRoles eliminados (1 min): **%d**\n%s", status.JoinsLastTenSeconds, status.ChannelDeletesLastMin, status.RoleDeletesLastMin, raidStatus)},
		)
	}
	env.Theme.BrandEmbed(embed)
	return respondSecurityPanel(session, interaction, embed, securityPanelComponents(interaction.GuildID))
}

func securityPanelComponents(guildID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Baneos", Emoji: &discordgo.ComponentEmoji{Name: "⛓️"}, Style: discordgo.SecondaryButton, CustomID: "security:view:" + guildID + ":bans"},
			discordgo.Button{Label: "Silencios", Emoji: &discordgo.ComponentEmoji{Name: "🤐"}, Style: discordgo.SecondaryButton, CustomID: "security:view:" + guildID + ":timeouts"},
			discordgo.Button{Label: "Historial", Emoji: &discordgo.ComponentEmoji{Name: "📜"}, Style: discordgo.SecondaryButton, CustomID: "security:search:" + guildID},
			discordgo.Button{Label: "Apelaciones", Emoji: &discordgo.ComponentEmoji{Name: "⚖️"}, Style: discordgo.SecondaryButton, CustomID: "security:view:" + guildID + ":appeals"},
			discordgo.Button{Label: "Bloqueo de pánico", Emoji: &discordgo.ComponentEmoji{Name: "🚨"}, Style: discordgo.DangerButton, CustomID: "security:panic:" + guildID},
		}},
	}
}

func handleSecurityComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "🔐 Esta acción solo está disponible dentro del servidor.", true)
	}
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 3 || parts[0] != "security" || parts[1] == "" || interaction.GuildID != parts[2] ||
		!hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "🔐 No tienes permiso para usar esta consola o el control ya venció.", true)
	}
	action := parts[1]
	switch action {
	case "home":
		return updateSecurityPanel(session, interaction, env, "La consola vuelve a estar lista. Cada control comprobará de nuevo tus permisos.")
	case "view":
		if len(parts) != 4 {
			return respond(session, interaction, "Ese panel de seguridad ya no está disponible.", true)
		}
		return showSecuritySection(ctx, session, interaction, env, parts[3])
	case "search":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionViewAuditLogs) &&
			!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) {
			return respond(session, interaction, "El historial requiere Ver registro de auditoría o Gestionar mensajes.", true)
		}
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "security_search:" + interaction.GuildID,
				Title:    "📜 Buscar expediente",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "user_id", Label: "ID del usuario", Style: discordgo.TextInputShort, Required: true, MinLength: 17, MaxLength: 20},
					}},
				},
			},
		})
	case "panic":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild) {
			return respond(session, interaction, "El bloqueo de emergencia requiere Gestionar servidor.", true)
		}
		if env.Lockdown == nil {
			return respond(session, interaction, "El protocolo de bloqueo no está disponible en este momento.", true)
		}
		info := env.Lockdown.Info(interaction.GuildID)
		if info.Active || len(info.Channels) > 0 {
			if err := env.Lockdown.Deactivate(ctx, interaction.GuildID); err != nil {
				return fmt.Errorf("restaurar permisos tras bloqueo de emergencia: %w", err)
			}
			return updateSecurityPanel(session, interaction, env, "🔓 Bloqueo desactivado; se restauraron los permisos previos de los canales.")
		}
		if err := env.Lockdown.Activate(ctx, interaction.GuildID, 5*time.Minute); err != nil {
			return fmt.Errorf("activar bloqueo de emergencia: %w", err)
		}
		return updateSecurityPanel(session, interaction, env, "🚨 Bloqueo de emergencia activado en los canales de texto durante **5 minutos**. Los permisos originales se restaurarán al finalizar.")
	case "select":
		return handleSecuritySelection(ctx, session, interaction, env)
	case "confirm":
		return handleSecurityRevoke(ctx, session, interaction, env, parts)
	case "extend":
		if len(parts) != 4 || !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
			return respond(session, interaction, "Extender un silencio requiere Moderar miembros.", true)
		}
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("security_extend:%s:%s", interaction.GuildID, parts[3]),
				Title:    "⏳ Extender silencio",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "minutes", Label: "Minutos adicionales (1–40320)", Style: discordgo.TextInputShort, Required: true, MinLength: 1, MaxLength: 5, Placeholder: "30"},
					}},
				},
			},
		})
	case "appeal":
		return handleAppealReview(ctx, session, interaction, env, parts)
	default:
		return respond(session, interaction, "Ese control de seguridad ya no está disponible.", true)
	}
}

func showSecuritySection(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, section string) error {
	embed := &discordgo.MessageEmbed{Color: env.Theme.Primary}
	components := []discordgo.MessageComponent{}
	switch section {
	case "bans":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionBanMembers) {
			return respond(session, interaction, "Consultar o revocar baneos requiere Banear miembros.", true)
		}
		bans, err := session.GuildBans(interaction.GuildID, 1000, "", "")
		if err != nil {
			return fmt.Errorf("listar baneos desde seguridad: %w", err)
		}
		embed.Title = "⛓️・PUERTAS DEL TÁRTARO"
		embed.Description = fmt.Sprintf("Hay **%d** baneos registrados. Selecciona una entrada para revisar el motivo y, si corresponde, revocar la sanción.", len(bans))
		options := make([]discordgo.SelectMenuOption, 0, 25)
		for _, ban := range bans {
			if ban == nil || ban.User == nil {
				continue
			}
			label := ban.User.Username
			if label == "" {
				label = ban.User.ID
			}
			options = append(options, discordgo.SelectMenuOption{Label: truncate(label, 100), Value: "ban:" + ban.User.ID, Description: truncate(ban.Reason, 100)})
			if len(options) == 25 {
				break
			}
		}
		if len(options) > 0 {
			components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{CustomID: "security:select:" + interaction.GuildID, Placeholder: "Elige un baneo para revisar", Options: options},
			}})
		} else {
			embed.Description += "\n\nEl registro está limpio; no hay baneos que mostrar."
		}
	case "timeouts":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
			return respond(session, interaction, "Consultar o quitar silencios requiere Moderar miembros.", true)
		}
		activeMembers, activeCount, err := listActiveTimeouts(session, interaction.GuildID, 25)
		if err != nil {
			return fmt.Errorf("listar silencios desde seguridad: %w", err)
		}
		embed.Title = "🤐・SILENCIOS ACTIVOS"
		options := make([]discordgo.SelectMenuOption, 0, 25)
		for _, member := range activeMembers {
			if member == nil || member.User == nil {
				continue
			}
			name := member.DisplayName()
			options = append(options, discordgo.SelectMenuOption{
				Label:       truncate(name, 100),
				Value:       "timeout:" + member.User.ID,
				Description: "Termina " + member.CommunicationDisabledUntil.Format("02 Jan 15:04 UTC"),
			})
			if len(options) == 25 {
				break
			}
		}
		embed.Description = fmt.Sprintf("Se detectaron **%d** silencios activos. Selecciona una entrada para revisar o retirar la sanción.", activeCount)
		if len(options) > 0 {
			components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{CustomID: "security:select:" + interaction.GuildID, Placeholder: "Elige un silencio para revisar", Options: options},
			}})
		} else {
			embed.Description += "\n\nNo hay silencios activos."
		}
	case "appeals":
		appeals, err := env.Store.ListSanctionAppeals(ctx, interaction.GuildID, "", 10)
		if err != nil {
			return fmt.Errorf("consultar apelaciones: %w", err)
		}
		embed.Title = "⚖️・APELACIONES DEL CLUB"
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Aceptar una apelación retira automáticamente baneos y silencios; expulsiones y advertencias requieren seguimiento manual."}
		if len(appeals) == 0 {
			embed.Description = "No hay apelaciones registradas. El archivo permanece en calma."
		} else {
			lines := make([]string, 0, len(appeals))
			for _, appeal := range appeals {
				lines = append(lines, fmt.Sprintf("**#%d · %s · %s** · <@%s>\n%s\n<t:%d:R>", appeal.ID, appeal.Status, appeal.Sanction, appeal.TargetID, truncate(appeal.Reason, 180), appeal.CreatedAt.Unix()))
			}
			embed.Description = strings.Join(lines, "\n\n")
			for _, appeal := range appeals {
				if appeal.Status != "pending" {
					continue
				}
				components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Aceptar #" + strconv.FormatInt(appeal.ID, 10), Style: discordgo.SuccessButton, CustomID: fmt.Sprintf("security:appeal:%s:%d:accept", interaction.GuildID, appeal.ID)},
					discordgo.Button{Label: "Rechazar #" + strconv.FormatInt(appeal.ID, 10), Style: discordgo.DangerButton, CustomID: fmt.Sprintf("security:appeal:%s:%d:deny", interaction.GuildID, appeal.ID)},
				}})
				if len(components) >= 4 {
					break
				}
			}
		}
	default:
		return respond(session, interaction, "Sección de seguridad desconocida.", true)
	}
	if embed.Footer == nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Solo tú puedes ver esta consola. Los cambios se verifican con permisos de Discord."}
	}
	env.Theme.BrandEmbed(embed)
	components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "Volver al panel", Emoji: &discordgo.ComponentEmoji{Name: "↩️"}, Style: discordgo.SecondaryButton, CustomID: "security:home:" + interaction.GuildID},
	}})
	return updateSecurityMessage(session, interaction, embed, components)
}

func listActiveTimeouts(session *discordgo.Session, guildID string, retain int) ([]*discordgo.Member, int, error) {
	active := make([]*discordgo.Member, 0, 25)
	count := 0
	after := ""
	for {
		members, err := session.GuildMembers(guildID, after, 1000)
		if err != nil {
			return nil, 0, err
		}
		if len(members) == 0 {
			break
		}
		lastID := ""
		for _, member := range members {
			if member == nil || member.User == nil {
				continue
			}
			lastID = member.User.ID
			if member.CommunicationDisabledUntil == nil || !member.CommunicationDisabledUntil.After(time.Now()) {
				continue
			}
			count++
			if retain > 0 && len(active) < retain {
				active = append(active, member)
			}
		}
		if len(members) < 1000 || lastID == "" || lastID == after {
			break
		}
		after = lastID
	}
	return active, count, nil
}

func handleSecuritySelection(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	data := interaction.MessageComponentData()
	if len(data.Values) != 1 {
		return respond(session, interaction, "Selecciona una sola sanción.", true)
	}
	parts := strings.SplitN(data.Values[0], ":", 2)
	if len(parts) != 2 {
		return respond(session, interaction, "La sanción seleccionada no es válida.", true)
	}
	kind, targetID := parts[0], parts[1]
	var required int64
	var description string
	switch kind {
	case "ban":
		required, description = discordgo.PermissionBanMembers, "Revocar este baneo quitará al usuario de la lista de baneados."
		bans, err := session.GuildBans(interaction.GuildID, 1000, "", "")
		if err != nil {
			return fmt.Errorf("verificar baneo seleccionado: %w", err)
		}
		found := false
		for _, ban := range bans {
			if ban != nil && ban.User != nil && ban.User.ID == targetID {
				found = true
				description += "\n**Motivo:** " + safeText(ban.Reason)
				break
			}
		}
		if !found {
			return respond(session, interaction, "Ese baneo ya no existe o no pertenece a este servidor.", true)
		}
	case "timeout":
		required, description = discordgo.PermissionModerateMembers, "Revocar este silencio permitirá que el usuario vuelva a hablar."
		member, err := session.GuildMember(interaction.GuildID, targetID)
		if err != nil {
			return fmt.Errorf("verificar silencio seleccionado: %w", err)
		}
		if member.CommunicationDisabledUntil == nil || !member.CommunicationDisabledUntil.After(time.Now()) {
			return respond(session, interaction, "Ese silencio ya expiró.", true)
		}
		description += "\n**Termina:** <t:" + strconv.FormatInt(member.CommunicationDisabledUntil.Unix(), 10) + ":R>"
		actions, err := env.Store.RecentStaffActionsForUser(ctx, interaction.GuildID, targetID, 10)
		if err != nil {
			return fmt.Errorf("consultar motivo del silencio seleccionado: %w", err)
		}
		for _, action := range actions {
			if action.Action == "timeout" {
				description += "\n**Motivo registrado:** " + safeText(action.Details)
				break
			}
		}
	default:
		return respond(session, interaction, "Tipo de sanción desconocido.", true)
	}
	if !hasStaffPermission(interaction.Member.Permissions, required) {
		return respond(session, interaction, "Tu permiso actual no permite revocar esta sanción.", true)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🧾・REVISIÓN DE SANCIÓN",
		Description: fmt.Sprintf("**Usuario:** <@%s>\n%s\n\nConfirma solo si corresponde retirar el castigo.", targetID, description),
		Color:       env.Theme.ColorWarning,
	}
	env.Theme.BrandEmbed(embed)
	buttons := []discordgo.MessageComponent{
		discordgo.Button{Label: "Revocar sanción", Emoji: &discordgo.ComponentEmoji{Name: "🕊️"}, Style: discordgo.DangerButton, CustomID: fmt.Sprintf("security:confirm:%s:%s:%s", interaction.GuildID, kind, targetID)},
	}
	if kind == "timeout" {
		buttons = append(buttons, discordgo.Button{Label: "Extender", Emoji: &discordgo.ComponentEmoji{Name: "⏳"}, Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("security:extend:%s:%s", interaction.GuildID, targetID)})
	}
	buttons = append(buttons, discordgo.Button{Label: "Cancelar", Style: discordgo.SecondaryButton, CustomID: "security:home:" + interaction.GuildID})
	return updateSecurityMessage(session, interaction, embed, []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}})
}

func handleTimeoutExtension(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.Member.User == nil || !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
		return respond(session, interaction, "Extender un silencio requiere Moderar miembros.", true)
	}
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || parts[0] != "security_extend" || parts[1] != interaction.GuildID {
		return respond(session, interaction, "Este formulario de extensión ya no es válido.", true)
	}
	rawMinutes := ""
	for _, row := range interaction.ModalSubmitData().Components {
		if actionRow, ok := row.(*discordgo.ActionsRow); ok {
			for _, component := range actionRow.Components {
				if input, ok := component.(*discordgo.TextInput); ok && input.CustomID == "minutes" {
					rawMinutes = strings.TrimSpace(input.Value)
				}
			}
		}
	}
	minutes, err := strconv.Atoi(rawMinutes)
	if err != nil || minutes < 1 || minutes > 40320 {
		return respond(session, interaction, "Indica entre **1 y 40 320 minutos** (máximo 28 días).", true)
	}
	member, err := session.GuildMember(interaction.GuildID, parts[2])
	if err != nil {
		return fmt.Errorf("consultar miembro para extender silencio: %w", err)
	}
	if member.CommunicationDisabledUntil == nil || !member.CommunicationDisabledUntil.After(time.Now()) {
		return respond(session, interaction, "Ese silencio ya expiró y no se puede extender.", true)
	}
	until := member.CommunicationDisabledUntil.Add(time.Duration(minutes) * time.Minute)
	if until.After(time.Now().Add(28 * 24 * time.Hour)) {
		return respond(session, interaction, "La extensión superaría el máximo permitido por Discord de 28 días.", true)
	}
	if err := session.GuildMemberTimeout(interaction.GuildID, parts[2], &until); err != nil {
		return fmt.Errorf("extender silencio del miembro: %w", err)
	}
	if err := env.Store.RecordStaffAction(ctx, interaction.GuildID, interaction.Member.User.ID, parts[2], "timeout_extend", fmt.Sprintf("%d minutos adicionales", minutes)); err != nil {
		return fmt.Errorf("registrar extensión de silencio: %w", err)
	}
	return respond(session, interaction, fmt.Sprintf("⏳ El silencio de <@%s> se extendió hasta <t:%d:F>.", parts[2], until.Unix()), true)
}

func handleSecurityRevoke(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, parts []string) error {
	if len(parts) != 5 || interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "La confirmación ya no es válida.", true)
	}
	guildID, kind, targetID := parts[2], parts[3], parts[4]
	if guildID != interaction.GuildID {
		return respond(session, interaction, "La sanción no pertenece a este servidor.", true)
	}
	var err error
	var action string
	switch kind {
	case "ban":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionBanMembers) {
			return respond(session, interaction, "Revocar este baneo requiere Banear miembros.", true)
		}
		err = session.GuildBanDelete(guildID, targetID)
		action = "unban"
	case "timeout":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
			return respond(session, interaction, "Revocar este silencio requiere Moderar miembros.", true)
		}
		err = session.GuildMemberTimeout(guildID, targetID, nil)
		action = "untimeout"
	default:
		return respond(session, interaction, "Tipo de sanción desconocido.", true)
	}
	if err != nil {
		return fmt.Errorf("revocar sanción %s de %s: %w", kind, targetID, err)
	}
	if err := env.Store.RecordStaffAction(ctx, guildID, interaction.Member.User.ID, targetID, action, "Revocada desde puertas-infernales"); err != nil {
		return fmt.Errorf("registrar revocación en auditoría: %w", err)
	}
	embed := &discordgo.MessageEmbed{Title: "🕊️・SANCIÓN REVOCADA", Description: fmt.Sprintf("La sanción de <@%s> fue retirada por <@%s>.\n\nLa acción quedó registrada en el expediente del servidor.", targetID, interaction.Member.User.ID), Color: env.Theme.ColorSuccess}
	env.Theme.BrandEmbed(embed)
	return updateSecurityMessage(session, interaction, embed, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Volver al panel", Style: discordgo.SecondaryButton, CustomID: "security:home:" + guildID},
		}},
	})
}

func handleAppealReview(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, parts []string) error {
	if len(parts) != 5 || interaction.Member == nil || interaction.Member.User == nil ||
		(!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) &&
			!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild)) {
		return respond(session, interaction, "Esta revisión está reservada al staff.", true)
	}
	guildID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || parts[2] != interaction.GuildID {
		return respond(session, interaction, "La apelación seleccionada no es válida.", true)
	}
	status := "denied"
	if parts[4] == "accept" {
		status = "accepted"
	} else if parts[4] != "deny" {
		return respond(session, interaction, "Resolución de apelación desconocida.", true)
	}
	appeals, err := env.Store.ListSanctionAppeals(ctx, interaction.GuildID, "", 25)
	if err != nil {
		return fmt.Errorf("consultar apelación seleccionada: %w", err)
	}
	var selected *storage.SanctionAppeal
	for idx := range appeals {
		if appeals[idx].ID == guildID {
			selected = &appeals[idx]
			break
		}
	}
	if selected == nil || selected.Status != "pending" {
		return respond(session, interaction, "La apelación ya fue revisada o no está disponible.", true)
	}
	if status == "accepted" {
		switch selected.Sanction {
		case "ban":
			if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionBanMembers) {
				return respond(session, interaction, "Aceptar esta apelación y retirar el baneo requiere Banear miembros.", true)
			}
			if err := session.GuildBanDelete(interaction.GuildID, selected.TargetID); err != nil {
				return fmt.Errorf("retirar baneo apelado: %w", err)
			}
		case "timeout":
			if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
				return respond(session, interaction, "Aceptar esta apelación y retirar el silencio requiere Moderar miembros.", true)
			}
			if err := session.GuildMemberTimeout(interaction.GuildID, selected.TargetID, nil); err != nil {
				return fmt.Errorf("retirar silencio apelado: %w", err)
			}
		}
	}
	if err := env.Store.ResolveSanctionAppeal(ctx, interaction.GuildID, selected.ID, interaction.Member.User.ID, status); err != nil {
		return fmt.Errorf("guardar resolución de apelación: %w", err)
	}
	if err := env.Store.RecordStaffAction(
		ctx, interaction.GuildID, interaction.Member.User.ID, selected.TargetID,
		"appeal_"+status, fmt.Sprintf("Apelación #%d sobre %s", selected.ID, selected.Sanction),
	); err != nil {
		return fmt.Errorf("registrar revisión de apelación en auditoría: %w", err)
	}
	return showSecuritySection(ctx, session, interaction, env, "appeals")
}

func handleSecuritySearch(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.GuildID == "" || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "El historial está reservado al staff.", true)
	}
	data := interaction.ModalSubmitData()
	if !strings.HasPrefix(data.CustomID, "security_search:") || strings.TrimPrefix(data.CustomID, "security_search:") != interaction.GuildID {
		return respond(session, interaction, "Este formulario ya no corresponde a este servidor.", true)
	}
	userID := ""
	for _, row := range data.Components {
		actionsRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, component := range actionsRow.Components {
			if input, ok := component.(*discordgo.TextInput); ok && input.CustomID == "user_id" {
				userID = strings.TrimSpace(input.Value)
			}
		}
	}
	if len(userID) < 17 || len(userID) > 20 {
		return respond(session, interaction, "El ID debe ser un identificador de Discord válido.", true)
	}
	if _, err := strconv.ParseUint(userID, 10, 64); err != nil {
		return respond(session, interaction, "El ID debe contener solo dígitos.", true)
	}
	actions, err := env.Store.RecentStaffActionsForUser(ctx, interaction.GuildID, userID, 10)
	if err != nil {
		return fmt.Errorf("buscar historial de moderación: %w", err)
	}
	appeals, err := env.Store.ListSanctionAppeals(ctx, interaction.GuildID, userID, 5)
	if err != nil {
		return fmt.Errorf("buscar apelaciones del usuario: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "📜・EXPEDIENTE DE SEGURIDAD",
		Description: fmt.Sprintf("**Miembro auditado:** <@%s>\n**Nivel de acceso:** búsqueda interna del servidor\n**Registros encontrados:** %d acciones · %d apelación(es)", userID, len(actions), len(appeals)),
		Color:       env.Theme.Primary,
	}
	if len(actions) == 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Historial", Value: "No hay acciones de staff registradas para este miembro."})
	} else {
		lines := make([]string, 0, len(actions))
		for _, action := range actions {
			lines = append(lines, fmt.Sprintf("• **%s** · <@%s> · %s · <t:%d:R>", action.Action, action.ActorID, truncate(action.Details, 100), action.CreatedAt.Unix()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Acciones recientes", Value: strings.Join(lines, "\n")})
	}
	if len(appeals) > 0 {
		lines := make([]string, 0, len(appeals))
		for _, appeal := range appeals {
			lines = append(lines, fmt.Sprintf("• **#%d · %s · %s** · %s", appeal.ID, appeal.Sanction, appeal.Status, truncate(appeal.Reason, 120)))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Apelaciones", Value: strings.Join(lines, "\n")})
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "El historial refleja únicamente acciones registradas por este bot."}
	env.Theme.BrandEmbed(embed)
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func updateSecurityPanel(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, notice string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🔐・PUERTAS INFERNALES",
		Description: notice,
		Color:       env.Theme.Primary,
	}
	env.Theme.BrandEmbed(embed)
	return updateSecurityMessage(session, interaction, embed, securityPanelComponents(interaction.GuildID))
}

func updateSecurityMessage(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}

func respondSecurityPanel(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	_, err := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	return err
}
