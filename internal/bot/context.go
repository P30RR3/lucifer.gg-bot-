package bot

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/ui"
)

type Context struct {
	Context     context.Context
	Session     *discordgo.Session
	Interaction *discordgo.InteractionCreate
	Env         *Env
	deferred    bool
	PlainText   bool
}

func (c *Context) Defer(ephemeral bool) error {
	data := &discordgo.InteractionResponseData{}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := discordutil.InteractionRespond(c.Session, c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: data,
	}); err != nil {
		return err
	}
	c.deferred = true
	return nil
}

func (c *Context) Respond(data *discordgo.InteractionResponseData) error {
	if c.isConfigurationCommand() {
		c.embedConfigurationStatus(data)
	}
	if data.Content != "" && !c.PlainText {
		data.Content = ui.FrameMessage(data.Content)
	}
	if c.Env != nil && c.Env.Theme != nil {
		for _, embed := range data.Embeds {
			c.Env.Theme.BrandEmbed(embed)
		}
	}
	if c.deferred {
		edit := &discordgo.WebhookEdit{}
		if data.Content != "" {
			edit.Content = &data.Content
		}
		if data.Embeds != nil {
			embeds := data.Embeds
			edit.Embeds = &embeds
		}
		if data.Components != nil {
			components := data.Components
			edit.Components = &components
		}
		return c.EditResponse(edit)
	}
	return discordutil.InteractionRespond(c.Session, c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func (c *Context) EditResponse(edit *discordgo.WebhookEdit) error {
	if c.isConfigurationCommand() {
		c.embedConfigurationEdit(edit)
	}
	if edit.Content != nil && !c.PlainText {
		content := ui.FrameMessage(*edit.Content)
		edit.Content = &content
	}
	if c.Env != nil && c.Env.Theme != nil && edit.Embeds != nil {
		for _, embed := range *edit.Embeds {
			c.Env.Theme.BrandEmbed(embed)
		}
	}
	_, err := discordutil.InteractionResponseEdit(c.Session, c.Interaction.Interaction, edit)
	return err
}

func (c *Context) isConfigurationCommand() bool {
	return c.Interaction != nil &&
		c.Interaction.Type == discordgo.InteractionApplicationCommand &&
		c.Interaction.ApplicationCommandData().Name == "config"
}

func (c *Context) embedConfigurationStatus(data *discordgo.InteractionResponseData) {
	if data == nil || strings.TrimSpace(data.Content) == "" {
		return
	}
	message := strings.TrimSpace(data.Content)
	data.Content = ""
	if len(data.Embeds) > 0 && data.Embeds[0] != nil {
		embed := *data.Embeds[0]
		embed.Fields = append(append([]*discordgo.MessageEmbedField(nil), embed.Fields...), &discordgo.MessageEmbedField{
			Name: "Estado", Value: message,
		})
		data.Embeds = append([]*discordgo.MessageEmbed(nil), data.Embeds...)
		data.Embeds[0] = &embed
		return
	}
	data.Embeds = []*discordgo.MessageEmbed{c.configurationStatusEmbed(message)}
}

func (c *Context) embedConfigurationEdit(edit *discordgo.WebhookEdit) {
	if edit == nil || edit.Content == nil || strings.TrimSpace(*edit.Content) == "" {
		return
	}
	message := strings.TrimSpace(*edit.Content)
	empty := ""
	edit.Content = &empty
	if edit.Embeds != nil && len(*edit.Embeds) > 0 && (*edit.Embeds)[0] != nil {
		embeds := append([]*discordgo.MessageEmbed(nil), (*edit.Embeds)...)
		embed := *embeds[0]
		embed.Fields = append(append([]*discordgo.MessageEmbedField(nil), embed.Fields...), &discordgo.MessageEmbedField{
			Name: "Estado", Value: message,
		})
		embeds[0] = &embed
		edit.Embeds = &embeds
		return
	}
	edit.Embeds = &[]*discordgo.MessageEmbed{c.configurationStatusEmbed(message)}
}

func (c *Context) configurationStatusEmbed(message string) *discordgo.MessageEmbed {
	lower := strings.ToLower(message)
	if strings.Contains(message, "❌") || strings.Contains(lower, "no pude") ||
		strings.Contains(lower, "no tengo permisos") || strings.Contains(lower, "error") {
		return c.Env.Theme.ErrorEmbed("Configuración", message)
	}
	if strings.Contains(message, "✅") || strings.Contains(lower, "actualizado") ||
		strings.Contains(lower, "configurado") || strings.Contains(lower, "listos") {
		return c.Env.Theme.SuccessEmbed("Configuración", message)
	}
	return c.Env.Theme.WarningEmbed("Configuración", message)
}
