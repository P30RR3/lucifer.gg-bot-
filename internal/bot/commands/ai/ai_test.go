package ai

import (
	"strings"
	"testing"
)

func TestLuxVoicePromptStylesGenerationsWithoutChangingTranslations(t *testing.T) {
	styled := luxVoicePrompt("Resume con claridad.")
	if !strings.Contains(styled, "elegancia de inspiración italiana") {
		t.Fatalf("generated response prompt is missing the shared voice: %q", styled)
	}
	if unchanged := luxVoicePrompt("Devuelve ÚNICAMENTE la traducción."); unchanged != "Devuelve ÚNICAMENTE la traducción." {
		t.Fatalf("translation prompt must remain unchanged, got %q", unchanged)
	}
}

func TestAskResponseUsesMarkdownAndEscapesMentions(t *testing.T) {
	got := formatAskMessage("🍸 Lucifer", "¿Quién eres?", "@everyone\nUna respuesta **elegante**.")
	for _, expected := range []string{"### ✦ 🍸 Lucifer", "**Te escuché decir:**", "> ¿Quién eres?", "@\u200beveryone", "**elegante**"} {
		if !strings.Contains(got, expected) {
			t.Errorf("formatAskMessage() is missing %q in %q", expected, got)
		}
	}
}

func TestAskLoadingProgressAdvancesEverySecond(t *testing.T) {
	first := askLoadingMessage("Lucifer preparando la respuesta", 1)
	next := askLoadingMessage("Lucifer preparando la respuesta", 2)
	if first == next || !strings.Contains(first, "· 1 s") || !strings.Contains(next, "· 2 s") {
		t.Fatalf("progress should visibly advance each second; got %q and %q", first, next)
	}
}

func TestAITextFormattingPreservesMarkdownAndDisablesMentions(t *testing.T) {
	got := formatAIText("🔥 El veredicto", "**Conclusión:** @everyone\nUna respuesta elegante.")
	if !strings.Contains(got, "**Conclusión:**") || !strings.Contains(got, "@\u200beveryone") {
		t.Fatalf("AI text should retain Markdown while preventing mentions, got %q", got)
	}
}
