package template

import (
	"time"
)

// RenderContext contains all data available for template rendering
type RenderContext struct {
	// User information
	UserName  string
	FirstName string
	LastName  string

	// Scenario information
	ScenarioName string
	ScenarioID   string

	// Payment information
	PaymentDate  *time.Time
	PaymentLink  string
	ProductName  string

	// Invite information
	InviteLink     string
	PrivateGroupID string

	// Custom variables can be added by merging
	CustomVars map[string]string
}

// VariableProvider provides template variables from context
type VariableProvider struct {
	// Could add caching or other features here
}

// NewVariableProvider creates a new variable provider
func NewVariableProvider() *VariableProvider {
	return &VariableProvider{}
}

// GetVariables extracts all available variables from the render context
// Returns a map that can be passed directly to Renderer.Render()
func (p *VariableProvider) GetVariables(ctx *RenderContext) map[string]string {
	vars := make(map[string]string)

	if ctx == nil {
		return vars
	}

	// User information
	if ctx.UserName != "" {
		vars["user_name"] = ctx.UserName
	}
	if ctx.FirstName != "" {
		vars["first_name"] = ctx.FirstName
	}
	if ctx.LastName != "" {
		vars["last_name"] = ctx.LastName
	}

	// Scenario information
	if ctx.ScenarioName != "" {
		vars["scenario_name"] = ctx.ScenarioName
	}
	if ctx.ScenarioID != "" {
		vars["scenario_id"] = ctx.ScenarioID
	}

	// Payment information
	if ctx.PaymentDate != nil {
		vars["payment_date"] = ctx.PaymentDate.Format("2006-01-02")
	}
	if ctx.PaymentLink != "" {
		vars["payment_link"] = ctx.PaymentLink
	}
	if ctx.ProductName != "" {
		vars["product_name"] = ctx.ProductName
	}

	// Invite information
	if ctx.InviteLink != "" {
		vars["invite_link"] = ctx.InviteLink
	}
	if ctx.PrivateGroupID != "" {
		vars["private_group_id"] = ctx.PrivateGroupID
	}

	// Custom variables
	for k, v := range ctx.CustomVars {
		vars[k] = v
	}

	return vars
}

// GetVariablesForUser creates a render context from user data and returns variables
// This is a convenience method for common user-based rendering
func (p *VariableProvider) GetVariablesForUser(userName, firstName, lastName string) map[string]string {
	ctx := &RenderContext{
		UserName:  userName,
		FirstName: firstName,
		LastName:  lastName,
	}
	return p.GetVariables(ctx)
}

// GetVariablesForScenario creates variables for scenario-based rendering
func (p *VariableProvider) GetVariablesForScenario(userName, scenarioName string) map[string]string {
	ctx := &RenderContext{
		UserName:     userName,
		ScenarioName: scenarioName,
	}
	return p.GetVariables(ctx)
}
