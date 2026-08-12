// Package adapter renders MCM servers into client-native configuration formats.
package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/futuretea/mcm/internal/manifest"
	"github.com/pelletier/go-toml/v2"
)

// Result contains a rendered native document and value-free ownership metadata.
type Result struct {
	Bytes   []byte
	Changes []Change
	Managed map[string]struct{}
}

// Change is a value-free managed server change.
type Change struct {
	Action string
	Name   string
}

// Render merges MCM servers into one supported target format.
func Render(target string, config manifest.Config, existing []byte, managed map[string]struct{}) (Result, error) {
	if err := manifest.Validate(config); err != nil {
		return Result{}, err
	}
	if target == "codex" {
		return renderTOML(config, existing, managed)
	}
	return renderJSON(target, config, existing, managed)
}

// IsTarget reports whether target is supported by this MVP.
func IsTarget(target string) bool {
	switch target {
	case "cursor", "claude-code", "codex", "vs-code", "qoder-cli", "qoder-ide", "opencode", "mcp-cli", "mcpc":
		return true
	default:
		return false
	}
}

func renderJSON(target string, config manifest.Config, existing []byte, managed map[string]struct{}) (Result, error) {
	if !IsTarget(target) {
		return Result{}, fmt.Errorf("unsupported target %q", target)
	}
	document, err := decodeDocument(existing, target == "opencode")
	if err != nil {
		return Result{}, err
	}
	servers, err := jsonServers(document, target)
	if err != nil {
		return Result{}, err
	}
	changes, nextManaged, err := mergeServers(servers, config, managed, func(server manifest.Server) map[string]any {
		return renderJSONServer(target, server)
	})
	if err != nil {
		return Result{}, err
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode %s configuration: %w", target, err)
	}
	return Result{Bytes: append(encoded, '\n'), Changes: changes, Managed: nextManaged}, nil
}

func decodeDocument(existing []byte, jsonc bool) (map[string]any, error) {
	if len(bytes.TrimSpace(existing)) == 0 {
		return map[string]any{}, nil
	}
	data := existing
	if jsonc {
		var err error
		data, err = stripJSONC(existing)
		if err != nil {
			return nil, err
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode JSON configuration: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode JSON configuration: multiple root values")
		}
		return nil, fmt.Errorf("decode JSON configuration: %w", err)
	}
	if document == nil {
		return nil, fmt.Errorf("configuration root must be an object")
	}
	return document, nil
}

func jsonServers(document map[string]any, target string) (map[string]any, error) {
	if target == "opencode" {
		mcp, err := objectValue(document, "mcp")
		if err != nil {
			return nil, err
		}
		return objectValue(mcp, "servers")
	}
	root := "mcpServers"
	if target == "vs-code" {
		root = "servers"
	}
	return objectValue(document, root)
}

func objectValue(parent map[string]any, key string) (map[string]any, error) {
	value, exists := parent[key]
	if !exists {
		child := map[string]any{}
		parent[key] = child
		return child, nil
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("configuration field %q must be an object", key)
	}
	return child, nil
}

func mergeServers(existing map[string]any, config manifest.Config, managed map[string]struct{}, render func(manifest.Server) map[string]any) ([]Change, map[string]struct{}, error) {
	if managed == nil {
		managed = map[string]struct{}{}
	}
	names := sortedServerNames(config.Servers)
	changes := make([]Change, 0, len(names)+len(managed))
	nextManaged := make(map[string]struct{}, len(names))
	for _, name := range names {
		_, exists := existing[name]
		if exists {
			if _, owned := managed[name]; !owned {
				return nil, nil, fmt.Errorf("unmanaged server %q already exists", name)
			}
			changes = append(changes, Change{Action: "update", Name: name})
		} else {
			changes = append(changes, Change{Action: "add", Name: name})
		}
		existing[name] = render(config.Servers[name])
		nextManaged[name] = struct{}{}
	}
	for name := range managed {
		if _, wanted := config.Servers[name]; wanted {
			continue
		}
		if _, exists := existing[name]; exists {
			delete(existing, name)
			changes = append(changes, Change{Action: "remove", Name: name})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Name == changes[j].Name {
			return changes[i].Action < changes[j].Action
		}
		return changes[i].Name < changes[j].Name
	})
	return changes, nextManaged, nil
}

func sortedServerNames(servers map[string]manifest.Server) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderJSONServer(target string, server manifest.Server) map[string]any {
	if target == "opencode" {
		if server.Transport == manifest.TransportStdio {
			command := append([]any{server.Command}, stringsToAny(server.Args)...)
			return map[string]any{"type": "local", "command": command}
		}
		return map[string]any{"type": "remote", "url": server.URL}
	}
	if server.Transport == manifest.TransportStdio {
		entry := map[string]any{"command": server.Command}
		if len(server.Args) > 0 {
			entry["args"] = stringsToAny(server.Args)
		}
		if target == "cursor" || target == "claude-code" || target == "vs-code" {
			entry["type"] = "stdio"
		}
		return entry
	}
	entry := map[string]any{"url": server.URL}
	if target == "claude-code" || target == "vs-code" || target == "qoder-cli" {
		entry["type"] = "http"
	}
	return entry
}

func stringsToAny(values []string) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}

func renderTOML(config manifest.Config, existing []byte, managed map[string]struct{}) (Result, error) {
	document := map[string]any{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := toml.Unmarshal(existing, &document); err != nil {
			return Result{}, fmt.Errorf("decode Codex TOML: %w", err)
		}
	}
	servers, err := tomlObject(document, "mcp_servers")
	if err != nil {
		return Result{}, err
	}
	changes, nextManaged, err := mergeServers(servers, config, managed, func(server manifest.Server) map[string]any {
		if server.Transport == manifest.TransportStdio {
			entry := map[string]any{"command": server.Command}
			if len(server.Args) > 0 {
				entry["args"] = server.Args
			}
			return entry
		}
		return map[string]any{"url": server.URL}
	})
	if err != nil {
		return Result{}, err
	}
	encoded, err := toml.Marshal(document)
	if err != nil {
		return Result{}, fmt.Errorf("encode Codex TOML: %w", err)
	}
	return Result{Bytes: encoded, Changes: changes, Managed: nextManaged}, nil
}

func tomlObject(parent map[string]any, key string) (map[string]any, error) {
	value, exists := parent[key]
	if !exists {
		child := map[string]any{}
		parent[key] = child
		return child, nil
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Codex field %q must be a table", key)
	}
	return child, nil
}
