package antiraid

import (
	"testing"
	"time"
)

func TestExpiredLockdownKeepsPermissionSnapshotsForRestoration(t *testing.T) {
	manager := NewLockdownManager(nil, time.Minute, nil)
	manager.active["guild"] = time.Now().Add(-time.Second)
	manager.affected["guild"] = map[string]permissionSnapshot{
		"channel": {existed: true, allow: 4, deny: 8},
	}

	if active, _ := manager.IsActive("guild"); active {
		t.Fatal("expired lockdown should not be reported as active")
	}
	info := manager.Info("guild")
	if info.Active || len(info.Channels) != 1 || info.Channels[0] != "channel" {
		t.Fatalf("expired lockdown snapshot was lost: %#v", info)
	}
}
