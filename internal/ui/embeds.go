package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// CharacterPersona define la estética del personaje asignado al embed
type CharacterPersona string

const (
	PersonaLucifer CharacterPersona = "lucifer"
	PersonaSystem  CharacterPersona = "system"
)

// ProgressStyle define el conjunto de glifos para renderizar barras visuales
type ProgressStyle string

const (
	ProgressStyleModern ProgressStyle = "modern" // [██████░░░░]
	ProgressStyleFire   ProgressStyle = "fire"   // [🔥🔥🔥💨💨💨]
	ProgressStyleDagger ProgressStyle = "dagger" // [🗡️🗡️🗡️▫️▫️▫️]
	ProgressStyleBlock  ProgressStyle = "block"  // [■■■■□□□□]
)

// Constantes de imágenes y gifs oficiales para el Lux
const (
	GifsLuciferPiano     = "https://media.giphy.com/media/3o7TKoHNJTWWLgljYQ/giphy.gif"
	GifsLuciferDevilFace = "https://media.giphy.com/media/l0HlCqV35hdEg2GUo/giphy.gif"
	IconMartiniLux       = "https://cdn-icons-png.flaticon.com/512/1598/1598189.png"
)

// EmbedBuilder facilita la creación fluida y estandarizada de embeds
type EmbedBuilder struct {
	theme *Theme
	embed *discordgo.MessageEmbed
}

func NewBuilder(theme *Theme) *EmbedBuilder {
	return &EmbedBuilder{
		theme: theme,
		embed: &discordgo.MessageEmbed{
			Color: theme.Primary,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    theme.FooterText,
				IconURL: theme.FooterIcon,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}
}

// Persona aplica los estilos, colores e íconos característicos del personaje
func (b *EmbedBuilder) Persona(p CharacterPersona) *EmbedBuilder {
	switch p {
	case PersonaLucifer:
		b.embed.Color = 0x8B0000 // Rojo diablo oscuro
		b.embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    "Lucifer Morningstar • Dueño del Lux",
			IconURL: IconMartiniLux,
		}
	case PersonaSystem:
		b.embed.Color = b.theme.ColorNeutral
		b.embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    "Lux Security System",
			IconURL: b.theme.AuthorIcon,
		}
	}
	return b
}

func (b *EmbedBuilder) Title(title string) *EmbedBuilder {
	b.embed.Title = title
	return b
}

func (b *EmbedBuilder) Desc(format string, args ...any) *EmbedBuilder {
	if len(args) > 0 {
		b.embed.Description = fmt.Sprintf(format, args...)
	} else {
		b.embed.Description = format
	}
	return b
}

func (b *EmbedBuilder) Color(c int) *EmbedBuilder {
	b.embed.Color = c
	return b
}

func (b *EmbedBuilder) Danger() *EmbedBuilder {
	b.embed.Color = b.theme.ColorDanger
	return b
}

func (b *EmbedBuilder) Success() *EmbedBuilder {
	b.embed.Color = b.theme.ColorSuccess
	return b
}

func (b *EmbedBuilder) Warning() *EmbedBuilder {
	b.embed.Color = b.theme.ColorWarning
	return b
}

func (b *EmbedBuilder) Field(name, value string, inline bool) *EmbedBuilder {
	b.embed.Fields = append(b.embed.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
	return b
}

func (b *EmbedBuilder) Thumbnail(url string) *EmbedBuilder {
	b.embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	return b
}

func (b *EmbedBuilder) Image(url string) *EmbedBuilder {
	b.embed.Image = &discordgo.MessageEmbedImage{URL: url}
	return b
}

func (b *EmbedBuilder) Footer(text string, iconURL ...string) *EmbedBuilder {
	icon := b.theme.FooterIcon
	if len(iconURL) > 0 && iconURL[0] != "" {
		icon = iconURL[0]
	}
	b.embed.Footer = &discordgo.MessageEmbedFooter{
		Text:    text,
		IconURL: icon,
	}
	return b
}

// ProgressBar incrusta un campo con una barra visual calculada
func (b *EmbedBuilder) ProgressBar(title string, current, total int, width int, style ProgressStyle) *EmbedBuilder {
	bar := RenderProgressBar(current, total, width, style)
	pct := 0
	if total > 0 {
		pct = int(math.Round(float64(current) / float64(total) * 100))
	}
	b.Field(title, fmt.Sprintf("%s **%d%%** (`%d/%d`)", bar, pct, current, total), false)
	return b
}

// LoadingStep formatea una barra para animaciones Live de procesamiento
func (b *EmbedBuilder) LoadingStep(taskName string, currentStep, totalSteps int) *EmbedBuilder {
	bar := RenderProgressBar(currentStep, totalSteps, 12, ProgressStyleModern)
	b.embed.Description = fmt.Sprintf("⏳ *%s*\n\n`%s` Pasos: %d/%d", taskName, bar, currentStep, totalSteps)
	return b
}

func (b *EmbedBuilder) Build() *discordgo.MessageEmbed {
	return b.embed
}

// ==========================================
// RENDERIZADOR DE BARRAS DE PROGRESO
// ==========================================

func RenderProgressBar(current, total int, width int, style ProgressStyle) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	if width < 5 {
		width = 10
	}

	fraction := float64(current) / float64(total)
	filledCount := int(math.Round(fraction * float64(width)))
	emptyCount := width - filledCount

	var fillRune, emptyRune string
	switch style {
	case ProgressStyleFire:
		fillRune, emptyRune = "🔥", "💨"
	case ProgressStyleDagger:
		fillRune, emptyRune = "🗡️", "▫️"
	case ProgressStyleBlock:
		fillRune, emptyRune = "■", "□"
	default: // ProgressStyleModern
		fillRune, emptyRune = "█", "░"
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < filledCount; i++ {
		sb.WriteString(fillRune)
	}
	for i := 0; i < emptyCount; i++ {
		sb.WriteString(emptyRune)
	}
	sb.WriteString("]")
	return sb.String()
}

// ==========================================
// PLANTILLAS PREDEFINIDAS DEL SISTEMA
// ==========================================

// BuildWelcomeCard crea una tarjeta de bienvenida VIP para nuevos usuarios
func BuildWelcomeCard(theme *Theme, guildName, userName, userAvatar string, memberCount int) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaLucifer).
		Title(fmt.Sprintf("🍸 ¡Bienvenido al Lux, %s!", userName)).
		Desc("Has cruzado las puertas del club más exclusivo de Los Ángeles. Aquí todos los deseos son escuchados y los pecados son bienvenidos.").
		Thumbnail(userAvatar).
		Field("👥 Eres el alma número", fmt.Sprintf("**#%d** en ingresar", memberCount), true).
		Field("📜 Normas del club", "Lee los canales fijados y respeta a la casa", true).
		Image(GifsLuciferPiano).
		Footer(fmt.Sprintf("%s • Entrada VIP", guildName)).
		Build()
}

