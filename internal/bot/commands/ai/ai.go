package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/ui"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(r *bot.Registry) error {
	// 1. Conversar con Lucifer (Audiencia Diplomática)
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "ask",
			Description: "Solicita una audiencia formal con Lucifer con distinción, elocuencia y franqueza ⚜️",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mensaje",
					Description: "Escriba su consulta o petición formal.",
					Required:    true,
					MaxLength:   1500,
				},
			},
		},
		LegacyHandler: c.handlePreguntar,
	}); err != nil {
		return err
	}

	// 2. Comentario Perspicaz / Roast de Salón
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "roast",
			Description: "Un brindis cordial y perspicaz con elocuente ironía aristocrática 🥂",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El distinguido destinatario del comentario",
					Required:    true,
				},
			},
		},
		LegacyHandler: c.handleRoast,
	}); err != nil {
		return err
	}

	// 3. Resumen Ejecutivo
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "resumen",
			Description: "Síntesis ejecutiva y diplomática de las deliberaciones recientes 📜",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "mensajes",
					Description: "Cantidad de intervenciones a examinar (10 a 50)",
					Required:    false,
				},
			},
		},
		LegacyHandler: c.handleResumen,
	}); err != nil {
		return err
	}

	// 4. Dictamen / Análisis de Caso
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "investigar-caso",
			Description: "Lucifer examina antecedentes y emite un dictamen imparcial ⚖️",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "detalles",
					Description: "Los hechos o elementos sometidos a análisis",
					Required:    true,
				},
			},
		},
		LegacyHandler: c.handleInvestigarCaso,
	}); err != nil {
		return err
	}

	// 5. Composiciones Magistrales
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "piano",
			Description: "Lucifer compone versos nobles y solemnes dedicados a su temática 🎹",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tema",
					Description: "Temática de la composición (ej. la gloria, el tiempo, la lealtad)",
					Required:    true,
				},
			},
		},
		LegacyHandler: c.handlePiano,
	}); err != nil {
		return err
	}

	// 6. Consejo de Cancillería
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "consejo",
			Description: "Plantea un dilema y recibe una orientación estratégica y pragmática 👑",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "dilema",
					Description: "¿Qué encrucijada requiere análisis?",
					Required:    true,
				},
			},
		},
		LegacyHandler: c.handleConsejo,
	}); err != nil {
		return err
	}

	// 7. Traducción Diplomática
	if err := r.AddCommand(&bot.Command{
		Category: "ia",
		AppCmd: &discordgo.ApplicationCommand{
			Name:        "traducir",
			Description: "Traducción de alto nivel diplomático preservando tono y elocuencia 🌐",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "texto",
					Description: "Texto o misiva a traducir",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "idioma",
					Description: "Idioma de destino (ej. inglés, francés, italiano, japonés)",
					Required:    true,
				},
			},
		},
		LegacyHandler: c.handleTraducir,
	}); err != nil {
		return err
	}

	return nil
}

// ==========================================
// HANDLERS
// ==========================================

