package config

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(r *bot.Registry) error {
	if err := r.Register(&bot.Command{
		Name:        "configuracion",
		Description: "Abre el asistente de configuración del servidor.",
		Category:    "config",
		DefaultPerm: discordgo.PermissionManageGuild,
		GuildOnly:   true,
		Handler:     c.handleSetup,
	}); err != nil {
		return err
	}

	// Comando raíz /config con subcomandos administrativos
	return r.AddCommand(&bot.Command{
		Category: "config",
		AppCmd: &discordgo.ApplicationCommand{
			Name:                     "config",
			Description:              "⚙️ Ajusta con elegancia los decretos y la seguridad del servidor.",
			DefaultMemberPermissions: int64Ptr(discordgo.PermissionAdministrator),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "ver",
					Description: "📜 Consulta el estado de los ajustes y defensas del servidor.",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "logs",
					Description: "📝 Elige dónde se escribirá el diario privado del Inframundo.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "canal",
							Description: "Canal de texto para el registro de seguridad",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "autorol",
					Description: "Asigna un rol automático al entrar (Pase VIP de recién llegado) 🎟️",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "rol",
							Description: "Rol otorgado automáticamente",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "alertas",
					Description: "🚨 Designa el canal para avisos urgentes del protocolo del Lux.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "canal",
							Description: "Canal de alertas de incursiones",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "antiraid",
					Description: "🔒 Ajusta la sensibilidad del protocolo antiraid.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "modo",
							Description: "Nivel de vigilancia de las puertas del club",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Tranquilo (Solo ataques masivos)", Value: "low"},
								{Name: "Equilibrado (Recomendado para servidores públicos)", Value: "medium"},
								{Name: "Estricto (Tolerancia cero a sospechosos)", Value: "strict"},
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "ia",
					Description: "🧠 Activa o descansa las funciones inteligentes del Lux.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "activado",
							Description: "¿Habilitar la terapia y respuestas neuronales?",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "reset",
					Description: "⚠️ Restaura los decretos del club a sus valores iniciales.",
				},
			},
		},
		LegacyHandler: c.handleRouter,
	})
}

