package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuretea/mcm/internal/store"
)

func TestRunServerAddRejectsDuplicateAndPreservesExistingValue(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "original-command"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(server add original) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "replacement-command"}, userHome, strings.NewReader(""), &out, &errOut); exitCode == 0 {
		t.Error("Run(server add duplicate) exit code = 0, want nonzero")
	}
	if !strings.Contains(errOut.String(), "already exists") {
		t.Errorf("Run(server add duplicate) stderr = %q, want duplicate guidance", errOut.String())
	}

	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor parent: %v", err)
	}
	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "apply", "--target", "cursor", "--yes"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(apply --target cursor --yes) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read cursor target: %v", err)
	}
	if !strings.Contains(string(data), "original-command") || strings.Contains(string(data), "replacement-command") {
		t.Errorf("cursor target = %q, want original duplicate value only", data)
	}
}

func TestRunServerUpdateUpdatesExistingAndRejectsMissingServer(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "original-command"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(server add) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "update", "--name", "local", "--command", "replacement-command"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Errorf("Run(server update existing) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	} else {
		targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			t.Fatalf("create cursor parent: %v", err)
		}
		out.Reset()
		errOut.Reset()
		if exitCode := Run([]string{"--home", root, "apply", "--target", "cursor", "--yes"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
			t.Fatalf("Run(apply --target cursor --yes) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
		}
		data, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read cursor target: %v", err)
		}
		if !strings.Contains(string(data), "replacement-command") {
			t.Errorf("cursor target = %q, want updated server value", data)
		}
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "update", "--name", "missing", "--command", "missing-command"}, userHome, strings.NewReader(""), &out, &errOut); exitCode == 0 {
		t.Error("Run(server update missing) exit code = 0, want nonzero")
	}
	if !strings.Contains(errOut.String(), "not found") {
		t.Errorf("Run(server update missing) stderr = %q, want missing-server guidance", errOut.String())
	}
}

func TestRunHelpWritesUsageToStdout(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"server", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer

			if exitCode := Run(args, t.TempDir(), strings.NewReader(""), &out, &errOut); exitCode != 0 {
				t.Errorf("Run(%q) exit code = %d, want 0; stderr: %s", args, exitCode, errOut.String())
			}
			if !strings.Contains(strings.ToLower(out.String()), "usage") {
				t.Errorf("Run(%q) stdout = %q, want usage", args, out.String())
			}
		})
	}
}

func TestRunServerUnknownFlagNamesRequestedCommand(t *testing.T) {
	for _, command := range []string{"add", "update"} {
		t.Run(command, func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer

			if exitCode := Run([]string{"server", command, "--unknown", "value"}, t.TempDir(), strings.NewReader(""), &out, &errOut); exitCode != 2 {
				t.Errorf("Run(server %s --unknown) exit code = %d, want 2; stderr: %s", command, exitCode, errOut.String())
			}
			if !strings.Contains(errOut.String(), "unknown server "+command+" flag") {
				t.Errorf("Run(server %s --unknown) stderr = %q, want the requested command name", command, errOut.String())
			}
		})
	}
}

func TestRunPlanMissingClientParentIncludesRecoveryAction(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "plan", "--target", "cursor"}, userHome, strings.NewReader(""), &out, &errOut); exitCode == 0 {
		t.Error("Run(plan --target cursor) exit code = 0, want nonzero when parent is absent")
	}
	parent := filepath.Join(userHome, ".cursor")
	if !strings.Contains(errOut.String(), parent) || !strings.Contains(strings.ToLower(errOut.String()), "create") {
		t.Errorf("Run(plan --target cursor) stderr = %q, want the missing parent and an action to create it", errOut.String())
	}
}

func TestRunPlanExistingNativeConfigWarnsOnStderr(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor parent: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("{\n  \"mcpServers\": {}\n}\n"), 0o600); err != nil {
		t.Fatalf("write existing cursor configuration: %v", err)
	}
	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "node"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(server add) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "plan", "--target", "cursor"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(plan --target cursor) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	if !strings.Contains(out.String(), "cursor  "+targetPath) || !strings.Contains(out.String(), "add: local") {
		t.Errorf("Run(plan --target cursor) stdout = %q, want plan data", out.String())
	}

	stderr := strings.ToLower(errOut.String())
	if (!strings.Contains(stderr, "reserial") && !strings.Contains(stderr, "rewritten")) || !strings.Contains(stderr, "formatting and comments may change") {
		t.Errorf("Run(plan --target cursor) stderr = %q, want a reserialization warning about formatting and comments", errOut.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "formatting and comments may change") {
		t.Errorf("Run(plan --target cursor) stdout = %q, want no formatting warning", out.String())
	}
}

