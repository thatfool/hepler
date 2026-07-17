package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// neutralizeHEPLEREnv blanks every HEPLER_* variable Load consults so each test
// starts from a clean slate regardless of what the surrounding environment
// injected. t.Setenv restores the originals when the test ends.
func neutralizeHEPLEREnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"HEPLER_OPENAI_API_BASE",
		"HEPLER_MODEL_NAME",
		"HEPLER_OPENAI_API_KEY",
		"HEPLER_OPENAI_API_KEY_CMD",
		"HEPLER_OPENAI_API_KEY_FILE",
	} {
		t.Setenv(k, "")
	}
}

func TestReadSecretInline(t *testing.T) {
	t.Setenv("TESTSECRET", "inline-value")
	t.Setenv("TESTSECRET_CMD", "")
	t.Setenv("TESTSECRET_FILE", "")

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "inline-value" {
		t.Fatalf("readSecret = %q, want %q", got, "inline-value")
	}
}

func TestReadSecretNothingSet(t *testing.T) {
	t.Setenv("TESTSECRET", "")
	t.Setenv("TESTSECRET_CMD", "")
	t.Setenv("TESTSECRET_FILE", "")

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "" {
		t.Fatalf("readSecret = %q, want empty when nothing is set", got)
	}
}

func TestReadSecretCMDTakesPrecedence(t *testing.T) {
	// _CMD wins over both _FILE and the inline variable.
	t.Setenv("TESTSECRET", "inline-value")
	t.Setenv("TESTSECRET_FILE", "/must/not/be/read")
	t.Setenv("TESTSECRET_CMD", "printf 'from-cmd'")

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "from-cmd" {
		t.Fatalf("readSecret = %q, want %q (_CMD must take precedence)", got, "from-cmd")
	}
}

func TestReadSecretFILETakesPrecedenceOverInline(t *testing.T) {
	t.Setenv("TESTSECRET", "inline-value")
	t.Setenv("TESTSECRET_CMD", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	t.Setenv("TESTSECRET_FILE", path)

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "from-file" {
		t.Fatalf("readSecret = %q, want %q (_FILE must take precedence over inline)", got, "from-file")
	}
}

func TestReadSecretCMDOutputTrimmed(t *testing.T) {
	t.Setenv("TESTSECRET_CMD", "printf '  \tthe-key  \n\t'")
	t.Setenv("TESTSECRET_FILE", "")

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "the-key" {
		t.Fatalf("readSecret = %q, want surrounding whitespace trimmed to %q", got, "the-key")
	}
}

func TestReadSecretFILEContentsTrimmed(t *testing.T) {
	t.Setenv("TESTSECRET_CMD", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("\n\tfrom-file\n\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	t.Setenv("TESTSECRET_FILE", path)

	got, err := readSecret("TESTSECRET")
	if err != nil {
		t.Fatalf("readSecret error: %v", err)
	}
	if got != "from-file" {
		t.Fatalf("readSecret = %q, want surrounding whitespace trimmed to %q", got, "from-file")
	}
}

func TestReadSecretCMDError(t *testing.T) {
	t.Setenv("TESTSECRET_CMD", "false")
	t.Setenv("TESTSECRET_FILE", "")

	_, err := readSecret("TESTSECRET")
	if err == nil {
		t.Fatal("readSecret expected error for failing _CMD, got nil")
	}
	if !strings.Contains(err.Error(), "TESTSECRET_CMD") {
		t.Fatalf("error %q should mention TESTSECRET_CMD", err.Error())
	}
}

func TestReadSecretFILEError(t *testing.T) {
	t.Setenv("TESTSECRET_FILE", filepath.Join(t.TempDir(), "does-not-exist"))
	t.Setenv("TESTSECRET_CMD", "")

	_, err := readSecret("TESTSECRET")
	if err == nil {
		t.Fatal("readSecret expected error for missing _FILE, got nil")
	}
	if !strings.Contains(err.Error(), "TESTSECRET_FILE") {
		t.Fatalf("error %q should mention TESTSECRET_FILE", err.Error())
	}
}

func TestLoadHappyPath(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "google/gemma-4-12b-qat")
	t.Setenv("HEPLER_OPENAI_API_KEY", "secret-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:11434/v1")
	}
	if cfg.Model != "google/gemma-4-12b-qat" {
		t.Errorf("Model = %q, want %q", cfg.Model, "google/gemma-4-12b-qat")
	}
	if cfg.APIKey != "secret-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret-key")
	}
}

