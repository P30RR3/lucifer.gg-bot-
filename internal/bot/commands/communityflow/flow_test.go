package communityflow

import (
	"testing"

	"krypton.dev/internal/bot"
)

func TestCogRegistersCommunityFlows(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register community flows: %v", err)
	}
	for _, name := range []string{"ayuda", "demanda", "deseo-profundo", "chantaje"} {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("command %q was not registered", name)
		}
	}
	if _, ok := registry.Get("ayuda-config"); ok {
		t.Error("ayuda-config should be replaced by automatic channel creation")
	}
	if _, ok := registry.Get("confesiones-config"); ok {
		t.Error("confesiones-config should be replaced by automatic channel creation")
	}
}

func TestParseTrialVerdictFailsClosed(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: `{"verdict":"culpable","reason":"El juego lo decidió."}`, want: "culpable"},
		{raw: `{"verdict":"inocente","reason":"No hay pruebas."}`, want: "inocente"},
		{raw: `{"verdict":"culpable","reason":"x"} texto extra`, want: "culpable"},
		{raw: "No hubo una respuesta válida.", want: "inocente"},
	}

	for _, test := range tests {
		got, _ := parseTrialVerdict(test.raw)
		if got != test.want {
			t.Errorf("parseTrialVerdict(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
}

func TestDeepDesireModalLabelFitsDiscordLimit(t *testing.T) {
	if length := len([]rune(deepDesireInputLabel)); length < 1 || length > 45 {
		t.Fatalf("deep desire modal label length = %d, want 1..45", length)
	}
}