// Router central de subcomandos
func (c *Cog) handleRouter(ctx *bot.Context) {
	data := ctx.Interaction.ApplicationCommandData()
	if len(data.Options) == 0 {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "⚠️ Debes especificar un subcomando de configuración.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	subcmd := data.Options[0]
	switch subcmd.Name {
	case "ver":
		c.subVer(ctx)
	case "logs":
		c.subLogs(ctx, subcmd)
	case "autorol":
		c.subAutorol(ctx, subcmd)
	case "alertas":
		c.subAlertas(ctx, subcmd)
	case "antiraid":
		c.subAntiraid(ctx, subcmd)
	case "ia":
		c.subIA(ctx, subcmd)
	case "reset":
		c.subReset(ctx)
	default:
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "❌ Subcomando desconocido.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
	}
}

// ==========================================
// SUBCOMANDOS
// ==========================================

func (c *Cog) subVer(ctx *bot.Context) {
	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "📜 *Abriendo la caja fuerte del despacho de Lucifer...* `[■■■□□□□□□□]`",
	})
	time.Sleep(800 * time.Millisecond)

	guild, _ := ctx.Session.Guild(ctx.Interaction.GuildID)

	embed := &discordgo.MessageEmbed{
		Title:       "🏛️ Decretos del Lux: " + guild.Name,
		Description: "Estado de la configuración operativa y de seguridad en este servidor.",
		Color:       ctx.Env.Theme.Primary,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: guild.IconURL("256")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📝 Canal de Auditoría (Logs)", Value: "<#1234567890> *(No asignado)*", Inline: true},
			{Name: "🚪 Puerta de Entrada (Bienvenidas)", Value: "*(Desactivado)*", Inline: true},
			{Name: "🎟️ Pase VIP (Autorol)", Value: "*(Ninguno)*", Inline: true},
			{Name: "🛡️ Guardia Pretoriana (Staff)", Value: "*(Permisos de Discord)*", Inline: true},
			{Name: "🚨 Canal de Alertas Antiraid", Value: "*(Canal de logs)*", Inline: true},
			{Name: "🔒 Sensibilidad Antiraid", Value: "🛡️ **Equilibrado** (Modo público)", Inline: true},
			{Name: "🧠 IA de Lucifer (Cerebras)", Value: "✅ **Activa**", Inline: true},
			{Name: "👑 Dueño Registrado", Value: fmt.Sprintf("<@%s>", guild.OwnerID), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Usa /config <opcion> para modificar cualquiera de estos parámetros.",
			IconURL: ctx.Env.Theme.FooterIcon,
		},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Expediente cargado con éxito.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subLogs(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channel := opt.Options[0].ChannelValue(ctx.Session)

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "📝 *Sincronizando el diario de auditoría...* `[■■■■□□□□□□]`",
	})
	time.Sleep(700 * time.Millisecond)

	// Validar permisos de escritura en el canal
	perms, err := ctx.Session.UserChannelPermissions(ctx.Session.State.User.ID, channel.ID)
	if err != nil || perms&discordgo.PermissionSendMessages == 0 {
		ctx.EditResponse(&discordgo.WebhookEdit{
			Content: stringPtr("❌ *No tengo permisos para enviar mensajes en ese canal. Revisa mis roles.*"),
		})
		return
	}

	settings, err := ctx.Env.Store.GetAuditSettings(ctx.Context, ctx.Interaction.GuildID)
	if err != nil {
		ctx.EditResponse(&discordgo.WebhookEdit{
			Content: stringPtr("❌ *No pude consultar la configuración del diario.*"),
		})
		ctx.Env.Log.Error("falló la lectura de configuración del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	settings.GuildID = ctx.Interaction.GuildID
	settings.LogChannelID = channel.ID
	if err := ctx.Env.Store.SaveAuditSettings(ctx.Context, settings); err != nil {
		ctx.EditResponse(&discordgo.WebhookEdit{
			Content: stringPtr("❌ *No pude guardar el canal del diario.*"),
		})
		ctx.Env.Log.Error("falló el guardado de configuración del diario", "guild_id", ctx.Interaction.GuildID, "err", err)
		return
	}
	if ctx.Env.Auditor != nil {
		ctx.Env.Auditor.UpdateSettings(settings)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📝 Diario del Inframundo Actualizado",
		Description: fmt.Sprintf("A partir de ahora, todos los incidentes, sanciones y expulsiones se redactarán en <#%s>.", channel.ID),
		Color:       ctx.Env.Theme.ColorSuccess,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Configuración guardada en la base de datos."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Decreto promulgado.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subBienvenidas(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channel := opt.Options[0].ChannelValue(ctx.Session)

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "🚪 *Preparando alfombra roja de recepción...* `[■■■■■□□□□□]`",
	})
	time.Sleep(700 * time.Millisecond)

	embed := &discordgo.MessageEmbed{
		Title:       "🚪 Entrada del Club Configurada",
		Description: fmt.Sprintf("Las nuevas almas que ingresen serán recibidas en <#%s> con la hospitalidad de Lucifer.", channel.ID),
		Color:       ctx.Env.Theme.ColorSuccess,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Recepción activa."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Puertas abiertas.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subAutorol(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	role := opt.Options[0].RoleValue(ctx.Session, ctx.Interaction.GuildID)

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "🎟️ *Sellando pases VIP automáticos...* `[■■■■■□□□□□]`",
	})
	time.Sleep(700 * time.Millisecond)

	guild, _ := ctx.Session.Guild(ctx.Interaction.GuildID)
	botMember, _ := ctx.Session.GuildMember(guild.ID, ctx.Session.State.User.ID)

	// Validar jerarquía del rol
	if !isRoleManageable(guild, botMember, role) {
		ctx.EditResponse(&discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("❌ *No puedo otorgar el rol <@&%s> porque está por encima o igual a mi rol más alto en la jerarquía del servidor.*", role.ID)),
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎟️ Pase VIP Automático Asignado",
		Description: fmt.Sprintf("Todos los mortales que pongan un pie en el club recibirán automáticamente el rol <@&%s>.", role.ID),
		Color:       ctx.Env.Theme.ColorSuccess,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Autorol enlazado con éxito."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Pases listos.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subStaff(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	role := opt.Options[0].RoleValue(ctx.Session, ctx.Interaction.GuildID)

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "🛡️ *Lucifer está designando la guardia del club...* `[■■■■■□□□□□]`",
	})
	time.Sleep(700 * time.Millisecond)

	embed := &discordgo.MessageEmbed{
		Title:       "🛡️ Guardia Pretoriana Designada",
		Description: fmt.Sprintf("Los portadores del rol <@&%s> ahora son reconocidos por Lucifer como personal autorizado del club.", role.ID),
		Color:       ctx.Env.Theme.Primary,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Privilegios de staff concedidos."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Guardia juramentada.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subAlertas(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channel := opt.Options[0].ChannelValue(ctx.Session)

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "🚨 *Alineando sirenas y campanas de emergencia...* `[■■■■■□□□□□]`",
	})
	time.Sleep(700 * time.Millisecond)

	embed := &discordgo.MessageEmbed{
		Title:       "🚨 Centro de Emergencias Establecido",
		Description: fmt.Sprintf("Si el motor Antiraid detecta ataques masivos o incursiones de bots, <#%s> será notificado al instante.", channel.ID),
		Color:       ctx.Env.Theme.ColorDanger,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Línea directa de emergencia lista."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Canal blindado.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subAntiraid(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	mode := opt.Options[0].StringValue()

	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "🔒 *Ajustando cerrojos en las puertas del infierno...* `[■■■■■■□□□□]`",
	})
	time.Sleep(800 * time.Millisecond)

	var modeTitle, modeDesc string
	var modeColor int

	switch mode {
	case "low":
		modeTitle = "Modo Tranquilo Activado"
		modeDesc = "El bot solo intervendrá ante inundaciones masivas evidentes de cuentas. Recomendado para comunidades muy pequeñas."
		modeColor = ctx.Env.Theme.ColorNeutral
	case "medium":
		modeTitle = "Modo Equilibrado Activado (Recomendado)"
		modeDesc = "El protocolo de Lucifer inspeccionará cuentas sospechosas, spam de enlaces y menciones indebidas sin entorpecer la charla cotidiana."
		modeColor = ctx.Env.Theme.ColorWarning
	case "strict":
		modeTitle = "Modo Estricto Activado (Defensa Máxima)"
		modeDesc = "Cuentas recién creadas o con comportamiento errático serán aisladas inmediatamente. Ideal si estás bajo ataque activo."
		modeColor = ctx.Env.Theme.ColorDanger
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔒 " + modeTitle,
		Description: modeDesc,
		Color:       modeColor,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Protocolos de seguridad reconfigurados."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Defensas operativas.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subIA(ctx *bot.Context, opt *discordgo.ApplicationCommandInteractionDataOption) {
	enabled := opt.Options[0].BoolValue()

	status := "Desactivada"
	desc := "Las funciones de IA permanecerán en reposo. Los comandos inteligentes del Lux no estarán disponibles."
	color := ctx.Env.Theme.ColorNeutral

	if enabled {
		status = "Habilitada"
		desc = "Cerebras AI está activo. Los miembros del servidor pueden conversar con Lucifer mediante `/ask`."
		color = ctx.Env.Theme.ColorSuccess
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🧠 Cerebras AI: %s", status),
		Description: desc,
		Color:       color,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Módulo neuronal actualizado."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) subReset(ctx *bot.Context) {
	ctx.Respond(&discordgo.InteractionResponseData{
		Content: "⚠️ *Quemando los antiguos pergaminos...* `[■■■□□□□□□□]`",
	})
	time.Sleep(1 * time.Second)

	embed := &discordgo.MessageEmbed{
		Title:       "♻️ Decretos Restaurados",
		Description: "Toda la configuración del servidor ha vuelto a sus valores por defecto de fábrica. Los canales de logs, pases VIP y alertas han quedado desvinculados.",
		Color:       ctx.Env.Theme.ColorDanger,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Reinicio administrativo completado."},
	}
	ctx.Env.Theme.BrandEmbed(embed)

	ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr("✅ *Tabula rasa. Todo reiniciado.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

// ==========================================
// AUXILIARES
// ==========================================

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func isRoleManageable(guild *discordgo.Guild, botMember *discordgo.Member, targetRole *discordgo.Role) bool {
	if botMember == nil || guild.OwnerID == targetRole.ID {
		return false
	}

	var highestBotPos int
	for _, rID := range botMember.Roles {
		for _, r := range guild.Roles {
			if r.ID == rID && r.Position > highestBotPos {
				highestBotPos = r.Position
			}
		}
	}

	return highestBotPos > targetRole.Position
}
