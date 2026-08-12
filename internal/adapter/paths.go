package adapter

import (
	"fmt"
	"path/filepath"
)

// ResolvePath selects one target file without creating or inspecting it.
func ResolvePath(target, userHome, mcmRoot, override, configured string) (string, error) {
	if !IsTarget(target) {
		return "", fmt.Errorf("unsupported target %q", target)
	}
	if target == "mcpc" {
		if override != "" || configured != "" {
			return "", fmt.Errorf("mcpc export path is fixed by the MCM root")
		}
		return filepath.Join(mcmRoot, "exports", "mcpc.json"), nil
	}
	if override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("target path must be absolute")
		}
		return override, nil
	}
	if configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("configured target path must be absolute")
		}
		return configured, nil
	}
	switch target {
	case "cursor":
		return filepath.Join(userHome, ".cursor", "mcp.json"), nil
	case "claude-code":
		return filepath.Join(userHome, ".claude.json"), nil
	case "codex":
		return filepath.Join(userHome, ".codex", "config.toml"), nil
	case "vs-code":
		return "", fmt.Errorf("vs-code requires an explicit target path")
	case "qoder-cli":
		return filepath.Join(userHome, ".qoder", "settings.json"), nil
	case "qoder-ide":
		return filepath.Join(userHome, ".qoder", "mcp.json"), nil
	case "opencode":
		return filepath.Join(userHome, ".config", "opencode", "opencode.json"), nil
	case "mcp-cli":
		return filepath.Join(userHome, ".config", "mcp", "mcp_servers.json"), nil
	default:
		return "", fmt.Errorf("unsupported target %q", target)
	}
}
