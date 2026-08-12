package manifest

import "testing"

func TestValidateRejectsLegacySSE(t *testing.T) {
	err := Validate(Config{
		Servers: map[string]Server{
			"legacy": {Transport: "sse"},
		},
	})
	if err == nil {
		t.Fatal("Validate accepted legacy SSE transport")
	}
}
