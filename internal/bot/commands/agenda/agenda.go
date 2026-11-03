package agenda

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

const (
	maxEventTitle       = 80
	maxEventDescription = 1500
	maxEventCapacity    = 1000
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(registry *bot.Registry) error {
	return registry.Register(&bot.Command{
		Name:        "agenda",
		Description: "Abre el panel interactivo de eventos del Lux.",
		Category:    "eventos",
		GuildOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "panel",
				Description: "Consulta, crea, reserva o cancela eventos con botones.",
			},
		},
		Handler: c.handleAgenda,
	})
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("agenda", handleAgendaComponent)
	router.RegisterModal("agenda_create", handleAgendaCreateModal)
}

func (c *Cog) handleAgenda(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 {
		return respond(session, interaction, "Usa `/agenda panel` para abrir los eventos del Lux.", true)
	}
	if options[0].Name != "panel" {
		return respond(session, interaction, "Esa acción no existe en la agenda.", true)
	}
	return showAgendaPanel(ctx, session, interaction, env, 0, "")
}

func showAgendaPanel(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, selectedID int64, notice string) error {
	events, err := env.Store.ListLuxEvents(ctx, interaction.GuildID, 20)
	if err != nil {
		return fmt.Errorf("cargar panel de agenda: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🗓️・LA AGENDA DEL LUX",
		Description: "Selecciona una noche para ver sus detalles y reservar tu lugar.",
		Color:       env.Theme.Primary,
	}
	if notice != "" {
		embed.Description = notice + "\n\n" + embed.Description
	}
	options := make([]discordgo.SelectMenuOption, 0, len(events))
	for _, event := range events {
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncateAgendaOption(fmt.Sprintf("#%d・%s", event.ID, event.Title), 100),
			Value:       strconv.FormatInt(event.ID, 10),
			Description: truncateAgendaOption(fmt.Sprintf("%s・%s", event.StartsAt.Format("02 Jan · 15:04 MST"), attendanceLabel(&event)), 100),
			Default:     event.ID == selectedID,
		})
	}
	if len(events) == 0 {
		embed.Description = "No hay eventos próximos. El staff puede crear el siguiente desde este panel."
		if notice != "" {
			embed.Description = notice + "\n\n" + embed.Description
		}
	} else if selectedID > 0 {
		event, err := env.Store.GetLuxEvent(ctx, interaction.GuildID, selectedID)
		switch {
		case errors.Is(err, storage.ErrLuxEventNotFound):
			selectedID = 0
		case err != nil:
			return fmt.Errorf("cargar evento seleccionado: %w", err)
		default:
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: fmt.Sprintf("🍸 #%d・%s", event.ID, safeText(event.Title)), Value: safeText(event.Description)},
				{Name: "🕰️ Inicio", Value: fmt.Sprintf("<t:%d:F>\n<t:%d:R>", event.StartsAt.Unix(), event.StartsAt.Unix()), Inline: true},
				{Name: "🥂 Aforo", Value: attendanceLabel(event), Inline: true},
				{Name: "👑 Organiza", Value: fmt.Sprintf("<@%s>", event.CreatorID), Inline: true},
			}
			if event.CreatorID == interaction.Member.User.ID || hasPermission(interaction, discordgo.PermissionManageEvents) {
				embed.Footer = &discordgo.MessageEmbedFooter{Text: "Como organizador o staff, puedes cancelar el evento desde los controles."}
			}
		}
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{CustomID: fmt.Sprintf("agenda:%s:create", interaction.Member.User.ID), Label: "✨ Crear evento", Style: discordgo.SuccessButton, Disabled: !hasPermission(interaction, discordgo.PermissionManageEvents)},
		}},
	}
	if len(options) > 0 {
		components = append([]discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.StringSelectMenu,
				CustomID:    fmt.Sprintf("agenda:%s:select", interaction.Member.User.ID),
				Placeholder: "✦ Elige un evento ✦",
				Options:     options,
				MinValues:   intPointer(1),
				MaxValues:   1,
			},
		}}}, components...)
		if selectedID > 0 {
			components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{CustomID: fmt.Sprintf("agenda:%s:join:%d", interaction.Member.User.ID, selectedID), Label: "🥂 Reservar lugar", Style: discordgo.PrimaryButton},
				discordgo.Button{CustomID: fmt.Sprintf("agenda:%s:leave:%d", interaction.Member.User.ID, selectedID), Label: "↩️ Liberar lugar", Style: discordgo.SecondaryButton},
			}})
			event, err := env.Store.GetLuxEvent(ctx, interaction.GuildID, selectedID)
			if err != nil {
				return fmt.Errorf("comprobar organizador del evento: %w", err)
			}
			if event.CreatorID == interaction.Member.User.ID || hasPermission(interaction, discordgo.PermissionManageEvents) {
				components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{CustomID: fmt.Sprintf("agenda:%s:cancel:%d", interaction.Member.User.ID, selectedID), Label: "🚫 Cancelar evento", Style: discordgo.DangerButton},
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

func handleAgendaComponent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	if interaction.Member == nil || interaction.Member.User == nil {
		return respond(session, interaction, "Este panel solo funciona dentro del servidor.", true)
	}
	parts := strings.Split(interaction.MessageComponentData().CustomID, ":")
	if len(parts) < 3 {
		return respond(session, interaction, "Este control de la agenda ya no es válido. Abre `/agenda panel` para actualizarlo.", true)
	}
	action := parts[2]
	if action == "create" {
		if !hasPermission(interaction, discordgo.PermissionManageEvents) {
			return respond(session, interaction, "Crear eventos requiere el permiso Gestionar eventos.", true)
		}
		return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("agenda_create:%s:%s", interaction.GuildID, interaction.Member.User.ID),
				Title:    "✨ Nueva noche en el Lux",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "title", Label: "Nombre del evento", Style: discordgo.TextInputShort, Required: true, MinLength: 5, MaxLength: maxEventTitle},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "description", Label: "Detalles", Style: discordgo.TextInputParagraph, Required: true, MinLength: 10, MaxLength: maxEventDescription},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "start", Label: "Inicio ISO 8601 con zona horaria", Style: discordgo.TextInputShort, Required: true, MaxLength: 35, Placeholder: "2026-12-01T20:30:00-04:00"},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "capacity", Label: "Cupo (0 = sin límite)", Style: discordgo.TextInputShort, Required: true, MaxLength: 4, Value: "0"},
					}},
				},
			},
		})
	}
	if action == "select" {
		values := interaction.MessageComponentData().Values
		if len(values) != 1 {
			return respond(session, interaction, "Elige un solo evento.", true)
		}
		id, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || id < 1 {
			return respond(session, interaction, "El evento seleccionado no es válido.", true)
		}
		return showAgendaPanel(ctx, session, interaction, env, id, "")
	}
	if len(parts) != 4 {
		return respond(session, interaction, "Este control ya no es válido.", true)
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || id < 1 {
		return respond(session, interaction, "El evento seleccionado no es válido.", true)
	}
	switch action {
	case "join":
		attendees, err := env.Store.JoinLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID)
		switch {
		case errors.Is(err, storage.ErrLuxEventNotFound), errors.Is(err, storage.ErrLuxEventClosed):
			return respond(session, interaction, "Ese evento ya no está disponible.", true)
		case errors.Is(err, storage.ErrLuxEventFull):
			return respond(session, interaction, "No quedan lugares disponibles para esa noche.", true)
		case errors.Is(err, storage.ErrLuxEventJoined):
			return respond(session, interaction, "Ya tienes un lugar reservado.", true)
		case err != nil:
			return fmt.Errorf("reservar lugar desde panel de agenda: %w", err)
		}
		return showAgendaPanel(ctx, session, interaction, env, id, fmt.Sprintf("✅ Lugar reservado. Ahora hay **%d asistentes**.", attendees))
	case "leave":
		attendees, err := env.Store.LeaveLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID)
		switch {
		case errors.Is(err, storage.ErrLuxEventNotFound), errors.Is(err, storage.ErrLuxEventClosed):
			return respond(session, interaction, "Ese evento ya no está disponible.", true)
		case errors.Is(err, storage.ErrLuxEventNotJoined):
			return respond(session, interaction, "No tienes un lugar reservado en ese evento.", true)
		case err != nil:
			return fmt.Errorf("liberar lugar desde panel de agenda: %w", err)
		}
		return showAgendaPanel(ctx, session, interaction, env, id, fmt.Sprintf("↩️ Lugar liberado. Quedan **%d asistentes**.", attendees))
	case "cancel":
		event, err := env.Store.GetLuxEvent(ctx, interaction.GuildID, id)
		if err != nil {
			return fmt.Errorf("consultar evento para cancelación: %w", err)
		}
		moderator := hasPermission(interaction, discordgo.PermissionManageEvents) || hasPermission(interaction, discordgo.PermissionManageGuild)
		if event.CreatorID != interaction.Member.User.ID && !moderator {
			return respond(session, interaction, "Solo quien organiza el evento o el staff autorizado puede cancelarlo.", true)
		}
		if err := env.Store.CancelLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID, moderator); err != nil {
			if errors.Is(err, storage.ErrLuxEventNotFound) || errors.Is(err, storage.ErrLuxEventClosed) {
				return respond(session, interaction, "Ese evento ya no está abierto.", true)
			}
			return fmt.Errorf("cancelar evento desde panel de agenda: %w", err)
		}
		return showAgendaPanel(ctx, session, interaction, env, 0, fmt.Sprintf("🚫 El evento **%s** fue cancelado.", safeText(event.Title)))
	default:
		return respond(session, interaction, "Acción de agenda desconocida.", true)
	}
}

func handleAgendaCreateModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	parts := strings.Split(interaction.ModalSubmitData().CustomID, ":")
	if len(parts) != 3 || interaction.Member == nil || interaction.Member.User == nil ||
		interaction.GuildID != parts[1] || interaction.Member.User.ID != parts[2] ||
		!hasPermission(interaction, discordgo.PermissionManageEvents) {
		return respond(session, interaction, "No tienes autorización para crear eventos en este servidor.", true)
	}
	form := interaction.ModalSubmitData().Components
	title := strings.TrimSpace(modalValue(form, "title"))
	description := strings.TrimSpace(modalValue(form, "description"))
	startsAt, err := parseStartTime(strings.TrimSpace(modalValue(form, "start")), time.Now())
	if len([]rune(title)) < 5 || len([]rune(title)) > maxEventTitle {
		return respond(session, interaction, "El título debe tener entre 5 y 80 caracteres.", true)
	}
	if len([]rune(description)) < 10 || len([]rune(description)) > maxEventDescription {
		return respond(session, interaction, "Los detalles deben tener entre 10 y 1.500 caracteres.", true)
	}
	if err != nil {
		return respond(session, interaction, "Usa una fecha ISO 8601 futura con zona, como `2026-12-01T20:30:00-04:00`.", true)
	}
	capacity, err := strconv.Atoi(strings.TrimSpace(modalValue(form, "capacity")))
	if err != nil || capacity < 0 || capacity > maxEventCapacity {
		return respond(session, interaction, "El cupo debe ser un número de 0 a 1.000; 0 significa sin límite.", true)
	}
	event := &storage.LuxEvent{
		GuildID: interaction.GuildID, ChannelID: interaction.ChannelID,
		CreatorID: interaction.Member.User.ID, Title: title,
		Description: description, StartsAt: startsAt, Capacity: capacity,
	}
	if err := env.Store.CreateLuxEvent(ctx, event); err != nil {
		return fmt.Errorf("crear evento desde panel: %w", err)
	}
	embed := eventEmbed(event, env)
	env.Theme.BrandEmbed(embed)
	return discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🍸 La noche ya está en la agenda. Abre `/agenda panel` para ver las reservas.",
			Embeds:  []*discordgo.MessageEmbed{embed},
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func modalValue(components []discordgo.MessageComponent, id string) string {
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

func truncateAgendaOption(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func intPointer(value int) *int {
	return &value
}

func (c *Cog) createEvent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	if !hasPermission(interaction, discordgo.PermissionManageEvents) {
		return respond(session, interaction, "Solo el equipo con permiso **Gestionar eventos** puede publicar eventos.", true)
	}

	title := strings.TrimSpace(option(options, "titulo").StringValue())
	description := strings.TrimSpace(option(options, "detalle").StringValue())
	if len([]rune(title)) < 5 || len([]rune(title)) > maxEventTitle {
		return respond(session, interaction, "El título debe tener entre 5 y 80 caracteres.", true)
	}
	if len([]rune(description)) < 10 || len([]rune(description)) > maxEventDescription {
		return respond(session, interaction, "La descripción debe tener entre 10 y 1.500 caracteres.", true)
	}

	startsAt, err := parseStartTime(option(options, "inicio").StringValue(), time.Now())
	if err != nil {
		return respond(session, interaction, "Usa una fecha ISO 8601 completa con zona horaria, por ejemplo `2026-12-01T20:30:00-04:00`.", true)
	}

	capacity := int64(0)
	if capacityOption := option(options, "cupo"); capacityOption != nil {
		capacity = capacityOption.IntValue()
	}
	if capacity < 0 || capacity > maxEventCapacity {
		return respond(session, interaction, "El cupo debe estar entre 0 y 1.000; usa 0 para un evento sin límite.", true)
	}

	event := &storage.LuxEvent{
		GuildID:     interaction.GuildID,
		ChannelID:   interaction.ChannelID,
		CreatorID:   interaction.Member.User.ID,
		Title:       title,
		Description: description,
		StartsAt:    startsAt,
		Capacity:    int(capacity),
	}
	if err := env.Store.CreateLuxEvent(ctx, event); err != nil {
		return fmt.Errorf("crear evento del Lux: %w", err)
	}

	embed := eventEmbed(event, env)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Evento #%d · Inscripciones desde /agenda panel", event.ID),
	}
	if err := discordutil.InteractionRespond(session, interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🍸 Las puertas del Lux están abiertas para este evento.",
			Embeds:  []*discordgo.MessageEmbed{embed},
		},
	}); err != nil {
		return fmt.Errorf("publicar evento del Lux: %w", err)
	}
	return nil
}

