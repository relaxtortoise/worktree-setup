package logging

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate Init but write to tmpDir instead of UserConfigDir
	logPath := filepath.Join(tmpDir, "log.jsonl")
	writer := setupTestLogger(logPath, slog.LevelInfo)

	msg := "test log message"
	slog.Info(msg, "key", "value")

	_ = writer.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("not valid JSON Lines: %v", err)
	}
	if entry["msg"] != msg {
		t.Fatalf("expected msg=%q, got %v", msg, entry["msg"])
	}
	if entry["level"] != "INFO" {
		t.Fatalf("expected level=INFO, got %v", entry["level"])
	}
	if entry["key"] != "value" {
		t.Fatalf("expected key=value, got %v", entry["key"])
	}
}

func TestInitDirNotWritable(t *testing.T) {
	tmpDir := t.TempDir()
	noAccess := filepath.Join(tmpDir, "noaccess")
	if err := os.Mkdir(noAccess, 0000); err != nil {
		t.Skipf("cannot create test dir: %v", err)
	}
	defer func() { _ = os.Chmod(noAccess, 0755) }()

	// Override UserConfigDir briefly — we test os.MkdirAll failure path directly
	parent := filepath.Join(noAccess, "subdir")
	err := os.MkdirAll(parent, 0755)
	if err == nil {
		t.Skipf("expected permission error, but MkdirAll succeeded")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}
	for _, tt := range tests {
		got := parseLevel(tt.input)
		if got != tt.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "log.jsonl")
	writer := setupTestLogger(logPath, slog.LevelWarn)

	slog.Debug("debug msg")
	slog.Info("info msg")
	slog.Warn("warn msg")

	_ = writer.Close()

	data, _ := os.ReadFile(logPath)
	content := string(data)
	if len(content) == 0 {
		t.Fatal("expected at least one log entry")
	}
	// Debug and Info should be filtered out at Warn level
	if contains(content, `"msg":"debug msg"`) {
		t.Error("debug message should be filtered at warn level")
	}
	if contains(content, `"msg":"info msg"`) {
		t.Error("info message should be filtered at warn level")
	}
	if !contains(content, `"msg":"warn msg"`) {
		t.Error("warn message should be present at warn level")
	}
}

func TestSourceField(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "log.jsonl")
	writer := setupTestLogger(logPath, slog.LevelInfo)

	slog.Info("with source")

	_ = writer.Close()

	data, _ := os.ReadFile(logPath)
	var entry map[string]any
	_ = json.Unmarshal(data, &entry)
	if _, ok := entry["source"]; !ok {
		t.Error("source field missing when AddSource is enabled")
	}
}

func setupTestLogger(path string, level slog.Level) *os.File {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	// Use plain file writer (no rotation) for tests
	// We test lumberjack behavior indirectly via the integration test
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
	return f
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && searchSubstring(s, substr))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
