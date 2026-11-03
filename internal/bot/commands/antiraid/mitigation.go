package antiraid

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Action string

const (
	ActionNone     Action = "none"
	ActionWarn     Action = "warn"
	ActionTimeout  Action = "timeout"
	ActionKick     Action = "kick"
	ActionBan      Action = "ban"
	ActionLockdown Action = "lockdown"
)

func ParseAction(s string) Action {
	switch s {
	case "warn":
		return ActionWarn
	case "timeout":
		return ActionTimeout
	case "kick":
		return ActionKick
	case "ban":
		return ActionBan
	case "lockdown":
		return ActionLockdown
	default:
		return ActionNone
	}
}

type Decision struct {
	Action      Action
	Duration    time.Duration
	Reason      string
	Detector    string
	Score       int
	DeleteMsgID string
	ChannelID   string
	NotifyUser  bool
}

type Mitigator struct {
	session  *discordgo.Session
	log      *slog.Logger
	lockdown *LockdownManager
}

func NewMitigator(s *discordgo.Session, log *slog.Logger, lockdown *LockdownManager) *Mitigator {
	if log == nil {
		log = slog.Default()
	}
	return &Mitigator{
		session:  s,
		log:      log.With("component", "mitigator"),
		lockdown: lockdown,
	}
}

var (
	ErrUnknownAction = errors.New("mitigation: unknown action")
	ErrActionFailed  = errors.New("mitigation: action failed")
)

func (m *Mitigator) Apply(ctx context.Context, guildID, userID string, d Decision) error {
	switch d.Action {
	case ActionNone:
		return nil

	case ActionWarn:
		if d.DeleteMsgID != "" && d.ChannelID != "" {
			_ = m.session.ChannelMessageDelete(d.ChannelID, d.DeleteMsgID)
		}
		return nil

	case ActionTimeout:
		dur := d.Duration
		if dur <= 0 {
			dur = 10 * time.Minute
		}
		until := time.Now().Add(dur)
		if err := m.session.GuildMemberTimeout(guildID, userID, &until); err != nil {
			return fmt.Errorf("%w: timeout: %v", ErrActionFailed, err)
		}
		if d.DeleteMsgID != "" && d.ChannelID != "" {
			_ = m.session.ChannelMessageDelete(d.ChannelID, d.DeleteMsgID)
		}
		return nil

	case ActionKick:
		if err := m.session.GuildMemberDeleteWithReason(guildID, userID, d.Reason); err != nil {
			return fmt.Errorf("%w: kick: %v", ErrActionFailed, err)
		}
		return nil

	case ActionBan:
		if err := m.session.GuildBanCreateWithReason(guildID, userID, d.Reason, 1); err != nil {
			return fmt.Errorf("%w: ban: %v", ErrActionFailed, err)
		}
		return nil

	case ActionLockdown:
		if m.lockdown == nil {
			return fmt.Errorf("%w: lockdown no configurado", ErrActionFailed)
		}
		dur := d.Duration
		if dur <= 0 {
			dur = 10 * time.Minute
		}
		return m.lockdown.Activate(ctx, guildID, dur)

	default:
		return fmt.Errorf("%w: %q", ErrUnknownAction, d.Action)
	}
}

func (m *Mitigator) ApplyBatch(ctx context.Context, guildID string, userIDs []string, d Decision) (int, int) {
	success, failed := 0, 0
	for _, uid := range userIDs {
		if err := m.Apply(ctx, guildID, uid, d); err != nil {
			failed++
			continue
		}
		success++
	}
	return success, failed
}

func (m *Mitigator) Timeout(guildID, userID string, dur time.Duration) error {
	until := time.Now().Add(dur)
	return m.session.GuildMemberTimeout(guildID, userID, &until)
}

func (m *Mitigator) RemoveTimeout(guildID, userID string) error {
	return m.session.GuildMemberTimeout(guildID, userID, nil)
}

func (m *Mitigator) Kick(guildID, userID, reason string) error {
	return m.session.GuildMemberDeleteWithReason(guildID, userID, reason)
}

func (m *Mitigator) Ban(guildID, userID, reason string) error {
	return m.session.GuildBanCreateWithReason(guildID, userID, reason, 1)
}

func (m *Mitigator) Unban(guildID, userID string) error {
	return m.session.GuildBanDelete(guildID, userID)
}

func (m *Mitigator) DeleteMessage(channelID, messageID string) error {
	return m.session.ChannelMessageDelete(channelID, messageID)
}

func (m *Mitigator) PurgeMessages(channelID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) == 1 {
		if err := m.session.ChannelMessageDelete(channelID, ids[0]); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err := m.session.ChannelMessagesBulkDelete(channelID, ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}
