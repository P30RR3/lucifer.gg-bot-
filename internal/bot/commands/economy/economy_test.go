package economy

import (
	"testing"

	"krypton.dev/internal/bot"
)

func TestCogRegistersCommands(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register economy cog: %v", err)
	}
	if got, want := registry.CategoryCount("economia"), 6; got != want {
		t.Fatalf("registered economy commands = %d, want %d", got, want)
	}
	if _, ok := registry.Get("tienda-admin"); !ok {
		t.Fatal("shop administration command was not registered")
	}
}

func TestRouletteColor(t *testing.T) {
	tests := []struct {
		number int
		color  string
	}{
		{number: 0, color: "verde"},
		{number: 1, color: "rojo"},
		{number: 2, color: "negro"},
		{number: 14, color: "rojo"},
		{number: 36, color: "rojo"},
	}

	for _, test := range tests {
		if got := rouletteColor(test.number); got != test.color {
			t.Errorf("rouletteColor(%d) = %q, want %q", test.number, got, test.color)
		}
	}
}

func TestRoulettePayout(t *testing.T) {
	tests := []struct {
		name        string
		bet         int64
		choice      string
		resultColor string
		want        int64
	}{
		{name: "red win", bet: 10, choice: "rojo", resultColor: "rojo", want: 20},
		{name: "black loss", bet: 10, choice: "negro", resultColor: "rojo", want: 0},
		{name: "green win", bet: 10, choice: "verde", resultColor: "verde", want: 360},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := roulettePayout(test.bet, test.choice, test.resultColor); got != test.want {
				t.Errorf("roulettePayout() = %d, want %d", got, test.want)
			}
		})
	}
}
