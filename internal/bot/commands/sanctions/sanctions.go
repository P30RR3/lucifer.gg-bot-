package sanctions

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	if err := registry.Register(&bot.Command{
		Name:        "panel-staff",
		Description: "🕵️ Abre el archivo del club: evidencias, historial y acciones autorizadas.",
		Category:    "moderacion",
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "usuario", Description: "Miembro que se analizará", Required: true},
		},
		Handler: c.handleInvestigation,
	}); err != nil {
		return err
	}
	if err := registry.Register(&bot.Command{
		Name:        "inframundo",
		Description: "Abre el panel del staff para revisar expedientes infernales.",
		Category:    "moderacion",
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "menu",
				Description: "Consulta tiempos, notas y controles de absolución.",
			},
		},
		Handler: c.handleInfernal,
	}); err != nil {
		return err
	}
	if err := registry.Register(&bot.Command{
		Name:        "infierno",
		Description: "Envía a un miembro al bucle de redención durante un tiempo definido.",
		Category:    "moderacion",
		DefaultPerm: discordgo.PermissionModerateMembers,
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "usuario", Description: "Miembro que será aislado", Required: true},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "minutos", Description: "Duración (1-1440 minutos)", Required: true, MinValue: floatPtr(1), MaxValue: 1440},
			{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo de moderación", Required: true, MaxLength: 300},
		},
		Handler: c.handleInfierno,
	}); err != nil {
		return err
	}
	return registry.Register(securityCenterCommand())
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("investigate", handleInvestigationComponent)
	router.RegisterModal("investigation_evidence", handleEvidenceModal)
	router.RegisterComponent("investigation_interrogate", handleInterrogationInvite)
	router.RegisterModal("investigation_answer", handleInterrogationAnswer)
	router.RegisterComponent("infernal", handleInfernalComponent)
	router.RegisterModal("infernal_answer", handleInfernalAnswer)
	router.RegisterModal("infernal_note", handleInfernalNote)
}

func (c *Cog) handleInvestigation(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "🛡️ Este archivo está reservado al staff con permisos de moderación de Discord.", true)
	}
	target := userOption(interaction.ApplicationCommandData().Options, "usuario", session)
	if target == nil {
		return respond(session, interaction, "🔎 No encontré a esa persona en el servidor.", true)
	}
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return fmt.Errorf("iniciar expediente privado: %w", err)
	}
	member, err := session.GuildMember(interaction.GuildID, target.ID)
	if err != nil {
		return editInvestigationError(session, interaction, env, "No pude abrir el expediente: el miembro ya no está en el servidor.", fmt.Errorf("consultar miembro de investigación: %w", err))
	}
	incidents, err := env.Store.ListIncidentsByUser(ctx, interaction.GuildID, target.ID, 5)
	if err != nil {
		return editInvestigationError(session, interaction, env, "No pude consultar los logs del expediente. Inténtalo de nuevo más tarde.", fmt.Errorf("consultar logs de investigación: %w", err))
	}
	evidence, err := env.Store.ListInvestigationEvidence(ctx, interaction.GuildID, target.ID, 5)
	if err != nil {
		return editInvestigationError(session, interaction, env, "No pude consultar las evidencias del expediente. Inténtalo de nuevo más tarde.", fmt.Errorf("consultar evidencias de investigación: %w", err))
	}
	responses, err := env.Store.ListInvestigationResponses(ctx, interaction.GuildID, target.ID, 3)
	if err != nil {
		return editInvestigationError(session, interaction, env, "No pude consultar las respuestas del expediente. Inténtalo de nuevo más tarde.", fmt.Errorf("consultar respuestas de investigación: %w", err))
	}
	appeals, err := env.Store.ListSanctionAppeals(ctx, interaction.GuildID, target.ID, 3)
	if err != nil {
		return editInvestigationError(session, interaction, env, "No pude consultar las apelaciones del expediente. Inténtalo de nuevo más tarde.", fmt.Errorf("consultar apelaciones de investigación: %w", err))
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🕵️・ARCHIVO PRIVADO DEL LUX・" + target.Username,
		Description: fmt.Sprintf("╭──────────── ✦ ────────────╮\n**Miembro:** <@%s>\n**Cuenta creada:** <t:%d:D>\n**Entre nosotros desde:** <t:%d:D>\n\nRevisa los registros con criterio; cada acción exige el permiso correspondiente de Discord.\n╰──────────── ✦ ────────────╯", target.ID, userCreationTime(target.ID).Unix(), member.JoinedAt.Unix()),
		Color:       env.Theme.Primary,
	}
	env.Theme.BrandEmbed(embed)
	if len(incidents) == 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "📜 Logs recientes", Value: "No hay incidentes registrados."})
	} else {
		var lines []string
		for _, incident := range incidents {
			lines = append(lines, fmt.Sprintf("• %s · %s · %s · <t:%d:R>", incident.Detector, incident.Action, truncate(incident.Reason, 100), incident.CreatedAt.Unix()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "📜 Logs recientes", Value: strings.Join(lines, "\n")})
	}
	embed.Fields = append(embed.Fields,
		&discordgo.MessageEmbedField{Name: "🧾 Evidencias", Value: fmt.Sprintf("%d registro(s). Usa el botón para consultar o añadir.", len(evidence)), Inline: true},
		&discordgo.MessageEmbedField{Name: "💬 Interrogatorio", Value: fmt.Sprintf("%d respuesta(s) recibida(s).", len(responses)), Inline: true},
	)
	if len(responses) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Última respuesta", Value: truncate(responses[0], 500)})
	}
	components := investigationButtons(interaction.GuildID, target.ID, interaction.Member.Permissions)
	if len(appeals) > 0 {
		pending := 0
		for _, appeal := range appeals {
			if appeal.Status == "pending" {
				pending++
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚖️ Apelaciones",
			Value:  fmt.Sprintf("%d registro(s) · %d pendiente(s)", len(appeals), pending),
			Inline: true,
		})
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Abrir apelaciones", Emoji: &discordgo.ComponentEmoji{Name: "⚖️"}, Style: discordgo.SecondaryButton, CustomID: "security:view:" + interaction.GuildID + ":appeals"},
		}})
	}
	_, err = discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		return fmt.Errorf("mostrar expediente privado: %w", err)
	}
	return nil
}

