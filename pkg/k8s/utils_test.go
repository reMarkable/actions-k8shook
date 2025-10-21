package k8s

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reMarkable/k8s-hook/pkg/types"
)

func TestCopyExternals(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Setenv("RUNNER_WORKSPACE", workspace); err != nil {
		t.Fatalf("Failed to set RUNNER_WORKSPACE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("RUNNER_WORKSPACE")
		if err != nil {
			t.Fatalf("Failed to unset RUNNER_WORKSPACE: %v", err)
		}
	}()

	// Setup source externals directory and file
	srcExternals := filepath.Join(workspace, "../../externals")
	dstExternals := filepath.Join(workspace, "../externals")
	if err := os.MkdirAll(srcExternals, 0o755); err != nil {
		t.Fatalf("Failed to create src externals dir: %v", err)
	}
	testFile := filepath.Join(srcExternals, "test.txt")
	content := []byte("externals content")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Call copyExternals
	copyExternals()

	// Check that file was copied
	copiedFile := filepath.Join(dstExternals, "test.txt")
	data, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Fatalf("Failed to read copied externals file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Copied externals file content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestWriteRunScript(t *testing.T) {
	args := types.InputArgs{
		ContainerDefinition: types.ContainerDefinition{
			PrependPath:          []string{"/usr/local/bin", "/custom/bin"},
			Entrypoint:           "bash",
			EntrypointArgs:       []string{"-c", "echo hello"},
			EnvironmentVariables: map[string]string{"FOO": "bar"},
			WorkingDirectory:     "/tmp",
		},
	}
	client := &K8sClient{}
	scriptPath, tempPath, err := client.writeRunScript(args)
	if err != nil {
		t.Fatalf("writeRunScript returned error: %v", err)
	}
	defer func() {
		err = os.Remove(tempPath)
		if err != nil {
			t.Fatalf("Failed to remove temp script file: %v", err)
		}
	}()

	// Check returned paths
	if filepath.Base(tempPath) != filepath.Base(scriptPath) {
		t.Errorf("Base names of scriptPath and tempPath should match")
	}

	// Check file exists
	data, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to read script file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "#!/bin/sh -l") {
		t.Errorf("Script missing shebang")
	}
	if !strings.Contains(content, "export PATH=/usr/local/bin:/custom/bin:$PATH") {
		t.Errorf("Script missing correct PATH export")
	}
	if !strings.Contains(content, `env "FOO=bar"`) {
		t.Errorf("Script missing environment variables")
	}
	if !strings.Contains(content, "cd /tmp && exec") {
		t.Errorf("Script missing cd and exec command")
	}
}

func TestScriptEnvironment(t *testing.T) {
	env := map[string]string{
		"FOO": `bar"baz$qux\test`,
		"BAR": "simple",
	}
	got, err := scriptEnvironment(env)
	if err != nil {
		t.Fatalf("scriptEnvironment returned error: %v", err)
	}
	wantPrefix := `env "FOO=bar\"baz\$qux\\test" "BAR=simple"`
	if got != wantPrefix && got != `env "BAR=simple" "FOO=bar\"baz\$qux\\test"` {
		t.Errorf("scriptEnvironment output mismatch:\ngot:  %s\nwant: %s", got, wantPrefix)
	}

	// Test invalid key
	invalidEnv := map[string]string{
		`FO"O`: "bad",
	}
	_, err = scriptEnvironment(invalidEnv)
	if err == nil {
		t.Errorf("Expected error for invalid key, got nil")
	}
}

func TestCopyFile(t *testing.T) {
	srcFile, err := os.CreateTemp("", "srcfile")
	if err != nil {
		t.Fatalf("Failed to create temp source file: %v", err)
	}
	defer func() {
		if err := os.Remove(srcFile.Name()); err != nil {
			t.Fatalf("Failed to remove temp source file: %v", err)
		}
	}()

	content := []byte("hello world")
	if _, err := srcFile.Write(content); err != nil {
		t.Fatalf("Failed to write to source file: %v", err)
	}
	err = srcFile.Close()
	if err != nil {
		t.Fatalf("Failed to close source file: %v", err)
	}

	dstFile := filepath.Join(os.TempDir(), "dstfile")
	defer func() {
		if err := os.Remove(dstFile); err != nil {
			t.Fatalf("Failed to remove temp destination file: %v", err)
		}
	}()

	mode := os.FileMode(0o644)
	if err := copyFile(srcFile.Name(), dstFile, mode); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("File content mismatch: got %q, want %q", string(data), string(content))
	}

	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}
	if info.Mode().Perm() != mode {
		t.Errorf("File mode mismatch: got %v, want %v", info.Mode().Perm(), mode)
	}
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dstdir")

	// Create files and nested directories in srcDir
	rootFile := filepath.Join(srcDir, "root.txt")
	nestedDir := filepath.Join(srcDir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}
	nestedFile := filepath.Join(nestedDir, "nested.txt")

	rootContent := []byte("root file content")
	nestedContent := []byte("nested file content")

	if err := os.WriteFile(rootFile, rootContent, 0o644); err != nil {
		t.Fatalf("Failed to write root file: %v", err)
	}
	if err := os.WriteFile(nestedFile, nestedContent, 0o644); err != nil {
		t.Fatalf("Failed to write nested file: %v", err)
	}

	// Call copyDir
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Check root file
	copiedRoot := filepath.Join(dstDir, "root.txt")
	data, err := os.ReadFile(copiedRoot)
	if err != nil {
		t.Fatalf("Failed to read copied root file: %v", err)
	}
	if string(data) != string(rootContent) {
		t.Errorf("Root file content mismatch: got %q, want %q", string(data), string(rootContent))
	}

	// Check nested file
	copiedNested := filepath.Join(dstDir, "nested", "nested.txt")
	data, err = os.ReadFile(copiedNested)
	if err != nil {
		t.Fatalf("Failed to read copied nested file: %v", err)
	}
	if string(data) != string(nestedContent) {
		t.Errorf("Nested file content mismatch: got %q, want %q", string(data), string(nestedContent))
	}
}
