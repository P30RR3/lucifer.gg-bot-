package components

import "krypton.dev/pkg/discordutil"
import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/ui"
)

// RegisterAll registra los manejadores de botones, selectores y formularios emergentes
func RegisterAll(router *bot.Router, env *bot.Env) {
	// 1. Botón: Reclamar Pase VIP (Autorol)
	router.RegisterLegacyComponent("btn_claim_vip", handleClaimVIP)

	// 2. Botones de la barra de tragos
	router.RegisterLegacyComponent("btn_drink_whisky", handleOrderWhisky)
	router.RegisterLegacyComponent("btn_drink_martini", handleOrderMartini)
	router.RegisterLegacyComponent("btn_drink_bourbon", handleOrderBourbon)

	// 5. Botón: Relanzar Dados del Trato con el Diablo
	router.RegisterLegacyComponent("btn_reroll_deal", handleRerollDeal)

	// 6. Menú Desplegable (Select Menu): Navegador de Ayuda
	router.RegisterLegacyComponent("select_help_category", handleHelpSelect)

	// 7. Botón: Descartar Alerta / Notificación (Solo Moderadores)
	router.RegisterLegacyComponent("btn_dismiss_alert", handleDismissAlert)

	// 8. Botón: consejo breve de bienestar
	router.RegisterLegacyComponent("btn_quick_therapy", handleQuickTherapy)
}

// ==========================================
// 1. RECLAMO DE PASE VIP (AUTOROL CON BOTÓN)
// ==========================================