// BuildIncidentReport crea un reporte formal de sanción o incidente para el canal de staff
func BuildIncidentReport(theme *Theme, action, offenderMention, staffMention, reason string) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaLucifer).
		Danger().
		Title(fmt.Sprintf("🔥 Castigo Aplicado: %s", strings.ToUpper(action))).
		Desc("Lucifer ha hecho cumplir las reglas del club con firmeza.").
		Field("👤 Infractor", offenderMention, true).
		Field("🛡️ Ejecutor", staffMention, true).
		Field("📝 Causa del Castigo", reason, false).
		Footer("Diario de Auditoría • Administración del Lux").
		Build()
}

// BuildLockdownAlert genera el comunicado oficial de cierre de puertas ante raids
func BuildLockdownAlert(theme *Theme, guildName string, detectedCount int, duration string) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaSystem).
		Danger().
		Title("🚨 PROTOCOLO DE DEFENSA: CLUB EN LOCKDOWN").
		Desc("Las puertas de **%s** se han sellado temporalmente para proteger a la comunidad.", guildName).
		Field("⚠️ Incursión Detectada", fmt.Sprintf("**%d usuarios** entraron en ráfaga", detectedCount), true).
		Field("⏱️ Duración del Aislamiento", duration, true).
		Field("🛡️ Acción Automática", "El protocolo del Lux está vigilando los accesos y conteniendo actividad sospechosa.", false).
		Image(GifsLuciferDevilFace).
		Footer("Alerta de Seguridad Nacional • Lux Defense").
		Build()
}

// BuildTherapyDossier arma una consulta de bienestar presentada por Lucifer
func BuildTherapyDossier(theme *Theme, patientName, question, advice string) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaLucifer).
		Success().
		Title("🕯️ Consulta de bienestar").
		Desc("Lucifer te escucha con atención y te ofrece una reflexión serena.").
		Field("Paciente", patientName, true).
		Field("Motivo de Consulta", fmt.Sprintf("> *\"%s\"*", question), false).
		Field("Reflexión", advice, false).
		Footer("Atención confidencial • El Lux").
		Build()
}

// BuildCaseFile arma un informe de caso presentado por Lucifer
func BuildCaseFile(theme *Theme, caseID, topic, evidence, verdict string) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaLucifer).
		Title(fmt.Sprintf("🕵️ Expediente del Lux #%s", caseID)).
		Desc("Caso presentado ante Lucifer:\n**%s**", topic).
		Field("🔍 Evidencia Recabada", evidence, false).
		Field("⚖️ Dictamen Final", verdict, false).
		Footer("Archivo confidencial del Lux").
		Build()
}

// BuildPianoBallad arma la tarjeta musical de Lucifer
func BuildPianoBallad(theme *Theme, songTitle, lyrics string) *discordgo.MessageEmbed {
	return NewBuilder(theme).
		Persona(PersonaLucifer).
		Title(fmt.Sprintf("🎹 «%s» — Lucifer al Piano", songTitle)).
		Desc("*(El club guarda silencio, las copas tintinean y Lucifer canta con sentimiento...)*\n\n```text\n%s\n```", lyrics).
		Image(GifsLuciferPiano).
		Footer("En vivo desde el escenario principal del Lux").
		Build()
}
