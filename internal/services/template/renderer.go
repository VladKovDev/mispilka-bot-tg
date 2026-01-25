package template

import (
	"regexp"
)

// Renderer handles template variable substitution
type Renderer struct {
	// varPattern matches {{scope.variable_name}} placeholders (scoped syntax only)
	// Matches {{scope.variable}} where scope and variable are word characters
	// Does NOT match {{variable}} without a scope
	varPattern *regexp.Regexp
}

// NewRenderer creates a new template renderer
func NewRenderer() *Renderer {
	return &Renderer{
		// Requires scoped variables: {{scope.variable}} syntax only
		// Matches {{scope.variable}} where scope and variable are word characters
		varPattern: regexp.MustCompile(`\{\{(\w+\.\w+(?:\.\w+)*)\}\}`),
	}
}

// Render replaces {{scope.variable}} placeholders in template with values from vars map
// Requires scoped syntax only: {{bot.name}}, {{user.name}}, {{scenario.name}}
// Does NOT support {{variable}} without a scope - these will be left unchanged
// If a variable is not found in the map, the placeholder is left unchanged
func (r *Renderer) Render(template string, vars map[string]string) (string, error) {
	result := r.varPattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name from {{scope.varName}}
		varName := match[2 : len(match)-2] // Remove {{ and }}

		// Look up the variable
		if value, ok := vars[varName]; ok {
			return value
		}
		// Variable not found, return original placeholder
		return match
	})

	return result, nil
}

// MustRender is like Render but panics on error
// For now, Render never returns an error, but this provides a convenient API
func (r *Renderer) MustRender(template string, vars map[string]string) string {
	result, err := r.Render(template, vars)
	if err != nil {
		panic(err)
	}
	return result
}

// RenderSimple is a convenience function for quick rendering without creating a Renderer
func RenderSimple(template string, vars map[string]string) string {
	r := NewRenderer()
	result, _ := r.Render(template, vars)
	return result
}

// ExtractVariables returns all variable names found in the template
func (r *Renderer) ExtractVariables(template string) []string {
	matches := r.varPattern.FindAllStringSubmatch(template, -1)
	varMap := make(map[string]struct{})
	for _, match := range matches {
		if len(match) > 1 {
			varMap[match[1]] = struct{}{}
		}
	}

	result := make([]string, 0, len(varMap))
	for v := range varMap {
		result = append(result, v)
	}
	return result
}

// HasVariables checks if template contains any placeholders
func (r *Renderer) HasVariables(template string) bool {
	return r.varPattern.MatchString(template)
}

// CountVariables counts how many unique variables are in the template
func (r *Renderer) CountVariables(template string) int {
	return len(r.ExtractVariables(template))
}