func handleClaimVIP(ctx *bot.Context) {
	guild, err := ctx.Session.Guild(ctx.Interaction.GuildID)
	if err != nil {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "❌ No se pudo verificar la información del club.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Buscar un rol representativo (o el configurado)
	var vipRole *discordgo.Role
	for _, r := range guild.Roles {
		lower := strings.ToLower(r.Name)
		if strings.Contains(lower, "miembro") || strings.Contains(lower, "vip") || strings.Contains(lower, "alma") {
			vipRole = r
			break
		}
	}

	if vipRole == nil {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "⚠️ El dueño del club aún no ha designado un rol de bienvenida.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Otorgar el rol al usuario
	err = ctx.Session.GuildMemberRoleAdd(ctx.Interaction.GuildID, ctx.Interaction.Member.User.ID, vipRole.ID)
	if err != nil {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "❌ Lucifer no pudo asignarte el pase. Revisa la jerarquía de roles del bot.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Success().
		Title("🎟️ Pase VIP Otorgado").
		Desc("%s", fmt.Sprintf("Bienvenido formalmente, %s. Ahora portas el rol **@%s**.\n\n*«Disfruta de la estancia... y no toques el piano sin permiso.»*", ctx.Interaction.Member.User.Mention(), vipRole.Name)).
		Build()

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

// ==========================================
// 2. LA BARRA DE TRAGOS DE LUCIFER
// ==========================================

func handleOrderWhisky(ctx *bot.Context) {
	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Title("🥃 Scotch de Malta en las Rocas").
		Desc("%s", fmt.Sprintf("*(Lucifer desliza el vaso por la madera brillante de la barra hacia %s)*\n\n«Un Macallan de 18 años. Solo para paladares que aprecian el verdadero fuego. Salud, querido.»", ctx.Interaction.Member.User.Mention())).
		Thumbnail("https://cdn-icons-png.flaticon.com/512/920/920582.png").
		Build()

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func handleOrderMartini(ctx *bot.Context) {
	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Title("🍸 Dry Martini Clásico").
		Desc("%s", fmt.Sprintf("*(Lucifer sirve la copa con una aceituna atravesada por un palillo plateado)*\n\n«Agitado, jamás mezclado. Un clásico elegante para %s. Que esta noche tus deseos se cumplan.»", ctx.Interaction.Member.User.Mention())).
		Thumbnail("https://cdn-icons-png.flaticon.com/512/1598/1598189.png").
		Build()

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func handleOrderBourbon(ctx *bot.Context) {
	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Title("🥃 Bourbon Doble del Lux").
		Desc("%s", fmt.Sprintf("*(Lucifer sirve el bourbon con una sonrisa cómplice frente a %s)*\n\n«Una elección audaz. Disfrútalo; las noches memorables empiezan con buen gusto.»", ctx.Interaction.Member.User.Mention())).
		Thumbnail("https://cdn-icons-png.flaticon.com/512/2405/2405479.png").
		Build()

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

// ==========================================
// 4. DADOS INTERACTIVOS (TRATO CON EL DIABLO)
// ==========================================

func handleRerollDeal(ctx *bot.Context) {
	rand.Seed(time.Now().UnixNano())
	d1 := rand.Intn(6) + 1
	d2 := rand.Intn(6) + 1
	total := d1 + d2

	var outcome string
	color := ctx.Env.Theme.ColorWarning

	if total > 7 {
		outcome = "🎉 **¡Ganaste el tiro!** Lucifer sonríe con respeto. El Diablo te debe un favor."
		color = ctx.Env.Theme.ColorSuccess
	} else if total == 7 {
		outcome = "⚖️ **Empate milagroso.** La moneda queda parada de canto. Tu alma sigue intacta por ahora."
	} else {
		outcome = "🔥 **Perdiste.** Lucifer te mira con ojos escarlata. Has cedido un pedazo de tu voluntad al Lux."
		color = ctx.Env.Theme.ColorDanger
	}

	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Color(color).
		Title("🎲 Nueva Tirada de Dados").
		Desc("%s", fmt.Sprintf("%s lanzó los dados de nuevo:\n\n**Dados:** `[%d]` + `[%d]` = **Total: %d**\n\n%s", ctx.Interaction.Member.User.Mention(), d1, d2, total, outcome)).
		Build()

	btn := discordgo.Button{
		Label:    "Lanzar de nuevo 🎲",
		Style:    discordgo.DangerButton,
		CustomID: "btn_reroll_deal",
	}

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}},
		},
	})
}

// ==========================================
// 5. SELECTOR DE AYUDA (INTERACTIVO)
// ==========================================

func handleHelpSelect(ctx *bot.Context) {
	data := ctx.Interaction.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}

	selected := data.Values[0]
	builder := ui.NewBuilder(ctx.Env.Theme)

	switch selected {
	case "help_general":
		builder.Persona(ui.PersonaLucifer).
			Title("🍸 Comandos Generales & Club").
			Field("/lux", "Estadísticas exclusivas del servidor", false).
			Field("/perfil", "Consulta el historial y fecha de entrada de un usuario", false).
			Field("/terapia", "Consulta tu historial y ajustes de confidencialidad", false).
			Field("/celestial", "Abre misiones, clasificación y bendiciones del servidor", false).
			Field("/avatar", "Visualiza avatares en máxima resolución", false).
			Field("/ping", "Verifica la latencia de respuesta con el inframundo", false)
	case "help_fun":
		builder.Persona(ui.PersonaLucifer).
			Title("😈 Diversión, IA & Rol").
			Field("/deseo", "Lucifer extrae lo que realmente deseas", false).
			Field("/ask", "Conversa con Lucifer (Cerebras AI)", false).
			Field("/roast", "Lucifer prepara una broma ligera y cordial", false).
			Field("/trato", "Minijuego de dados apostando la suerte con el Diablo", false).
			Field("/piano", "Lucifer compone versos a la medida", false).
			Field("/deseo-profundo", "Publica un deseo de forma anónima", false)
	case "help_mod":
		builder.Persona(ui.PersonaLucifer).
			Danger().
			Title("🛡️ Sanciones & Moderación (Solo Staff)").
			Field("/banear", "Lucifer aplica un baneo autorizado por el staff", false).
			Field("/expulsar", "Saca a patadas a un infractor del club (Kick)", false).
			Field("/silenciar", "Aplica un Timeout temporal de minutos", false).
			Field("/limpiar", "Borra hasta 100 mensajes de un canal al instante", false).
			Field("/panel-staff", "Revisa expedientes, evidencias y acciones permitidas", false).
			Field("/puertas-infernales", "Administra sanciones, apelaciones y bloqueos de emergencia", false).
			Field("/infierno", "Envía a un miembro al bucle de redención", false).
			Field("/inframundo menu", "Supervisa expedientes y absoluciones", false)
	case "help_util":
		builder.Persona(ui.PersonaLucifer).
			Title("🕵️‍♀️ Utilidad & Auditoría").
			Field("/panel-staff", "Lucifer abre el expediente de un miembro", false).
			Field("/puertas-infernales", "Revisa baneos, silencios, apelaciones y alertas", false).
			Field("/inframundo menu", "Supervisa expedientes y absoluciones", false).
			Field("/infierno", "Envía a un miembro al bucle de redención", false).
			Field("/ideas panel", "Consulta, vota y resuelve propuestas", false).
			Field("/agenda panel", "Consulta eventos y reserva un lugar", false).
			Field("/auditoria", "Lucifer examina la seguridad y la configuración", false).
			Field("/recordatorio", "Lucifer te envía un recordatorio automático", false).
			Field("/encuesta", "Crea votaciones elegantes con opciones", false).
			Field("/reloj", "Consulta la hora actual en Los Ángeles y otras capitales", false)
	}

	embed := builder.Build()

	// Actualizamos el mensaje original sin crear uno nuevo
	err := discordutil.InteractionRespond(ctx.Session, ctx.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: ctx.Interaction.Message.Components,
		},
	})
	if err != nil {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "❌ No se pudo actualizar el menú.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
	}
}

