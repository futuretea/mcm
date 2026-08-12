package manifest

import "testing"

func TestDecodeRejectsUnknownServerField(t *testing.T) {
	data := []byte(`version: 1
servers:
  example:
    transport: stdio
    command: example-server
    env:
      EXAMPLE_TOKEN: value
`)

	_, err := Decode(data)
	if err == nil {
		t.Fatal("Decode accepted an unknown server field")
	}
}
