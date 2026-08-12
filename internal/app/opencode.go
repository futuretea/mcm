package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/futuretea/mcm/internal/safeio"
)

var afterOpenCodeResolution = func(string) error { return nil }

// ResolveOpenCodePath selects the sole supported OpenCode JSON or JSONC entry.
func ResolveOpenCodePath(userHome string) (string, error) {
	directory := filepath.Join(userHome, ".config", "opencode")
	jsonPath := filepath.Join(directory, "opencode.json")
	jsoncPath := filepath.Join(directory, "opencode.jsonc")
	jsonExists, err := regularEntryExists(jsonPath)
	if err != nil {
		return "", err
	}
	jsoncExists, err := regularEntryExists(jsoncPath)
	if err != nil {
		return "", err
	}
	if jsonExists && jsoncExists {
		return "", fmt.Errorf("OpenCode has both opencode.json and opencode.jsonc")
	}
	if jsoncExists {
		return jsoncPath, nil
	}
	return jsonPath, nil
}

func regularEntryExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", path)
	}
	return true, nil
}

func validateOpenCodeSibling(target *safeio.Target, path string) error {
	name := filepath.Base(path)
	sibling := ""
	switch name {
	case "opencode.json":
		sibling = "opencode.jsonc"
	case "opencode.jsonc":
		sibling = "opencode.json"
	default:
		return nil
	}
	exists, err := target.EntryExists(sibling)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("OpenCode has both opencode.json and opencode.jsonc")
	}
	return nil
}

func isDefaultOpenCodePath(userHome, path string) bool {
	directory := filepath.Join(userHome, ".config", "opencode")
	return path == filepath.Join(directory, "opencode.json") || path == filepath.Join(directory, "opencode.jsonc")
}
