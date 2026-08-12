package manifest

import (
	"os"
	"testing"
)

func TestLocationSaveAndLoadRoundTripsConfig(t *testing.T) {
	home := t.TempDir()
	location := NewLocation(home, "", "")
	if err := location.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	config := Config{
		Version: 1,
		Servers: map[string]Server{
			"example": {
				Transport: TransportStdio,
				Command:   "example-server",
			},
		},
	}
	if err := location.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := location.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Servers["example"].Command; got != "example-server" {
		t.Errorf("loaded server command = %q, want %q", got, "example-server")
	}

	info, err := os.Stat(location.ConfigPath)
	if err != nil {
		t.Fatalf("stat config file %q: %v", location.ConfigPath, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config file permissions = %o, want %o", got, 0o600)
	}
}