func editInvestigationError(session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, message string, cause error) error {
	logger := env.Log
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("falló la carga de un expediente de moderación", "guild_id", interaction.GuildID, "err", cause)
	_, editErr := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{Content: &message})
	if editErr != nil {
		return errors.Join(cause, fmt.Errorf("mostrar error del expediente: %w", editErr))
	}
	return nil
}

func handleInvestigationComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 4 || interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "Esta acción de investigación ya no es válida.", true)
	}
	_, guildID, targetID, action := parts[0], parts[1], parts[2], parts[3]
	if guildID != interaction.GuildID || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "Esta acción requiere permisos de moderación en el servidor.", true)
	}
	if action == "evidence" || action == "add_evidence" || action == "interrogate" || action == "responses" {
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) &&
			!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild) {
			return respond(session, interaction, "Esta consulta requiere Gestionar mensajes o Gestionar servidor.", true)
		}
	}
	switch action {
	case "evidence":
		evidence, err := env.Store.ListInvestigationEvidence(ctx, guildID, targetID, 10)
		if err != nil {
			return fmt.Errorf("listar evidencias desde panel: %w", err)
		}
		if len(evidence) == 0 {
			return respond(session, interaction, "No hay evidencias guardadas para este miembro.", true)
		}
		for idx := range evidence {
			evidence[idx] = fmt.Sprintf("**%d.** %s", idx+1, truncate(evidence[idx], 400))
		}
		return respond(session, interaction, strings.Join(evidence, "\n"), true)
	case "add_evidence":
		return openEvidenceModal(session, interaction, guildID, targetID)
	case "interrogate":
		target, err := session.User(targetID)
		if err != nil {
			return fmt.Errorf("consultar usuario para interrogatorio: %w", err)
		}
		dm, err := session.UserChannelCreate(targetID)
		if err != nil {
			return fmt.Errorf("abrir DM de interrogatorio: %w", err)
		}
		_, err = discordutil.ChannelMessageSendComplex(session, dm.ID, &discordgo.MessageSend{
			Content: "🛡️ El staff de un servidor te solicita una explicación. Puedes responder voluntariamente mediante este formulario.",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{CustomID: fmt.Sprintf("investigation_interrogate:%s:%s", guildID, targetID), Label: "Responder al staff", Style: discordgo.PrimaryButton},
				}},
			},
		})
		if err != nil {
			return fmt.Errorf("enviar solicitud de interrogatorio: %w", err)
		}
		return respond(session, interaction, fmt.Sprintf("📨 Envié una solicitud voluntaria de respuesta a **%s**.", target.Username), true)
	case "responses":
		responses, err := env.Store.ListInvestigationResponses(ctx, guildID, targetID, 10)
		if err != nil {
			return fmt.Errorf("listar respuestas desde panel: %w", err)
		}
		if len(responses) == 0 {
			return respond(session, interaction, "Todavía no hay respuestas de interrogatorio.", true)
		}
		for idx := range responses {
			responses[idx] = fmt.Sprintf("**%d.** %s", idx+1, truncate(responses[idx], 400))
		}
		return respond(session, interaction, strings.Join(responses, "\n"), true)
	case "timeout", "kick", "ban":
		return executeInvestigationAction(ctx, session, interaction, env, guildID, targetID, action)
	default:
		return respond(session, interaction, "Esa acción de investigación no existe.", true)
	}
}

