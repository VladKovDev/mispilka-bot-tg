package template

import (
	"testing"
	"time"
)

func TestVariableProvider_GetVariables(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		ctx  *RenderContext
		want map[string]string
	}{
		{
			name: "user context only",
			ctx: &RenderContext{
				UserName:  "alice",
				FirstName: "Alice",
				LastName:  "Smith",
			},
			want: map[string]string{
				"user_name":  "alice",
				"first_name": "Alice",
				"last_name":  "Smith",
			},
		},
		{
			name: "user context with payment link",
			ctx: &RenderContext{
				UserName:    "bob",
				PaymentLink: "https://pay.example.com/abc123",
			},
			want: map[string]string{
				"user_name":    "bob",
				"payment_link": "https://pay.example.com/abc123",
			},
		},
		{
			name: "user context with invite link",
			ctx: &RenderContext{
				UserName:   "charlie",
				InviteLink: "https://t.me/+AbCdEfGhIjKlMnOp",
			},
			want: map[string]string{
				"user_name":   "charlie",
				"invite_link": "https://t.me/+AbCdEfGhIjKlMnOp",
			},
		},
		{
			name: "full context with scenario",
			ctx: &RenderContext{
				UserName:        "david",
				FirstName:       "David",
				ScenarioName:    "Course 101",
				PaymentLink:     "https://pay.example.com/xyz",
				InviteLink:      "https://t.me/+Invite",
				ProductName:     "Premium Course",
				PrivateGroupID:  "-1001234567890",
			},
			want: map[string]string{
				"user_name":        "david",
				"first_name":       "David",
				"scenario_name":    "Course 101",
				"payment_link":     "https://pay.example.com/xyz",
				"invite_link":      "https://t.me/+Invite",
				"product_name":     "Premium Course",
				"private_group_id": "-1001234567890",
			},
		},
		{
			name: "context with payment date",
			ctx: &RenderContext{
				UserName:    "eve",
				PaymentDate: &now,
			},
			want: map[string]string{
				"user_name":     "eve",
				"payment_date":  now.Format("2006-01-02"),
			},
		},
		{
			name: "empty context",
			ctx:  &RenderContext{},
			want: map[string]string{},
		},
		{
			name: "nil context",
			ctx:  nil,
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewVariableProvider()
			got := p.GetVariables(tt.ctx)
			assertMapsEqual(t, got, tt.want)
		})
	}
}

func TestVariableProvider_WithCustomVariables(t *testing.T) {
	p := NewVariableProvider()
	ctx := &RenderContext{
		UserName: "alice",
	}

	custom := map[string]string{
		"custom_var": "custom_value",
		"course":     "Go Programming",
	}

	vars := p.GetVariables(ctx)
	for k, v := range custom {
		vars[k] = v
	}

	expected := map[string]string{
		"user_name":   "alice",
		"custom_var":  "custom_value",
		"course":      "Go Programming",
	}
	assertMapsEqual(t, vars, expected)
}

func assertMapsEqual(t *testing.T, got, want map[string]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("got %d variables, want %d", len(got), len(want))
	}

	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Errorf("missing variable: %s", k)
			continue
		}
		if gotV != wantV {
			t.Errorf("variable %s: got %q, want %q", k, gotV, wantV)
		}
	}

	// Check for extra variables in got
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("extra variable: %s = %q", k, got[k])
		}
	}
}
