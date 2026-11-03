package discordutil

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestApplyResponseVisibilityDefaultsToPublicAndKeepsErrorsPrivate(t *testing.T) {
	interaction := &discordgo.Interaction{GuildID: "guild-public", Member: &discordgo.Member{User: &discordgo.User{ID: "user-public"}}}
	data := &discordgo.InteractionResponseData{Content: "Operación completada.", Flags: discordgo.MessageFlagsEphemeral}

	applyResponseVisibility(interaction, discordgo.InteractionResponseChannelMessageWithSource, data)
	if data.Flags&discordgo.MessageFlagsEphemeral != 0 {
		t.Fatal("successful response should be public by default")
	}

	errorData := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{{Title: "Error", Color: 0x6A0712}},
	}
	applyResponseVisibility(interaction, discordgo.InteractionResponseChannelMessageWithSource, errorData)
	if errorData.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatal("error response should remain private")
	}

	textError := &discordgo.InteractionResponseData{
		Content: "No tienes permisos para completar esta acción.",
	}
	applyResponseVisibility(interaction, discordgo.InteractionResponseChannelMessageWithSource, textError)
	if textError.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatal("plain-text error should remain private")
	}
}

func TestApplyResponseVisibilityRespectsPrivatePreference(t *testing.T) {
	const guildID, userID = "guild-private", "user-private"
	SetResponseVisibility(guildID, userID, false)
	interaction := &discordgo.Interaction{GuildID: guildID, Member: &discordgo.Member{User: &discordgo.User{ID: userID}}}
	data := &discordgo.InteractionResponseData{Content: "Operación completada."}

	applyResponseVisibility(interaction, discordgo.InteractionResponseChannelMessageWithSource, data)
	if data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatal("response should be private after opting out of public visibility")
	}
}
