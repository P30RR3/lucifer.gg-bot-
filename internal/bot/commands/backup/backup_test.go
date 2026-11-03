package backup

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/storage"
)

func TestValidateSnapshotRejectsMissingCategory(t *testing.T) {
	snapshot := guildSnapshot{
		Version: snapshotVersion,
		Channels: []channelSnapshot{{
			ID: "text", Name: "general", Type: discordgo.ChannelTypeGuildText, ParentID: "missing",
		}},
	}
	if err := validateSnapshot(snapshot); err == nil {
		t.Fatal("expected a missing category reference to be rejected")
	}
}

func TestValidateSnapshotAcceptsCategoryAndTextChannel(t *testing.T) {
	snapshot := guildSnapshot{
		Version: snapshotVersion,
		Channels: []channelSnapshot{
			{ID: "category", Name: "Lounge", Type: discordgo.ChannelTypeGuildCategory},
			{ID: "text", Name: "general", Type: discordgo.ChannelTypeGuildText, ParentID: "category"},
		},
	}
	if err := validateSnapshot(snapshot); err != nil {
		t.Fatalf("valid snapshot rejected: %v", err)
	}
}

func TestRemapOverwritesChangesOnlyBackedUpRoleIDs(t *testing.T) {
	input := []*discordgo.PermissionOverwrite{
		{ID: "old-role", Type: discordgo.PermissionOverwriteTypeRole, Allow: 1},
		{ID: "user", Type: discordgo.PermissionOverwriteTypeMember, Deny: 2},
	}
	got := remapOverwrites(input, map[string]string{"old-role": "new-role"})
	if got[0].ID != "new-role" {
		t.Fatalf("role overwrite ID = %q, want new-role", got[0].ID)
	}
	if got[1].ID != "user" {
		t.Fatalf("member overwrite ID = %q, want user", got[1].ID)
	}
	if input[0].ID != "old-role" {
		t.Fatal("remapping must not mutate the source snapshot")
	}
}

func TestSupportedChannelTypeExcludesThreads(t *testing.T) {
	if supportedChannelType(discordgo.ChannelTypeGuildPublicThread) {
		t.Fatal("threads cannot be recreated from a structural backup")
	}
	if !supportedChannelType(discordgo.ChannelTypeGuildForum) {
		t.Fatal("forum channels should be supported")
	}
}

func TestRemapBotChannelSettingsClearsAndReplacesReferences(t *testing.T) {
	settings := remapBotChannelSettings(
		metadataChannelSettings(),
		map[string]string{"old-welcome": "new-welcome", "old-help": "new-help"},
	)
	if settings.WelcomeChannel != "new-welcome" || settings.HelpChannelID != "new-help" {
		t.Fatalf("mapped settings = %+v", settings)
	}
	if settings.LogChannelID != "" || settings.ConfessionChannelID != "" {
		t.Fatalf("missing channels should be cleared, got %+v", settings)
	}
}

func TestValidateSnapshotRejectsMissingServerChannel(t *testing.T) {
	snapshot := guildSnapshot{
		Version: snapshotVersion,
		ServerChannels: serverChannelSettings{
			SystemChannelID: "missing",
		},
	}
	if err := validateSnapshot(snapshot); err == nil {
		t.Fatal("expected a missing server channel reference to be rejected")
	}
}

func metadataChannelSettings() storage.GuildBackupChannelSettings {
	return storage.GuildBackupChannelSettings{
		LogChannelID: "missing-log", WelcomeChannel: "old-welcome",
		HelpChannelID: "old-help", ConfessionChannelID: "missing-confession",
	}
}