func executeInvestigationAction(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, guildID, targetID, action string) error {
	permission := int64(discordgo.PermissionModerateMembers)
	switch action {
	case "kick":
		permission = discordgo.PermissionKickMembers
	case "ban":
		permission = discordgo.PermissionBanMembers
	}
	perms, err := session.UserChannelPermissions(interaction.Member.User.ID, interaction.ChannelID)
	if err != nil {
		return fmt.Errorf("comprobar permiso de acción de investigación: %w", err)
	}
	if !hasStaffPermission(perms, permission) {
		return respond(session, interaction, "No tienes el permiso específico de Discord para esa acción.", true)
	}
	reason := "Acción manual desde /panel-staff"
	var actionErr error
	switch action {
	case "timeout":
		until := time.Now().Add(10 * time.Minute)
		actionErr = session.GuildMemberTimeout(guildID, targetID, &until)
	case "kick":
		actionErr = session.GuildMemberDeleteWithReason(guildID, targetID, reason)
	case "ban":
		actionErr = session.GuildBanCreateWithReason(guildID, targetID, reason, 0)
	}
	if actionErr != nil {
		return fmt.Errorf("ejecutar %s desde investigación: %w", action, actionErr)
	}
	if err := env.Store.RecordStaffAction(ctx, guildID, interaction.Member.User.ID, targetID, action, reason); err != nil {
		return fmt.Errorf("registrar acción de investigación: %w", err)
	}
	return respond(session, interaction, "✅ Acción de moderación aplicada y registrada.", true)
}

func openEvidenceModal(session *discordgo.Session, interaction *discordgo.InteractionCreate, guildID, targetID string) error {
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("investigation_evidence:%s:%s", guildID, targetID),
			Title:    "Añadir evidencia",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "evidence", Label: "Describe la evidencia verificable", Style: discordgo.TextInputParagraph, Required: true, MinLength: 5, MaxLength: 1000},
				}},
			},
		},
	})
}

func handleEvidenceModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "El formulario de evidencia ya no es válido.", true)
	}
	guildID, targetID := parts[1], parts[2]
	if interaction.GuildID != guildID ||
		(!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) &&
			!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild)) {
		return respond(session, interaction, "Solo el staff puede añadir evidencia al expediente.", true)
	}
	evidence := strings.TrimSpace(modalText(interaction.ModalSubmitData().Components, "evidence"))
	if len([]rune(evidence)) < 5 {
		return respond(session, interaction, "La evidencia debe incluir detalles verificables.", true)
	}
	if err := env.Store.AddInvestigationEvidence(ctx, guildID, targetID, interaction.Member.User.ID, evidence); err != nil {
		return fmt.Errorf("guardar evidencia desde modal: %w", err)
	}
	return respond(session, interaction, "✅ Evidencia guardada en el expediente.", true)
}

func handleInterrogationInvite(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) != 3 || interaction.User == nil || interaction.User.ID != parts[2] {
		return respond(session, interaction, "Esta solicitud de respuesta no está dirigida a ti.", true)
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("investigation_answer:%s:%s", parts[1], parts[2]),
			Title:    "Respuesta voluntaria al staff",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "answer", Label: "Tu explicación", Style: discordgo.TextInputParagraph, Required: true, MinLength: 1, MaxLength: 1000},
				}},
			},
		},
	})
}

