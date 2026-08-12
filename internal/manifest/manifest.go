// Package manifest defines MCM's source-of-truth configuration model.
package manifest

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	TransportStdio          = "stdio"
	TransportStreamableHTTP = "streamable-http"
)

// Config is the versioned MCM manifest.
type Config struct {
	Version int               `yaml:"version"`
	Servers map[string]Server `yaml:"servers"`
	Targets map[string]Target `yaml:"targets"`
}

// Server is a public MCP server definition.
type Server struct {
	Transport string   `yaml:"transport"`
	Command   string   `yaml:"command,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	URL       string   `yaml:"url,omitempty"`
}

// Target optionally overrides a client's target file path.
type Target struct {
	Path string `yaml:"path,omitempty"`
}

// Decode parses a manifest and rejects fields outside the supported schema.
func Decode(data []byte) (Config, error) {
	var config Config
	if err := yaml.UnmarshalWithOptions(data, &config, yaml.Strict()); err != nil {
		return Config{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := Validate(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Validate rejects unsupported transports and contradictory server shapes.
func Validate(config Config) error {
	if config.Version != 1 {
		return fmt.Errorf("manifest version must be 1")
	}
	for name, server := range config.Servers {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("server name must not be empty")
		}

		switch server.Transport {
		case TransportStdio:
			if server.Command == "" || server.URL != "" {
				return fmt.Errorf("stdio server %q requires command and no url", name)
			}
		case TransportStreamableHTTP:
			if server.URL == "" || server.Command != "" || len(server.Args) != 0 {
				return fmt.Errorf("streamable-http server %q requires url only", name)
			}
		default:
			return fmt.Errorf("server %q has unsupported transport %q", name, server.Transport)
		}
	}
	return nil
}