func TestLoadAPIKeyOptional(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "gemma")
	// No API key of any kind is provided.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty (the key is optional)", cfg.APIKey)
	}
}

func TestLoadMissingBaseURL(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_MODEL_NAME", "gemma")

	_, err := Load()
	if err == nil {
		t.Fatal("Load expected error for missing HEPLER_OPENAI_API_BASE, got nil")
	}
	if !strings.Contains(err.Error(), "HEPLER_OPENAI_API_BASE") {
		t.Fatalf("error %q should mention HEPLER_OPENAI_API_BASE", err.Error())
	}
}

func TestLoadMissingModel(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")

	_, err := Load()
	if err == nil {
		t.Fatal("Load expected error for missing HEPLER_MODEL_NAME, got nil")
	}
	if !strings.Contains(err.Error(), "HEPLER_MODEL_NAME") {
		t.Fatalf("error %q should mention HEPLER_MODEL_NAME", err.Error())
	}
}

func TestLoadMissingBothRequired(t *testing.T) {
	neutralizeHEPLEREnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load expected error when both required variables are missing, got nil")
	}
	if !strings.Contains(err.Error(), "HEPLER_OPENAI_API_BASE") {
		t.Fatalf("error %q should mention HEPLER_OPENAI_API_BASE", err.Error())
	}
	if !strings.Contains(err.Error(), "HEPLER_MODEL_NAME") {
		t.Fatalf("error %q should mention HEPLER_MODEL_NAME", err.Error())
	}
}

func TestLoadAPIKeyFromCMD(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "gemma")
	t.Setenv("HEPLER_OPENAI_API_KEY_CMD", "printf 'cmd-key'")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.APIKey != "cmd-key" {
		t.Errorf("APIKey = %q, want %q (resolved from _CMD)", cfg.APIKey, "cmd-key")
	}
}

func TestLoadAPIKeyFromFile(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "gemma")
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("file-key"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	t.Setenv("HEPLER_OPENAI_API_KEY_FILE", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("APIKey = %q, want %q (resolved from _FILE)", cfg.APIKey, "file-key")
	}
}

func TestLoadAPIKeyCMDErrorPropagates(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "gemma")
	t.Setenv("HEPLER_OPENAI_API_KEY_CMD", "false")

	_, err := Load()
	if err == nil {
		t.Fatal("Load expected error from failing HEPLER_OPENAI_API_KEY_CMD, got nil")
	}
	if !strings.Contains(err.Error(), "HEPLER_OPENAI_API_KEY_CMD") {
		t.Fatalf("error %q should mention HEPLER_OPENAI_API_KEY_CMD", err.Error())
	}
}

func TestLoadAPIKeyFILEErrorPropagates(t *testing.T) {
	neutralizeHEPLEREnv(t)
	t.Setenv("HEPLER_OPENAI_API_BASE", "http://localhost:11434/v1")
	t.Setenv("HEPLER_MODEL_NAME", "gemma")
	t.Setenv("HEPLER_OPENAI_API_KEY_FILE", filepath.Join(t.TempDir(), "missing"))

	_, err := Load()
	if err == nil {
		t.Fatal("Load expected error from missing HEPLER_OPENAI_API_KEY_FILE, got nil")
	}
	if !strings.Contains(err.Error(), "HEPLER_OPENAI_API_KEY_FILE") {
		t.Fatalf("error %q should mention HEPLER_OPENAI_API_KEY_FILE", err.Error())
	}
}