func handleInterrogationAnswer(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || interaction.User == nil || interaction.User.ID != parts[2] {
		return respond(session, interaction, "Esta respuesta no corresponde a tu solicitud.", true)
	}
	answer := strings.TrimSpace(modalText(interaction.ModalSubmitData().Components, "answer"))
	if answer == "" {
		return respond(session, interaction, "La respuesta no puede estar vacía.", true)
	}
	if err := env.Store.SaveInvestigationResponse(ctx, parts[1], parts[2], answer); err != nil {
		return fmt.Errorf("guardar respuesta voluntaria: %w", err)
	}
	return respond(session, interaction, "✅ Tu respuesta fue enviada al expediente de moderación.", true)
}

func (c *Cog) handleInfernal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if len(interaction.ApplicationCommandData().Options) == 0 ||
		interaction.ApplicationCommandData().Options[0].Name != "menu" {
		return respond(session, interaction, "🔥 Usa `/inframundo menu` para revisar los expedientes activos.", true)
	}
	if interaction.Member == nil || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "El panel requiere permisos de moderación de Discord.", true)
	}
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return fmt.Errorf("iniciar panel del inframundo: %w", err)
	}
	return showInfernalPanel(ctx, session, interaction, env, "", "")
}

func (c *Cog) handleInfierno(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	options := interaction.ApplicationCommandData().Options
	target := userOption(options, "usuario", session)
	minutes := optionInt(options, "minutos")
	reason := strings.TrimSpace(optionString(options, "motivo"))
	if target == nil {
		return respond(session, interaction, "No encontré ese usuario.", true)
	}
	if target.Bot || target.ID == interaction.Member.User.ID || minutes < 1 || minutes > 1440 || reason == "" {
		return respond(session, interaction, "Elige a otro miembro real, una duración de 1 a 1.440 minutos y un motivo.", true)
	}
	return c.isolate(ctx, session, interaction, env, target, time.Duration(minutes)*time.Minute, reason)
}

func showInfernalPanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, selectedTarget, notice string) error {
	cases, err := env.Store.ListActiveInfernalCases(ctx, interaction.GuildID, 25)
	if err != nil {
		if interaction.Type == discordgo.InteractionApplicationCommand {
			message := "No pude cargar los expedientes del inframundo. Inténtalo de nuevo más tarde."
			_, editErr := discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{Content: &message})
			if editErr != nil {
				return errors.Join(fmt.Errorf("cargar panel del inframundo: %w", err), fmt.Errorf("mostrar error del panel: %w", editErr))
			}
			logger := env.Log
			if logger == nil {
				logger = slog.Default()
			}
			logger.Error("falló la carga del panel del inframundo", "guild_id", interaction.GuildID, "err", err)
			return nil
		}
		return fmt.Errorf("cargar panel del inframundo: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🔒・CENTRO DE CONTROL DEL INFRAMUNDO",
		Description: "Expedientes activos, tiempos restantes y controles de supervisión para el staff.",
		Color:       env.Theme.ColorDanger,
	}
	if notice != "" {
		embed.Description = notice + "\n\n" + embed.Description
	}
	options := make([]discordgo.SelectMenuOption, 0, len(cases))
	var selected *storage.InfernalCase
	for idx := range cases {
		caseFile := &cases[idx]
		options = append(options, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("Expediente %02d · …%s", idx+1, lastDigits(caseFile.TargetID, 6)),
			Value:       caseFile.TargetID,
			Description: truncate(fmt.Sprintf("Termina %s · %s", caseFile.ExpiresAt.Format("02 Jan · 15:04 MST"), caseFile.Reason), 100),
			Default:     caseFile.TargetID == selectedTarget,
		})
		if selectedTarget == "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("🔒 Expediente %02d · <@%s>", idx+1, caseFile.TargetID),
				Value:  fmt.Sprintf("Termina <t:%d:R> · %s", caseFile.ExpiresAt.Unix(), truncate(safeText(caseFile.Reason), 140)),
				Inline: false,
			})
		}
		if caseFile.TargetID == selectedTarget {
			selected = caseFile
		}
	}
	if len(cases) == 0 {
		embed.Description = "No hay aislamientos activos en este servidor."
		if notice != "" {
			embed.Description = notice + "\n\n" + embed.Description
		}
	} else if selected != nil {
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "👤 Miembro", Value: fmt.Sprintf("<@%s>", selected.TargetID), Inline: true},
			{Name: "⏳ Tiempo restante", Value: fmt.Sprintf("<t:%d:R>", selected.ExpiresAt.Unix()), Inline: true},
			{Name: "🧾 Motivo", Value: truncate(safeText(selected.Reason), 500)},
			{Name: "📝 Nota del expediente", Value: truncate(emptyNote(selected.Note), 500)},
			{Name: "🧵 Bucle de redención", Value: fmt.Sprintf("<#%s>", selected.ThreadID), Inline: true},
		}
	} else {
		embed.Description += "\n\nSelecciona un expediente para revisar sus detalles y acciones."
	}
	ownerID := interaction.Member.User.ID
	components := []discordgo.MessageComponent{}
	if len(options) > 0 {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.StringSelectMenu,
				CustomID:    fmt.Sprintf("infernal:panel:%s:select", ownerID),
				Placeholder: "✦ Selecciona un expediente activo ✦",
				Options:     options,
				MinValues:   intPtr(1),
				MaxValues:   1,
			},
		}})
		if selected != nil {
			buttons := make([]discordgo.MessageComponent, 0, 2)
			if hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) ||
				hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild) {
				buttons = append(buttons, discordgo.Button{
					CustomID: fmt.Sprintf("infernal:panel:%s:note:%s", ownerID, selected.TargetID),
					Label:    "📝 Añadir nota",
					Style:    discordgo.SecondaryButton,
				})
			}
			if hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
				buttons = append(buttons, discordgo.Button{
					CustomID: fmt.Sprintf("infernal:panel:%s:absolve:%s", ownerID, selected.TargetID),
					Label:    "🕊️ Absolver",
					Style:    discordgo.SuccessButton,
				})
			}
			if len(buttons) > 0 {
				components = append(components, discordgo.ActionsRow{Components: buttons})
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
	_, err = discordutil.InteractionResponseEdit(session, interaction.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		return fmt.Errorf("mostrar panel del inframundo: %w", err)
	}
	return nil
}

