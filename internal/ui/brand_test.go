package ui

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestBrandMessageIsConsistentAndIdempotent(t *testing.T) {
	first := BrandMessage("El Lux está listo.")
	second := BrandMessage(first)
	if !strings.Contains(first, "LA DOLCE VITA") || !strings.Contains(first, "Con eleganza") {
		t.Fatalf("message is missing the shared Lux style: %q", first)
	}
	if second != first {
		t.Fatalf("branding should be idempotent:\nfirst: %q\nsecond: %q", first, second)
	}
}

func TestBrandEmbedAndComponentsShareLuxIdentity(t *testing.T) {
	embed := &discordgo.MessageEmbed{Title: "Estado"}
	BrandEmbed(embed)
	if embed.Author == nil || !strings.Contains(embed.Author.Name, "Il Lux") {
		t.Fatalf("embed author is missing the Lux identity: %#v", embed.Author)
	}

	if embed.Footer == nil || !strings.Contains(embed.Footer.Text, "eleganza italiana") {
		t.Fatalf("embed footer is missing the Italian signature: %#v", embed.Footer)
	}
	footer := embed.Footer.Text
	BrandEmbed(embed)
	if embed.Footer.Text != footer {
		t.Fatalf("embed branding should be idempotent, got %q then %q", footer, embed.Footer.Text)
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Confirmar", CustomID: "confirm"},
		}},
	}
	BrandComponents(components)
	row := components[0].(discordgo.ActionsRow)
	button := row.Components[0].(discordgo.Button)
	if !strings.HasPrefix(button.Label, "✦ ") {
		t.Fatalf("button label is missing the shared style: %q", button.Label)
	}

	form := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{Label: "Motivo", CustomID: "reason"},
		}},
	}
	BrandComponents(form)
	input := form[0].(discordgo.ActionsRow).Components[0].(discordgo.TextInput)
	if !strings.HasPrefix(input.Label, "✦ ") {
		t.Fatalf("form label is missing the shared style: %q", input.Label)
	}
}

func TestBrandEmbedReplacesBrightColorsWithDarkLuxRed(t *testing.T) {
	embed := &discordgo.MessageEmbed{Title: "Estado", Color: 0x2ECC71}
	BrandEmbed(embed)
	if embed.Color != defaultBrand.Primary {
		t.Fatalf("bright embed color should use the dark Lux palette: got %#06x, want %#06x", embed.Color, defaultBrand.Primary)
	}
}