func (c *Cog) handlePreguntar(ctx *bot.Context) {
	ctx.PlainText = true
	opts := ctx.Interaction.ApplicationCommandData().Options
	pregunta := opts[0].StringValue()

	if ctx.Env.AI == nil {
		// Plain text error (NO EMBED) as requested
		_ = ctx.Respond(&discordgo.InteractionResponseData{
			Content: ui.FormatErrorMessage("Servicio Diplomático No Disponible", "La conexión con el oráculo de cancillería no se encuentra en línea en este instante. Por favor, reintente en unos minutos."),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	mode, err := aiResponseMode(ctx)
	if err != nil {
		_ = ctx.Respond(&discordgo.InteractionResponseData{
			Content: ui.FormatErrorMessage("Ajustes de Privacidad", "No fue posible verificar sus preferencias de entrega. Intente nuevamente."),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	loadingLabel := "Lucifer está deliberando sobre su consulta con estricto rigor diplomático..."
	systemPrompt := "Eres Lucifer, un regente aristocrático, distinguido, carismático y de elocuencia diplomática intachable. Eres perspicaz, refinado y directo, sin ser vulgar ni cruel. Responde en 2 a 4 oraciones con alta distinción en español formal pero seductor. No hagas referencias a series de televisión ni a locales nocturnos."

	// Initial Live message (NO EMBED)
	if err := ctx.Respond(&discordgo.InteractionResponseData{
		Content: liveLoadingMessage(loadingLabel, 0),
		Flags:   responseFlags(mode),
	}); err != nil {
		ctx.Env.Log.Error("no se pudo iniciar /ask", "err", err)
		return
	}

	go func() {
		type result struct {
			answer string
			err    error
		}
		results := make(chan result, 1)
		go func() {
			answer, err := callAI(ctx, systemPrompt, pregunta)
			results <- result{answer: answer, err: err}
		}()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		elapsed := 0
		for {
			select {
			case <-ticker.C:
				elapsed++
				_ = ctx.EditResponse(&discordgo.WebhookEdit{
					Content: stringPtr(liveLoadingMessage(loadingLabel, elapsed)),
					Embeds:  &[]*discordgo.MessageEmbed{},
				})
			case res := <-results:
				if res.err != nil {
					_ = ctx.EditResponse(&discordgo.WebhookEdit{
						Content: stringPtr(ui.FormatErrorMessage("Audiencia Interrumpida", "La deliberación no pudo ser completada. Por favor, presente su consulta nuevamente.")),
						Embeds:  &[]*discordgo.MessageEmbed{},
					})
					return
				}

				// Final response: Clean, elegant, charismatic plain text (NO EMBED)
				finalMessage := formatAskResponse(ctx.Interaction.Member.User.Username, pregunta, res.answer)
				if err := deliverAIContent(ctx, finalMessage, mode); err != nil {
					ctx.Env.Log.Error("error entregando respuesta de /ask", "err", err)
				}
				return
			}
		}
	}()
}

func liveLoadingMessage(label string, elapsed int) string {
	const totalBars = 12
	filled := (elapsed * 2) % (totalBars + 1)
	bar := strings.Repeat("▰", filled) + strings.Repeat("▱", totalBars-filled)
	label = strings.ReplaceAll(label, "@", "@\u200b")
	return fmt.Sprintf("⏳ **%s**\n`[%s]` ✨ *Sincronizando protocolo real (%d s)*", label, bar, elapsed)
}

func formatAskResponse(username, question, answer string) string {
	question = strings.ReplaceAll(safeAIText(question), "\n", "\n> ")
	answer = strings.ReplaceAll(strings.TrimSpace(answer), "@", "@\u200b")

	return fmt.Sprintf("### ⚜️ Audiencia Oficial con Lucifer\n\n> **Consulta presentada por @%s:**\n> %s\n\n**Dictamen de Lucifer:**\n%s\n\n*Transmitido con distinción protocolar y cortesía absoluta. 🥂*", safeAIText(username), question, answer)
}

func (c *Cog) handleRoast(ctx *bot.Context) {
	ctx.PlainText = true
	target := ctx.Interaction.ApplicationCommandData().Options[0].UserValue(ctx.Session)
	runAITextWithProgress(ctx, fmt.Sprintf("Lucifer prepara un brindis perspicaz para @%s...", target.Username), "No fue posible formular el comentario.", func() (string, string, error) {
		systemPrompt := "Eres Lucifer: aristocrático, distinguido y mordaz con ingenio soberano. Escribe una chanza sutil y elegante de no más de 2 oraciones dirigida a la persona indicada. Debe ser refinada, inteligente y sin vulgaridades ni alusiones a series."
		prompt := fmt.Sprintf("Formule un comentario ingenioso, aristocrático y cordial para el invitado '%s'.", target.Username)
		resp, err := callAI(ctx, systemPrompt, prompt)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### 🥂 Brindis de Honor con Lucifer\n\n**Dedicado con distinción a @%s:**\n> %s\n\n*Servido a la temperatura exacta, con una sonrisa impecable y la más alta consideración.*", safeAIText(target.Username), strings.TrimSpace(resp))
		return "Brindis", body, nil
	})
}

func (c *Cog) handleResumen(ctx *bot.Context) {
	ctx.PlainText = true
	limit := 25
	if len(ctx.Interaction.ApplicationCommandData().Options) > 0 {
		limit = int(ctx.Interaction.ApplicationCommandData().Options[0].IntValue())
		if limit < 10 {
			limit = 10
		} else if limit > 50 {
			limit = 50
		}
	}

	runAITextWithProgress(ctx, fmt.Sprintf("Lucifer examina las últimas %d intervenciones del senado...", limit), "No fue posible recopilar la deliberación.", func() (string, string, error) {
		messages, err := ctx.Session.ChannelMessages(ctx.Interaction.ChannelID, limit, "", "", "")
		if err != nil || len(messages) == 0 {
			return "", "", fmt.Errorf("insuficientes mensajes para analizar")
		}

		var b strings.Builder
		for i := len(messages) - 1; i >= 0; i-- {
			m := messages[i]
			if !m.Author.Bot && m.Content != "" {
				b.WriteString(fmt.Sprintf("%s: %s\n", m.Author.Username, m.Content))
			}
		}

		systemPrompt := "Redacta un informe ejecutivo conciso de 3 puntos sobre el debate sostenido. Sé diplomático, elocuente y preciso."
		summary, err := callAI(ctx, systemPrompt, b.String())
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### 📜 Informe Ejecutivo de Cancillería\n\n**Intervenciones analizadas:** %d registros oficiales\n\n%s\n\n*Certificado bajo el sello de Lucifer.*", len(messages), summary)
		return "Resumen", body, nil
	})
}

func (c *Cog) handleInvestigarCaso(ctx *bot.Context) {
	ctx.PlainText = true
	caso := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	runAITextWithProgress(ctx, "Lucifer examina las pruebas y testimonios...", "No se pudo concluir el análisis.", func() (string, string, error) {
		systemPrompt := "Analiza los hechos con lógica rigurosa y templanza diplomática. Emite una conclusión de máximo 3 oraciones distinguiendo hechos de rumores."
		analisis, err := callAI(ctx, systemPrompt, caso)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### ⚖️ Dictamen Imparcial de Lucifer\n\n> **Hechos expuestos:**\n> %s\n\n**Resolución protocolar:**\n%s", safeAIText(caso), analisis)
		return "Caso", body, nil
	})
}

func (c *Cog) handlePiano(ctx *bot.Context) {
	ctx.PlainText = true
	tema := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	runAITextWithProgress(ctx, "Lucifer afina el gran piano de cola...", "No fue posible interpretar la melodía.", func() (string, string, error) {
		systemPrompt := "Compón una pieza poética solemne, majestuosa y de refinada métrica sobre el tema propuesto. Entre 4 y 6 versos rimados con una reflexión final de alta dignidad."
		poema, err := callAI(ctx, systemPrompt, tema)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### 🎹 Composición Solemne al Piano\n\n*La estancia guarda silencio. Lucifer acaricia las teclas de ébano y marfil…*\n\n%s\n\n*— Inspirada solemnemente en: «%s»*", poema, safeAIText(tema))
		return "Piano", body, nil
	})
}

func (c *Cog) handleConsejo(ctx *bot.Context) {
	ctx.PlainText = true
	dilema := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	runAITextWithProgress(ctx, "Lucifer sopesa los equilibrios de poder...", "El consejo quedó diferido.", func() (string, string, error) {
		prompt := fmt.Sprintf("Dilema planteado: \"%s\". Brinda una orientación estratégica, perspicaz y libre de ingenuidad en 2 a 3 oraciones.", dilema)
		advice, err := callAI(ctx, "Eres Lucifer, regente de alto rango y refinado consejero diplomático. Sé pragmático, respetuoso y brillante.", prompt)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### 👑 Consejo de Cancillería\n\n> **Encrucijada planteada:**\n> %s\n\n**Orientación:**\n%s", safeAIText(dilema), advice)
		return "Consejo", body, nil
	})
}

func (c *Cog) handleTraducir(ctx *bot.Context) {
	ctx.PlainText = true
	opts := ctx.Interaction.ApplicationCommandData().Options
	texto := opts[0].StringValue()
	idioma := opts[1].StringValue()

	runAITextWithProgress(ctx, "Lucifer redacta la misiva en los registros diplomáticos...", "No fue posible traducir el texto.", func() (string, string, error) {
		systemPrompt := fmt.Sprintf("Traduce el texto al idioma: %s. Emplea un lenguaje culto, natural y distinguido. Devuelve ÚNICAMENTE la traducción, sin comillas adicionales.", idioma)
		trad, err := callAI(ctx, systemPrompt, texto)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("### 🌐 Despacho Diplomático • Traducción al %s\n\n> **Texto original:**\n> %s\n\n**Versión diplomática:**\n%s", strings.Title(idioma), safeAIText(texto), trad)
		return "Traductor", body, nil
	})
}

// ==========================================
// PUENTE Y GENERACIÓN
// ==========================================

func callAI(ctx *bot.Context, systemPrompt, userMessage string) (string, error) {
	aiCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	resp, err := ctx.Env.AI.AskSimple(aiCtx, systemPrompt, userMessage)
	if err != nil {
		return "", err
	}

	resp = strings.TrimSpace(resp)
	if resp == "" {
		return "", fmt.Errorf("empty response received")
	}
	return resp, nil
}

func runAITextWithProgress(ctx *bot.Context, loadingText, failureText string, generate func() (string, string, error)) {
	ctx.PlainText = true
	mode, err := aiResponseMode(ctx)
	if err != nil {
		_ = ctx.Respond(&discordgo.InteractionResponseData{
			Content: ui.FormatErrorMessage("Ajustes de Privacidad", "No se pudo verificar la modalidad de respuesta."),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	if err := ctx.Respond(&discordgo.InteractionResponseData{
		Content: liveLoadingMessage(loadingText, 0),
		Flags:   responseFlags(mode),
	}); err != nil {
		ctx.Env.Log.Error("no se pudo iniciar respuesta", "err", err)
		return
	}

	go func() {
		type result struct {
			title string
			body  string
			err   error
		}
		results := make(chan result, 1)
		go func() {
			title, body, err := generate()
			results <- result{title: title, body: body, err: err}
		}()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		elapsed := 0
		for {
			select {
			case <-ticker.C:
				elapsed++
				_ = ctx.EditResponse(&discordgo.WebhookEdit{
					Content: stringPtr(liveLoadingMessage(loadingText, elapsed)),
					Embeds:  &[]*discordgo.MessageEmbed{},
				})
			case res := <-results:
				if res.err != nil {
					_ = ctx.EditResponse(&discordgo.WebhookEdit{
						Content: stringPtr(ui.FormatErrorMessage("Solicitud No Concluida", failureText)),
						Embeds:  &[]*discordgo.MessageEmbed{},
					})
					return
				}

				if err := deliverAIContent(ctx, res.body, mode); err != nil {
					ctx.Env.Log.Error("no se pudo entregar contenido", "err", err)
				}
				return
			}
		}
	}()
}

func aiResponseMode(ctx *bot.Context) (string, error) {
	if ctx.Interaction.GuildID == "" || ctx.Interaction.Member == nil || ctx.Interaction.Member.User == nil {
		return "private", nil
	}
	preferences, err := ctx.Env.Store.GetActivityPreferences(ctx.Context, ctx.Interaction.GuildID, ctx.Interaction.Member.User.ID)
	if err != nil {
		return "", err
	}
	return preferences.ResponseMode, nil
}

func responseFlags(mode string) discordgo.MessageFlags {
	if mode == "public" {
		return 0
	}
	return discordgo.MessageFlagsEphemeral
}

func deliverAIContent(ctx *bot.Context, content, mode string) error {
	if mode == "dm" {
		userID := ctx.Interaction.Member.User.ID
		channel, err := ctx.Session.UserChannelCreate(userID)
		if err != nil {
			return errors.Join(fmt.Errorf("abrir dm: %w", err), errors.New("no se pudo abrir el canal directo"))
		}
		if _, err := ctx.Session.ChannelMessageSend(channel.ID, content); err != nil {
			return errors.Join(fmt.Errorf("enviar dm: %w", err), errors.New("no se pudo enviar el mensaje directo"))
		}
		content = "📩 **Despacho enviado:** Lucifer le ha remitido la respuesta formal a su canal de correspondencia privada."
	}
	return ctx.EditResponse(&discordgo.WebhookEdit{
		Content: stringPtr(content),
		Embeds:  &[]*discordgo.MessageEmbed{},
	})
}

func safeAIText(value string) string {
	value = strings.ReplaceAll(value, "@", "@\u200b")
	value = strings.ReplaceAll(value, "`", "ˋ")
	return value
}

func stringPtr(s string) *string {
	return &s
}
