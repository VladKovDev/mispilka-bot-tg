package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		templateDir string
		fileName    string
		setup       func(string) error
		want        string
		wantErr     bool
	}{
		{
			name:        "load existing template",
			templateDir: tmpDir,
			fileName:    "welcome.md",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "welcome.md"), []byte("Hello {{name}}"), 0644)
			},
			want:    "Hello {{name}}",
			wantErr: false,
		},
		{
			name:        "load template with multiline content",
			templateDir: tmpDir,
			fileName:    "intro.md",
			setup: func(dir string) error {
				content := "Welcome {{name}}!\n\nThis is {{course}}.\nEnjoy!"
				return os.WriteFile(filepath.Join(dir, "intro.md"), []byte(content), 0644)
			},
			want:    "Welcome {{name}}!\n\nThis is {{course}}.\nEnjoy!",
			wantErr: false,
		},
		{
			name:        "load non-existent template",
			templateDir: tmpDir,
			fileName:    "missing.md",
			setup:       func(dir string) error { return nil },
			want:        "",
			wantErr:     true,
		},
		{
			name:        "load empty template",
			templateDir: tmpDir,
			fileName:    "empty.md",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "empty.md"), []byte(""), 0644)
			},
			want:    "",
			wantErr: false,
		},
		{
			name:        "load template with unicode",
			templateDir: tmpDir,
			fileName:    "unicode.md",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "unicode.md"), []byte("Привет {{name}} 🎉"), 0644)
			},
			want:    "Привет {{name}} 🎉",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(tt.templateDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			l := NewLoader(tt.templateDir)
			got, err := l.Load(tt.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Loader.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Loader.Load() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoader_LoadAndRender(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup template file with scoped variables
	templateContent := "Hello {{user.name}}, welcome to {{scenario.place}}!"
	templatePath := filepath.Join(tmpDir, "welcome.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	l := NewLoader(tmpDir)
	r := NewRenderer()

	// Load and render
	template, err := l.Load("welcome.md")
	if err != nil {
		t.Fatalf("failed to load template: %v", err)
	}

	vars := map[string]string{"user.name": "Alice", "scenario.place": "Wonderland"}
	result, err := r.Render(template, vars)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Hello Alice, welcome to Wonderland!"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
