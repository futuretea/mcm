package adapter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestRenderOpenCodeJSONCPreservesStringContentAndNumberLexeme(t *testing.T) {
	config := manifest.Config{
		Version: 1,
		Servers: map[string]manifest.Server{
			"local": {
				Transport: manifest.TransportStdio,
				Command:   "node",
			},
		},
	}
	existing := []byte(`{"note":"value,} // /* */", "mcp":{"timeout": 9007199254740993,},}`)

	result, err := Render("opencode", config, existing, map[string]struct{}{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(result.Bytes))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("rendered output is not strict JSON: %v\n%s", err, result.Bytes)
	}
	if got, want := document["note"], "value,} // /* */"; got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
	mcp, ok := document["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("mcp = %T, want object", document["mcp"])
	}
	if got, want := mcp["timeout"], json.Number("9007199254740993"); got != want {
		t.Errorf("mcp.timeout = %v, want %v", got, want)
	}
}