func (c *Cog) isolate(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, target *discordgo.User, duration time.Duration, reason string) error {
	perms, err := session.UserChannelPermissions(session.State.User.ID, interaction.ChannelID)
	if err != nil {
		return fmt.Errorf("comprobar permisos de hilo infernal: %w", err)
	}
	required := int64(discordgo.PermissionCreatePrivateThreads | discordgo.PermissionManageThreads | discordgo.PermissionSendMessagesInThreads)
	if perms&required != required {
		return respond(session, interaction, "Necesito permisos para crear y gestionar hilos privados en este canal.", true)
	}
	thread, err := session.ThreadStartComplex(interaction.ChannelID, &discordgo.ThreadStart{
		Name:                threadName("🔒・Bucle de redención・" + target.Username),
		AutoArchiveDuration: 1440,
		Type:                discordgo.ChannelTypeGuildPrivateThread,
		Invitable:           false,
	})
	if err != nil {
		return fmt.Errorf("crear hilo de aislamiento: %w", err)
	}
	if err := session.ThreadMemberAdd(thread.ID, target.ID); err != nil {
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("invitar usuario al hilo de aislamiento: %w", err), deleteErr)
	}
	first, err := rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("crear tarea de redención: %w", err), deleteErr)
	}
	second, err := rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("crear tarea de redención: %w", err), deleteErr)
	}
	answer := first.Int64() + second.Int64()
	challenge := fmt.Sprintf("Resuelve la operación de redención: %d + %d = ?", first.Int64(), second.Int64())
	until := time.Now().Add(duration)
	record := storage.InfernalCase{
		GuildID: interaction.GuildID, TargetID: target.ID, ThreadID: thread.ID,
		ActorID: interaction.Member.User.ID, Reason: reason,
		Challenge: challenge, Answer: strconv.FormatInt(answer, 10), ExpiresAt: until,
	}
	if err := env.Store.StartInfernalCase(ctx, record); err != nil {
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("guardar expediente infernal: %w", err), deleteErr)
	}
	if err := session.GuildMemberTimeout(interaction.GuildID, target.ID, &until); err != nil {
		_ = env.Store.ResolveInfernalCase(ctx, interaction.GuildID, target.ID, "rollback")
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("aplicar timeout de aislamiento: %w", err), deleteErr)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🔒・BUCLE INFERNAL・TAREA DE REDENCIÓN",
		Description: fmt.Sprintf("<@%s>, tu aislamiento termina <t:%d:R>.\n\n**📜 Motivo:** %s\n\n**🧩 Para recuperar el acceso antes:**\n%s", target.ID, until.Unix(), reason, challenge),
		Color:       env.Theme.ColorWarning,
		Footer:      &discordgo.MessageEmbedFooter{Text: "También puedes esperar hasta que el timeout termine. El staff puede revisar el tiempo, añadir notas o absolverte."},
	}
	env.Theme.BrandEmbed(embed)
	_, err = discordutil.ChannelMessageSendComplex(session, thread.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{CustomID: fmt.Sprintf("infernal:task:%s:%s", interaction.GuildID, target.ID), Label: "🧩 Completar tarea", Style: discordgo.PrimaryButton},
			}},
		},
		AllowedMentions: &discordgo.MessageAllowedMentions{Users: []string{target.ID}},
	})
	if err != nil {
		_ = session.GuildMemberTimeout(interaction.GuildID, target.ID, nil)
		_ = env.Store.ResolveInfernalCase(ctx, interaction.GuildID, target.ID, "rollback")
		_, deleteErr := session.ChannelDelete(thread.ID)
		return errors.Join(fmt.Errorf("publicar panel de redención: %w", err), deleteErr)
	}
	dm, dmErr := session.UserChannelCreate(target.ID)
	dmSent := false
	if dmErr == nil {
		_, dmErr = discordutil.ChannelMessageSendComplex(session, dm.ID, &discordgo.MessageSend{
			Content: fmt.Sprintf("🔒 Tienes una tarea de redención pendiente en **%s**.\n%s", interaction.GuildID, challenge),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{CustomID: fmt.Sprintf("infernal:task:%s:%s", interaction.GuildID, target.ID), Label: "🧩 Completar tarea", Style: discordgo.PrimaryButton},
				}},
			},
		})
		dmSent = dmErr == nil
	}
	if dmErr != nil {
		env.Log.Warn("no se pudo enviar la tarea de redención por DM", "guild_id", interaction.GuildID, "user_id", target.ID, "err", dmErr)
	}
	delivery := "El usuario puede esperar al fin del timeout."
	if dmSent {
		delivery = "También le envié por DM una opción para completar la tarea antes."
	}
	return respond(session, interaction, fmt.Sprintf("🔒 <@%s> fue aislado por %d minutos en <#%s>. %s", target.ID, int(duration.Minutes()), thread.ID, delivery), true)
}

func handleInfernalComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) == 4 && parts[1] == "task" {
		return handleInfernalTaskComponent(ctx, session, interaction, env, parts)
	}
	if len(parts) < 4 || parts[1] != "panel" || interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "Este control del inframundo ya no es válido.", true)
	}
	if parts[2] != interaction.Member.User.ID || !hasAnyStaffPermission(interaction.Member.Permissions) {
		return respond(session, interaction, "Este panel es personal y requiere permisos de moderación.", true)
	}
	action := parts[3]
	if action == "select" {
		values := interaction.MessageComponentData().Values
		if len(values) != 1 {
			return respond(session, interaction, "Selecciona un expediente.", true)
		}
		return showInfernalPanel(ctx, session, interaction, env, values[0], "")
	}
	if len(parts) != 5 {
		return respond(session, interaction, "Este control del inframundo ya no es válido.", true)
	}
	targetID := parts[4]
	switch action {
	case "note":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) &&
			!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild) {
			return respond(session, interaction, "Añadir notas requiere Gestionar mensajes o Gestionar servidor.", true)
		}
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("infernal_note:%s:%s:%s", interaction.GuildID, targetID, interaction.Member.User.ID),
				Title:    "📝 Nota del expediente",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "note", Label: "Nota para el staff", Style: discordgo.TextInputParagraph, Required: true, MaxLength: 500},
					}},
				},
			},
		})
	case "absolve":
		if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionModerateMembers) {
			return respond(session, interaction, "Absolver requiere el permiso Moderar miembros.", true)
		}
		record, err := env.Store.GetInfernalCase(ctx, interaction.GuildID, targetID)
		if err != nil {
			return respond(session, interaction, "No hay un expediente activo para esa persona.", true)
		}
		if !record.Active || !time.Now().Before(record.ExpiresAt) {
			return respond(session, interaction, "Ese expediente ya terminó.", true)
		}
		if err := session.GuildMemberTimeout(interaction.GuildID, targetID, nil); err != nil {
			return fmt.Errorf("retirar timeout desde panel infernal: %w", err)
		}
		if err := env.Store.ResolveInfernalCase(ctx, interaction.GuildID, targetID, interaction.Member.User.ID); err != nil {
			return fmt.Errorf("cerrar expediente desde panel infernal: %w", err)
		}
		archived := true
		if record.ThreadID != "" {
			if _, err := session.ChannelEditComplex(record.ThreadID, &discordgo.ChannelEdit{Archived: &archived}); err != nil {
				env.Log.Warn("no se pudo archivar el hilo de redención", "thread_id", record.ThreadID, "err", err)
			}
		}
		return showInfernalPanel(ctx, session, interaction, env, "", fmt.Sprintf("🕊️ <@%s> fue absuelto y su timeout se retiró.", targetID))
	default:
		return respond(session, interaction, "Acción de expediente desconocida.", true)
	}
}

func handleInfernalTaskComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, parts []string) error {
	userID := ""
	if interaction.User != nil {
		userID = interaction.User.ID
	} else if interaction.Member != nil && interaction.Member.User != nil {
		userID = interaction.Member.User.ID
	}
	if userID != parts[3] || (interaction.GuildID != "" && interaction.GuildID != parts[2]) {
		return respond(session, interaction, "Esta tarea de redención no está dirigida a ti.", true)
	}
	record, err := env.Store.GetInfernalCase(ctx, parts[2], parts[3])
	if err != nil {
		return respond(session, interaction, "No tienes una tarea de redención activa.", true)
	}
	if !record.Active || !time.Now().Before(record.ExpiresAt) ||
		(interaction.GuildID != "" && record.ThreadID != interaction.ChannelID) {
		return respond(session, interaction, "No tienes una tarea de redención activa.", true)
	}
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("infernal_answer:%s:%s", parts[2], parts[3]),
			Title:    "🧩 Tarea de redención",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "answer", Label: "Escribe el resultado", Style: discordgo.TextInputShort, Required: true, MaxLength: 20},
				}},
			},
		},
	})
}

func handleInfernalNote(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 4 || interaction.Member == nil || interaction.Member.User == nil ||
		interaction.GuildID != parts[1] || interaction.Member.User.ID != parts[3] {
		return respond(session, interaction, "Este formulario de expediente ya no es válido.", true)
	}
	if !hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageMessages) &&
		!hasStaffPermission(interaction.Member.Permissions, discordgo.PermissionManageGuild) {
		return respond(session, interaction, "Añadir notas requiere Gestionar mensajes o Gestionar servidor.", true)
	}
	note := strings.TrimSpace(modalText(interaction.ModalSubmitData().Components, "note"))
	if note == "" {
		return respond(session, interaction, "La nota no puede estar vacía.", true)
	}
	if err := env.Store.UpdateInfernalNote(ctx, parts[1], parts[2], note); err != nil {
		return fmt.Errorf("guardar nota desde panel infernal: %w", err)
	}
	return respond(session, interaction, "📝 Nota añadida al expediente.", true)
}

func handleInfernalAnswer(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 {
		return respond(session, interaction, "Esta respuesta no corresponde a tu tarea.", true)
	}
	userID := ""
	if interaction.User != nil {
		userID = interaction.User.ID
	} else if interaction.Member != nil && interaction.Member.User != nil {
		userID = interaction.Member.User.ID
	}
	if userID != parts[2] || (interaction.GuildID != "" && interaction.GuildID != parts[1]) {
		return respond(session, interaction, "Esta respuesta no corresponde a tu tarea.", true)
	}
	answer := strings.TrimSpace(modalText(interaction.ModalSubmitData().Components, "answer"))
	record, err := env.Store.GetInfernalCase(ctx, parts[1], parts[2])
	if err != nil || !record.Active || !time.Now().Before(record.ExpiresAt) ||
		(interaction.GuildID != "" && record.ThreadID != interaction.ChannelID) ||
		!strings.EqualFold(strings.TrimSpace(record.Answer), strings.TrimSpace(answer)) {
		return respond(session, interaction, "❌ Esa respuesta no es correcta o la tarea ya terminó.", true)
	}
	if err := session.GuildMemberTimeout(parts[1], parts[2], nil); err != nil {
		return fmt.Errorf("restaurar acceso tras tarea de redención: %w", err)
	}
	threadID, err := env.Store.CompleteInfernalTask(ctx, parts[1], parts[2], answer)
	if err != nil {
		return respond(session, interaction, "❌ Esa respuesta no es correcta o la tarea ya terminó.", true)
	}
	_, sendErr := discordutil.ChannelMessageSend(session, threadID, "✅ **Redención completada.** Tu timeout fue retirado y el staff recibió la actualización.")
	if sendErr != nil {
		return fmt.Errorf("publicar finalización de redención: %w", sendErr)
	}
	archived := true
	if _, err := session.ChannelEditComplex(threadID, &discordgo.ChannelEdit{Archived: &archived}); err != nil {
		return fmt.Errorf("archivar hilo tras redención: %w", err)
	}
	return respond(session, interaction, "🕊️ Tarea completada. Tu timeout fue retirado.", true)
}