func (c *Cog) listEvents(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env) error {
	events, err := env.Store.ListLuxEvents(ctx, interaction.GuildID, 10)
	if err != nil {
		return fmt.Errorf("listar eventos del Lux: %w", err)
	}
	if len(events) == 0 {
		return respond(session, interaction, "No hay eventos próximos. El equipo puede crear uno desde `/agenda panel`.", true)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗓️ Próximas noches en el Lux",
		Description: "Consulta la hora en tu zona con las marcas de tiempo de Discord.",
		Color:       env.Theme.Primary,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Usa /agenda panel para consultar y reservar"},
	}
	for _, event := range events {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("#%d · %s", event.ID, safeText(event.Title)),
			Value:  fmt.Sprintf("<t:%d:F> · %s", event.StartsAt.Unix(), attendanceLabel(&event)),
			Inline: false,
		})
	}
	env.Theme.BrandEmbed(embed)
	return respondEmbed(session, interaction, embed, true)
}

func (c *Cog) showEvent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	event, err := getEvent(ctx, env, interaction, options)
	if errors.Is(err, storage.ErrLuxEventNotFound) {
		return respond(session, interaction, "No encontré ese evento en este servidor.", true)
	}
	if err != nil {
		return fmt.Errorf("consultar evento del Lux: %w", err)
	}
	return respondEmbed(session, interaction, eventEmbed(event, env), true)
}