// ==========================================
// 6. CONTROL DE ALERTAS & TERAPIA
// ==========================================

func handleDismissAlert(ctx *bot.Context) {
	// Solo miembros con permisos de moderación pueden cerrar la alerta
	perms, _ := ctx.Session.UserChannelPermissions(ctx.Interaction.Member.User.ID, ctx.Interaction.ChannelID)
	if perms&discordgo.PermissionManageMessages == 0 && perms&discordgo.PermissionAdministrator == 0 {
		ctx.Respond(&discordgo.InteractionResponseData{
			Content: "⛔ Solo el staff con permisos de moderación puede archivar esta alerta.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	_ = ctx.Session.ChannelMessageDelete(ctx.Interaction.ChannelID, ctx.Interaction.Message.ID)
}

func handleQuickTherapy(ctx *bot.Context) {
	tips := []string{
		"La culpa es un sentimiento inútil a menos que te impulse a cambiar el presente.",
		"No tienes que complacer a todos en este club. Tu paz mental vale más que su aprobación.",
		"A veces, dar un paso atrás no es cobardía, es perspectiva.",
		"Recuerda que incluso Lucifer aprendió a reconocer sus vulnerabilidades.",
	}
	rand.Seed(time.Now().UnixNano())
	chosenTip := tips[rand.Intn(len(tips))]

	embed := ui.NewBuilder(ctx.Env.Theme).
		Persona(ui.PersonaLucifer).
		Success().
		Title("🕯️ Una pausa con Lucifer").
		Desc("%s", fmt.Sprintf("%s, respira hondo:\n\n> *\"%s\"*", ctx.Interaction.Member.User.Mention(), chosenTip)).
		Footer("Bienestar del Lux").
		Build()

	ctx.Respond(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

// ==========================================
// GENERADORES DE COMPONENTES REUTILIZABLES
// ==========================================

// NewDrinkButtonsRow genera la botonera interactiva del bar
func NewDrinkButtonsRow() discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Scotch Whisky 🥃",
				Style:    discordgo.SecondaryButton,
				CustomID: "btn_drink_whisky",
			},
			discordgo.Button{
				Label:    "Dry Martini 🍸",
				Style:    discordgo.PrimaryButton,
				CustomID: "btn_drink_martini",
			},
			discordgo.Button{
				Label:    "Bourbon Fuerte 🔥",
				Style:    discordgo.DangerButton,
				CustomID: "btn_drink_bourbon",
			},
		},
	}
}

// NewVIPClaimRow genera el botón para reclamar el rol de bienvenida
func NewVIPClaimRow() discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Reclamar Pase VIP 🎟️",
				Style:    discordgo.SuccessButton,
				CustomID: "btn_claim_vip",
			},
		},
	}
}
