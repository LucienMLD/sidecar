package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkIsPluginInstalled measures the cost of plugin detection on every workspace creation
func BenchmarkIsPluginInstalled(b *testing.B) {
	dir := b.TempDir()
	registry := `{"version":1,"plugins":{"compound-engineering@every-marketplace":{"version":"2.31.1","installPath":"/tmp/ce"}}}`
	pluginsDir := filepath.Join(dir, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(registry), 0644); err != nil {
		b.Fatal(err)
	}
	b.Setenv("HOME", dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsPluginInstalled("compound-engineering@every-marketplace")
	}
}

// BenchmarkIsPluginInstalledMiss measures cache miss performance
func BenchmarkIsPluginInstalledMiss(b *testing.B) {
	dir := b.TempDir()
	b.Setenv("HOME", dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsPluginInstalled("compound-engineering@every-marketplace")
	}
}

// BenchmarkDefaultPrompts measures the cost of generating default prompts
func BenchmarkDefaultPrompts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultPrompts()
	}
}

// BenchmarkLoadPrompts measures the cost of loading and merging prompts
func BenchmarkLoadPrompts(b *testing.B) {
	globalDir := b.TempDir()
	projectDir := b.TempDir()

	// Setup global config
	globalConfig := `{
  "prompts": [
    {"name": "Prompt 1", "ticketMode": "required", "body": "Body 1"},
    {"name": "Prompt 2", "ticketMode": "optional", "body": "Body 2"},
    {"name": "Prompt 3", "ticketMode": "none", "body": "Body 3"}
  ]
}`
	os.WriteFile(filepath.Join(globalDir, "config.json"), []byte(globalConfig), 0644)

	// Setup project config
	sidecarDir := filepath.Join(projectDir, ".sidecar")
	os.MkdirAll(sidecarDir, 0755)
	projectConfig := `{
  "prompts": [
    {"name": "Prompt 4", "ticketMode": "required", "body": "Body 4"},
    {"name": "Prompt 2", "ticketMode": "required", "body": "Override Body 2"}
  ]
}`
	os.WriteFile(filepath.Join(sidecarDir, "config.json"), []byte(projectConfig), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LoadPrompts(globalDir, projectDir)
	}
}

// BenchmarkExpandPromptTemplate measures template expansion performance
func BenchmarkExpandPromptTemplate(b *testing.B) {
	body := `td usage --new-session

Full compound-engineering workflow for {{ticket || 'this feature'}}:

1. /workflows:plan
2. /deepen-plan
3. /workflows:work
4. /workflows:review
5. /workflows:compound`

	b.Run("with_ticket", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ExpandPromptTemplate(body, "td-abc123")
		}
	})

	b.Run("with_fallback", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ExpandPromptTemplate(body, "")
		}
	})
}

// BenchmarkExtractFallback measures regex performance for fallback extraction
func BenchmarkExtractFallback(b *testing.B) {
	body := "Review {{ticket || 'open reviews'}} and fix issues"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractFallback(body)
	}
}