func investigationButtons(guildID, targetID string, permissions int64) []discordgo.MessageComponent {
	button := func(action, label string, style discordgo.ButtonStyle) discordgo.MessageComponent {
		return discordgo.Button{
			CustomID: fmt.Sprintf("investigate:%s:%s:%s", guildID, targetID, action),
			Label:    label,
			Style:    style,
		}
	}
	rows := make([]discordgo.MessageComponent, 0, 3)
	if hasStaffPermission(permissions, discordgo.PermissionManageMessages) ||
		hasStaffPermission(permissions, discordgo.PermissionManageGuild) {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			button("evidence", "📁 Evidencias", discordgo.SecondaryButton),
			button("add_evidence", "➕ Añadir evidencia", discordgo.SecondaryButton),
			button("interrogate", "💬 Interrogar", discordgo.PrimaryButton),
			button("responses", "📨 Respuestas", discordgo.SecondaryButton),
		}})
	}
	modActions := make([]discordgo.MessageComponent, 0, 3)
	if hasStaffPermission(permissions, discordgo.PermissionModerateMembers) {
		modActions = append(modActions, button("timeout", "🔇 Timeout 10 min", discordgo.DangerButton))
	}
	if hasStaffPermission(permissions, discordgo.PermissionKickMembers) {
		modActions = append(modActions, button("kick", "👢 Expulsar", discordgo.DangerButton))
	}
	if hasStaffPermission(permissions, discordgo.PermissionBanMembers) {
		modActions = append(modActions, button("ban", "⛔ Banear", discordgo.DangerButton))
	}
	if len(modActions) > 0 {
		rows = append(rows, discordgo.ActionsRow{Components: modActions})
	}
	return rows
}

func hasAnyStaffPermission(permissions int64) bool {
	return permissions&(discordgo.PermissionManageMessages|discordgo.PermissionManageGuild|discordgo.PermissionModerateMembers|
		discordgo.PermissionKickMembers|discordgo.PermissionBanMembers|discordgo.PermissionAdministrator) != 0
}

func hasStaffPermission(permissions, required int64) bool {
	return permissions&discordgo.PermissionAdministrator != 0 || permissions&required == required
}

func intPtr(value int) *int {
	return &value
}

func lastDigits(value string, count int) string {
	if len(value) <= count {
		return value
	}
	return value[len(value)-count:]
}

func userOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string, session *discordgo.Session) *discordgo.User {
	for _, option := range options {
		if option.Name == name {
			return option.UserValue(session)
		}
	}
	return nil
}

func optionString(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if option.Name == name {
			return option.StringValue()
		}
	}
	return ""
}

func optionInt(options []*discordgo.ApplicationCommandInteractionDataOption, name string) int64 {
	for _, option := range options {
		if option.Name == name {
			return option.IntValue()
		}
	}
	return 0
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

func truncate(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max-1]) + "…"
}

func safeText(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "@", "@\u200b"))
}

func threadName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
	runes := []rune(value)
	if len(runes) > 90 {
		value = string(runes[:90])
	}
	if value == "" {
		return "lux-inframundo"
	}
	return value
}

func emptyNote(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Sin notas"
	}
	return value
}

func userCreationTime(userID string) time.Time {
	var id int64
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		return time.Time{}
	}
	timestamp := (id >> 22) + 1420070400000
	return time.UnixMilli(timestamp)
}

func floatPtr(value float64) *float64 {
	return &value
}