func TestRunApplyYesPreviewsAndWarnsBeforeRewritingNativeConfig(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	var out bytes.Buffer
	var setupErr bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &setupErr); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, setupErr.String())
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor parent: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("{\n  \"mcpServers\": {}\n}\n"), 0o600); err != nil {
		t.Fatalf("write existing cursor configuration: %v", err)
	}
	out.Reset()
	setupErr.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "node"}, userHome, strings.NewReader(""), &out, &setupErr); exitCode != 0 {
		t.Fatalf("Run(server add) exit code = %d, want 0; stderr: %s", exitCode, setupErr.String())
	}

	previewOut := &previewObserver{path: targetPath}
	errOut := &warningObserver{path: targetPath}
	if exitCode := Run([]string{"--home", root, "apply", "--target", "cursor", "--yes"}, userHome, strings.NewReader(""), previewOut, errOut); exitCode != 0 {
		t.Fatalf("Run(apply --target cursor --yes) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	if !strings.Contains(previewOut.beforeRewrite, "\"mcpServers\": {}") || strings.Contains(previewOut.beforeRewrite, "local") {
		t.Errorf("target when the plan preview was written = %q, want the original configuration", previewOut.beforeRewrite)
	}
	if !strings.HasSuffix(previewOut.String(), "applied\n") {
		t.Errorf("Run(apply --yes) stdout = %q, want final applied status", previewOut.String())
	}
	if !strings.Contains(errOut.beforeRewrite, "\"mcpServers\": {}") || strings.Contains(errOut.beforeRewrite, "local") {
		t.Errorf("target when reserialization warning was written = %q, want the original configuration", errOut.beforeRewrite)
	}
}

func TestRunApplyYesRecoversPendingJournalBeforePlanning(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor parent: %v", err)
	}
	native := []byte("{\n  \"mcpServers\": {}\n}\n")
	if err := os.WriteFile(targetPath, native, 0o600); err != nil {
		t.Fatalf("write cursor configuration: %v", err)
	}
	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "node"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(server add) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	digest := sha256.Sum256(native)
	emptyState := store.State{Version: 1, Targets: map[string]store.TargetState{}}
	if err := store.New(root).WriteIntent(store.Intent{
		Target:         "cursor",
		Path:           targetPath,
		ExpectedDigest: hex.EncodeToString(digest[:]),
		DesiredDigest:  "pending-write",
		OldState:       emptyState,
		NewState:       emptyState,
	}); err != nil {
		t.Fatalf("WriteIntent() error: %v", err)
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "apply", "--target", "cursor", "--yes"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(apply --yes) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}
	if data, err := os.ReadFile(targetPath); err != nil {
		t.Fatalf("read cursor configuration: %v", err)
	} else if !strings.Contains(string(data), "local") {
		t.Errorf("cursor configuration = %q, want applied local server", data)
	}
	if _, err := os.Stat(filepath.Join(root, "journal", "cursor.json")); !os.IsNotExist(err) {
		t.Errorf("journal intent after Run(apply --yes): stat error = %v, want not exist", err)
	}
}

type previewObserver struct {
	bytes.Buffer
	path          string
	beforeRewrite string
}

func (observer *previewObserver) Write(data []byte) (int, error) {
	if strings.HasPrefix(string(data), "cursor  ") && observer.beforeRewrite == "" {
		contents, err := os.ReadFile(observer.path)
		if err != nil {
			return 0, err
		}
		observer.beforeRewrite = string(contents)
	}
	return observer.Buffer.Write(data)
}

type warningObserver struct {
	bytes.Buffer
	path          string
	beforeRewrite string
}

func (observer *warningObserver) Write(data []byte) (int, error) {
	if strings.Contains(string(data), "reserialized as native") && observer.beforeRewrite == "" {
		contents, err := os.ReadFile(observer.path)
		if err != nil {
			return 0, err
		}
		observer.beforeRewrite = string(contents)
	}
	return observer.Buffer.Write(data)
}