func (c *Cog) joinEvent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de evento válido.", true)
	}
	attendees, err := env.Store.JoinLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID)
	switch {
	case errors.Is(err, storage.ErrLuxEventNotFound):
		return respond(session, interaction, "No encontré ese evento en este servidor.", true)
	case errors.Is(err, storage.ErrLuxEventClosed):
		return respond(session, interaction, "Ese evento ya fue cancelado.", true)
	case errors.Is(err, storage.ErrLuxEventFull):
		return respond(session, interaction, "No quedan plazas para ese evento.", true)
	case errors.Is(err, storage.ErrLuxEventJoined):
		return respond(session, interaction, "Ya tienes una plaza reservada. Libérala desde `/agenda panel` si cambiaste de planes.", true)
	case err != nil:
		return fmt.Errorf("reservar plaza del evento: %w", err)
	}
	return respond(session, interaction, fmt.Sprintf("✅ Plaza reservada. Ahora hay **%d asistentes**. ¡Nos vemos en el Lux!", attendees), true)
}

func (c *Cog) leaveEvent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de evento válido.", true)
	}
	attendees, err := env.Store.LeaveLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID)
	switch {
	case errors.Is(err, storage.ErrLuxEventNotFound):
		return respond(session, interaction, "No encontré ese evento en este servidor.", true)
	case errors.Is(err, storage.ErrLuxEventClosed):
		return respond(session, interaction, "Ese evento ya no admite cambios de inscripción.", true)
	case errors.Is(err, storage.ErrLuxEventNotJoined):
		return respond(session, interaction, "No tienes una plaza reservada en ese evento.", true)
	case err != nil:
		return fmt.Errorf("liberar plaza del evento: %w", err)
	}
	return respond(session, interaction, fmt.Sprintf("La plaza quedó libre. Quedan **%d asistentes** inscritos.", attendees), true)
}

