package adapter

import (
	"encoding/json"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
	"github.com/pelletier/go-toml/v2"
)

var _ func(string, manifest.Config, []byte, map[string]struct{}) (Result, error) = Render

func TestRenderTargets(t *testing.T) {
	config := manifest.Config{
		Version: 1,
		Servers: map[string]manifest.Server{
			"local": {
				Transport: manifest.TransportStdio,
				Command:   "node",
				Args:      []string{"x"},
			},
			"remote": {
				Transport: manifest.TransportStreamableHTTP,
				URL:       "https://example.test/mcp",
			},
		},
	}

	tests := []struct {
		target string
		verify func(*testing.T, []byte)
	}{
		{
			target: "cursor",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				local := object(t, object(t, document, "mcpServers"), "local")
				equalString(t, local, "type", "stdio")
			},
		},
		{
			target: "claude-code",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				remote := object(t, object(t, document, "mcpServers"), "remote")
				equalString(t, remote, "type", "http")
			},
		},
		{
			target: "codex",
			verify: func(t *testing.T, bytes []byte) {
				var document map[string]any
				if err := toml.Unmarshal(bytes, &document); err != nil {
					t.Fatalf("Codex output is not TOML: %v\n%s", err, bytes)
				}
				servers := object(t, document, "mcp_servers")
				local := object(t, servers, "local")
				remote := object(t, servers, "remote")
				equalString(t, local, "command", "node")
				equalString(t, remote, "url", "https://example.test/mcp")
			},
		},
		{
			target: "vs-code",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				if _, ok := document["servers"].(map[string]any); !ok {
					t.Errorf("VS Code output has no root servers object: %v", document)
				}
			},
		},
		{
			target: "qoder-cli",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				remote := object(t, object(t, document, "mcpServers"), "remote")
				equalString(t, remote, "type", "http")
			},
		},
		{
			target: "qoder-ide",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				remote := object(t, object(t, document, "mcpServers"), "remote")
				equalString(t, remote, "url", "https://example.test/mcp")
			},
		},
		{
			target: "opencode",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				servers := object(t, object(t, document, "mcp"), "servers")
				local := object(t, servers, "local")
				remote := object(t, servers, "remote")
				equalString(t, local, "type", "local")
				command, ok := local["command"].([]any)
				if !ok || len(command) != 2 || command[0] != "node" || command[1] != "x" {
					t.Errorf("OpenCode local command = %v, want [node x]", local["command"])
				}
				equalString(t, remote, "type", "remote")
			},
		},
		{
			target: "mcp-cli",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				if _, ok := document["mcpServers"].(map[string]any); !ok {
					t.Errorf("mcp-cli output has no root mcpServers object: %v", document)
				}
			},
		},
		{
			target: "mcpc",
			verify: func(t *testing.T, bytes []byte) {
				document := decodeJSON(t, bytes)
				if _, ok := document["mcpServers"].(map[string]any); !ok {
					t.Errorf("mcpc output has no root mcpServers object: %v", document)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			result, err := Render(test.target, config, nil, map[string]struct{}{})
			if err != nil {
				t.Fatalf("Render(%q) error: %v", test.target, err)
			}
			test.verify(t, result.Bytes)
		})
	}
}

func decodeJSON(t *testing.T, bytes []byte) map[string]any {
	t.Helper()

	var document map[string]any
	if err := json.Unmarshal(bytes, &document); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, bytes)
	}
	return document
}

func object(t *testing.T, document map[string]any, name string) map[string]any {
	t.Helper()

	value, ok := document[name].(map[string]any)
	if !ok {
		t.Fatalf("%q = %T, want object", name, document[name])
	}
	return value
}

func equalString(t *testing.T, document map[string]any, name, want string) {
	t.Helper()

	if got, ok := document[name].(string); !ok || got != want {
		t.Errorf("%q = %q, want %q", name, got, want)
	}
}
