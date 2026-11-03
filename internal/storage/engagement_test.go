package storage

import (
	"reflect"
	"testing"
)

func TestExtractActivityWordsCountsWordsWithoutCapturingIdentifiers(t *testing.T) {
	got := extractActivityWords("Hola, hola Discord. @everyone visita https://example.com/a texto@correo.com 123456789012345678")
	want := map[string]int64{
		"hola":    2,
		"discord": 1,
		"visita":  1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractActivityWords() = %#v, want %#v", got, want)
	}
}

func TestCelestialMissionsHaveDailyAndWeeklyTargets(t *testing.T) {
	if len(celestialMissions) < 3 {
		t.Fatalf("expected daily and weekly missions, got %d", len(celestialMissions))
	}
	periods := map[string]bool{}
	for _, mission := range celestialMissions {
		periods[mission.period] = true
		if mission.id == "" || mission.target <= 0 || mission.points <= 0 {
			t.Errorf("mission has invalid configuration: %#v", mission)
		}
	}
	if !periods["day"] || !periods["week"] {
		t.Fatalf("expected both daily and weekly missions, got %#v", periods)
	}
}