func (c *Cog) cancelEvent(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, env *bot.Env, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return respond(session, interaction, "Indica un número de evento válido.", true)
	}
	moderator := hasPermission(interaction, discordgo.PermissionManageEvents) ||
		hasPermission(interaction, discordgo.PermissionManageGuild)
	err := env.Store.CancelLuxEvent(ctx, interaction.GuildID, id, interaction.Member.User.ID, moderator)
	switch {
	case errors.Is(err, storage.ErrLuxEventNotFound):
		return respond(session, interaction, "No encontré ese evento en este servidor.", true)
	case errors.Is(err, storage.ErrLuxEventClosed):
		return respond(session, interaction, "Ese evento ya estaba cancelado.", true)
	case errors.Is(err, storage.ErrLuxEventNotOwner):
		return respond(session, interaction, "Solo quien creó el evento o el equipo con permisos puede cancelarlo.", true)
	case err != nil:
		return fmt.Errorf("cancelar evento del Lux: %w", err)
	}

	event, err := env.Store.GetLuxEvent(ctx, interaction.GuildID, id)
	if err != nil {
		return fmt.Errorf("leer evento cancelado: %w", err)
	}
	embed := eventEmbed(event, env)
	embed.Title = "🚫 Evento cancelado · " + safeText(event.Title)
	embed.Color = env.Theme.ColorDanger
	return respondEmbed(session, interaction, embed, false)
}

func getEvent(ctx context.Context, env *bot.Env, interaction *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) (*storage.LuxEvent, error) {
	id := option(options, "id").IntValue()
	if id <= 0 {
		return nil, errors.New("número de evento inválido")
	}
	return env.Store.GetLuxEvent(ctx, interaction.GuildID, id)
}

func eventEmbed(event *storage.LuxEvent, env *bot.Env) *discordgo.MessageEmbed {
	description := safeText(event.Description)
	if event.Status == "cancelled" {
		description = "~~" + description + "~~\n\n**Este evento fue cancelado.**"
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🍸 Evento #%d · %s", event.ID, safeText(event.Title)),
		Description: description,
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Inicio", Value: fmt.Sprintf("<t:%d:F>\n<t:%d:R>", event.StartsAt.Unix(), event.StartsAt.Unix()), Inline: true},
			{Name: "Asistencia", Value: attendanceLabel(event), Inline: true},
			{Name: "Organiza", Value: fmt.Sprintf("ID `%s`", event.CreatorID), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Agenda del Lux · Gestiona tu plaza desde /agenda panel",
			IconURL: env.Theme.FooterIcon,
		},
	}
	env.Theme.BrandEmbed(embed)
	return embed
}

func attendanceLabel(event *storage.LuxEvent) string {
	if event.Capacity == 0 {
		return fmt.Sprintf("%d inscritos · sin límite", event.Attendees)
	}
	return fmt.Sprintf("%d / %d plazas", event.Attendees, event.Capacity)
}

func parseStartTime(value string, now time.Time) (time.Time, error) {
	startsAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	if startsAt.Before(now.Add(10*time.Minute)) || startsAt.After(now.AddDate(0, 6, 0)) {
		return time.Time{}, errors.New("el evento debe iniciar dentro de 10 minutos a 6 meses")
	}
	return startsAt, nil
}

func hasPermission(interaction *discordgo.InteractionCreate, permission int64) bool {
	if interaction.Member == nil {
		return false
	}
	permissions := interaction.Member.Permissions
	return permissions&permission == permission ||
		permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator
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

func safeText(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "@", "@\u200b")
}
