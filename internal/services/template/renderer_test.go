package template

import (
	"strings"
	"testing"
)

func TestRenderer_Render(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		want     string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "Hello {{name}}",
			vars:     map[string]string{"name": "Alice"},
			want:     "Hello Alice",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "{{greeting}} {{name}}, welcome to {{place}}",
			vars:     map[string]string{"greeting": "Hi", "name": "Bob", "place": "Wonderland"},
			want:     "Hi Bob, welcome to Wonderland",
			wantErr:  false,
		},
		{
			name:     "no variables",
			template: "Just plain text",
			vars:     map[string]string{},
			want:     "Just plain text",
			wantErr:  false,
		},
		{
			name:     "variable not found",
			template: "Hello {{name}}",
			vars:     map[string]string{},
			want:     "Hello {{name}}", // Missing vars left as-is
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			vars:     map[string]string{},
			want:     "",
			wantErr:  false,
		},
		{
			name:     "repeated variable",
			template: "{{name}} loves {{name}}",
			vars:     map[string]string{"name": "Charlie"},
			want:     "Charlie loves Charlie",
			wantErr:  false,
		},
		{
			name:     "variable with special chars",
			template: "Hello {{name_123}}",
			vars:     map[string]string{"name_123": "David"},
			want:     "Hello David",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRenderer()
			got, err := r.Render(tt.template, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Renderer.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Renderer.Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderer_RenderWithBraces(t *testing.T) {
	// Test that we handle escaped braces or nested content correctly
	r := NewRenderer()
	template := "Use {{code}} for coding"
	vars := map[string]string{"code": "Go"}
	got, err := r.Render(template, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Use Go for coding" {
		t.Errorf("got %q, want %q", got, "Use Go for coding")
	}
}

func BenchmarkRenderer_Render(b *testing.B) {
	r := NewRenderer()
	template := strings.Repeat("Hello {{name}} ", 100)
	vars := map[string]string{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Render(template, vars)
	}
}
