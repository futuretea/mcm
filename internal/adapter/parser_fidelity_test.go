package adapter

import (
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
	"github.com/pelletier/go-toml/v2"
)

func TestRenderRejectsTrailingJSONGarbage(t *testing.T) {
	config := manifest.Config{Version: 1, Servers: map[string]manifest.Server{}}

	for _, target := range []string{"cursor", "opencode"} {
		t.Run(target, func(t *testing.T) {
			existing := []byte(`{"mcpServers": {}} trailing`)
			if target == "opencode" {
				existing = []byte(`{"mcp": {"servers": {}}} trailing`)
			}

			if _, err := Render(target, config, existing, map[string]struct{}{}); err == nil {
				t.Fatal("Render() error = nil, want error for trailing garbage")
			}
		})
	}
}

func TestRenderCodexPreservesUnmanagedSingleQuoteString(t *testing.T) {
	config := manifest.Config{
		Version: 1,
		Servers: map[string]manifest.Server{
			"local": {
				Transport: manifest.TransportStdio,
				Command:   "node",
			},
		},
	}

	result, err := Render("codex", config, []byte("label = \"O'Reilly's\"\n"), map[string]struct{}{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	var document map[string]any
	if err := toml.Unmarshal(result.Bytes, &document); err != nil {
		t.Fatalf("rendered output is not TOML: %v\n%s", err, result.Bytes)
	}
	if got, want := document["label"], "O'Reilly's"; got != want {
		t.Errorf("label = %q, want %q", got, want)
	}
}
