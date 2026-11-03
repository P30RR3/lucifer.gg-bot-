package config

import "testing"

func TestCerebrasDefaultsUseCurrentSharedModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{name: "empty", want: "gpt-oss-120b"},
		{name: "retired default", model: "llama-3.3-70b", want: "gpt-oss-120b"},
		{name: "custom model", model: "qwen-3.8-27b", want: "qwen-3.8-27b"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := CerebrasConfig{Model: test.model}
			applyCerebrasDefaults(&config)
			if config.Model != test.want {
				t.Fatalf("model = %q, want %q", config.Model, test.want)
			}
		})
	}
}
